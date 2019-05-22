package io

import (
	"crypto/sha256"
	"fmt"
	"io"
	"regexp"
	"sync"

	"github.com/stellar/go/support/historyarchive"
	"github.com/stellar/go/xdr"
)

var bucketRegex = regexp.MustCompile(`(bucket/[0-9a-z]{2}/[0-9a-z]{2}/[0-9a-z]{2}/bucket-[0-9a-z]+\.xdr\.gz)`)

// readResult is the result of reading a bucket value
type readResult struct {
	entry xdr.LedgerEntry
	e     error
}

// MemoryStateReader is an in-memory streaming implementation that reads HistoryArchiveState
type MemoryStateReader struct {
	has      *historyarchive.HistoryArchiveState
	archive  *historyarchive.Archive
	sequence uint32
	active   bool
	readChan chan readResult
	once     *sync.Once
}

// enforce MemoryStateReader to implement StateReader
var _ StateReader = &MemoryStateReader{}

// MakeMemoryStateReader is a factory method for MemoryStateReader
func MakeMemoryStateReader(archive *historyarchive.Archive, sequence uint32, bufferSize uint16) (*MemoryStateReader, error) {
	has, e := archive.GetCheckpointHAS(sequence)
	if e != nil {
		return nil, fmt.Errorf("unable to get checkpoint HAS at ledger sequence %d: %s", sequence, e)
	}

	return &MemoryStateReader{
		has:      &has,
		archive:  archive,
		sequence: sequence,
		active:   false,
		readChan: make(chan readResult, bufferSize),
		once:     &sync.Once{},
	}, nil
}

func getBucketPath(r *regexp.Regexp, s string) (string, error) {
	matches := r.FindStringSubmatch(s)
	if len(matches) != 2 {
		return "", fmt.Errorf("regex string submatch needs full match and one more subgroup, i.e. length should be 2 but was %d", len(matches))
	}
	return matches[1], nil
}

func (msr *MemoryStateReader) bufferNext() {
	defer close(msr.readChan)

	// iterate from newest to oldest bucket and track keys already seen
	seen := map[string]bool{}
	for _, hash := range msr.has.Buckets() {
		if !msr.archive.BucketExists(hash) {
			msr.readChan <- readResult{xdr.LedgerEntry{}, fmt.Errorf("bucket hash does not exist: %s", hash)}
			return
		}

		// read bucket detail
		filepathChan, errChan := msr.archive.ListBucket(historyarchive.HashPrefix(hash))

		// read from channels
		var filepath string
		var e error
		var ok bool
		select {
		case fp, okb := <-filepathChan:
			// example filepath: prd/core-testnet/core_testnet_001/bucket/be/3c/bf/bucket-be3cbfc2d7e4272c01a1a22084573a04dad96bf77aa7fc2be4ce2dec8777b4f9.xdr.gz
			filepath, e, ok = fp, nil, okb
		case err, okb := <-errChan:
			filepath, e, ok = "", err, okb
			// TODO do we need to do anything special if e is nil here?
		}
		if !ok {
			// move on to next bucket when this bucket is fully consumed or empty
			continue
		}

		// process values
		if e != nil {
			msr.readChan <- readResult{xdr.LedgerEntry{}, fmt.Errorf("received error on errChan when listing buckets for hash '%s': %s", hash, e)}
			return
		}

		bucketPath, e := getBucketPath(bucketRegex, filepath)
		if e != nil {
			msr.readChan <- readResult{xdr.LedgerEntry{}, fmt.Errorf("cannot get bucket path for filepath '%s' with hash '%s': %s", filepath, hash, e)}
			return
		}

		var shouldContinue bool
		shouldContinue = msr.streamBucketContents(bucketPath, hash, seen)
		if !shouldContinue {
			return
		}
	}
}

// streamBucketContents pushes value onto the read channel, returning false when the channel needs to be closed otherwise true
func (msr *MemoryStateReader) streamBucketContents(
	bucketPath string,
	hash historyarchive.Hash,
	seen map[string]bool,
) bool {
	rdr, e := msr.archive.GetXdrStream(bucketPath)
	if e != nil {
		msr.readChan <- readResult{xdr.LedgerEntry{}, fmt.Errorf("cannot get xdr stream for bucketPath '%s': %s", bucketPath, e)}
		return false
	}
	defer rdr.Close()

	n := 0
	for {
		var entry xdr.BucketEntry
		if e = rdr.ReadOne(&entry); e != nil {
			if e == io.EOF {
				// proceed to the next bucket hash
				return true
			}
			msr.readChan <- readResult{xdr.LedgerEntry{}, fmt.Errorf("Error on XDR record %d of bucketPath '%s': %s", n, bucketPath, e)}
			return false
		}

		liveEntry, ok := entry.GetLiveEntry()
		if ok {
			// ignore entry if we've seen it previously
			key := liveEntry.LedgerKey()
			keyBytes, e := key.MarshalBinary()
			if e != nil {
				msr.readChan <- readResult{xdr.LedgerEntry{}, fmt.Errorf("Error marshaling XDR record %d of bucketPath '%s': %s", n, bucketPath, e)}
				return false
			}
			shasum := fmt.Sprintf("%x", sha256.Sum256(keyBytes))

			if seen[shasum] {
				n++
				continue
			}
			seen[shasum] = true

			// since readChan is a buffered channel we block here until one item is consumed on the dequeue side.
			// this is our intended behavior, which ensures we only buffer exactly bufferSize results in the channel.
			msr.readChan <- readResult{liveEntry, nil}
		}
		// we can ignore dead entries because we're only ever concerned with the first live entry values
		n++
	}
}

// GetSequence impl.
func (msr *MemoryStateReader) GetSequence() uint32 {
	return msr.sequence
}

// Read returns a new ledger entry on each call, returning false when the stream ends
func (msr *MemoryStateReader) Read() (bool, xdr.LedgerEntry, error) {
	msr.once.Do(func() {
		go msr.bufferNext()
	})

	// blocking call. anytime we consume from this channel, the background goroutine will stream in the next value
	result, ok := <-msr.readChan
	if !ok {
		// when channel is closed then return false with empty values
		return false, xdr.LedgerEntry{}, nil
	}

	if result.e != nil {
		return true, xdr.LedgerEntry{}, fmt.Errorf("error while reading from background channel: %s", result.e)
	}
	return true, result.entry, nil
}
