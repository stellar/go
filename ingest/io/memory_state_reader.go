package io

import (
	"fmt"
	"io"
	"regexp"

	"github.com/stellar/go/support/historyarchive"
	"github.com/stellar/go/xdr"
)

// readResult is the result of reading a bucket value
type readResult struct {
	entry xdr.LedgerEntry
	e     error
}

// MemoryStateReader is the in-memory representation of HistoryArchiveState
type MemoryStateReader struct {
	has             *historyarchive.HistoryArchiveState
	archive         *historyarchive.Archive
	sequence        uint32
	bucketHashRegex *regexp.Regexp
	active          bool
	readChan        chan readResult
}

// enforce MemoryStateReader to implement StateReader
var _ StateReader = &MemoryStateReader{}

// MakeMemoryStateReader is a factory method for MemoryStateReader
func MakeMemoryStateReader(archive *historyarchive.Archive, sequence uint32, bufferSize uint16) (*MemoryStateReader, error) {
	has, e := archive.GetCheckpointHAS(sequence)
	if e != nil {
		return nil, fmt.Errorf("unable to get checkpoint HAS at ledger sequence %d: %s", sequence, e)
	}

	bhr, e := makeRegex()
	if e != nil {
		return nil, fmt.Errorf("unable to compile regexp: %s", e)
	}

	return &MemoryStateReader{
		has:             &has,
		archive:         archive,
		sequence:        sequence,
		bucketHashRegex: bhr,
		active:          false,
		readChan:        make(chan readResult, bufferSize),
	}, nil
}

func makeRegex() (*regexp.Regexp, error) {
	return regexp.Compile(`(bucket/[0-9a-z]{2}/[0-9a-z]{2}/[0-9a-z]{2}/bucket-[0-9a-z]+\.xdr\.gz)`)
}

func getBucketPath(r *regexp.Regexp, s string) (string, error) {
	matches := r.FindStringSubmatch(s)
	if len(matches) != 2 {
		return "", fmt.Errorf("regex string submatch needs full match and one more subgroup, i.e. length should be 2 but was %d", len(matches))
	}
	return matches[1], nil
}

// BufferReads triggers the streaming logic needed to be done before Read() can actually produce a result
func (msr *MemoryStateReader) BufferReads() {
	msr.active = true
	go msr.bufferNext()
}

func (msr *MemoryStateReader) bufferNext() {
	for _, hash := range msr.has.Buckets() {
		if !msr.archive.BucketExists(hash) {
			msr.readChan <- readResult{xdr.LedgerEntry{}, fmt.Errorf("bucket hash does not exist: %s", hash)}
			close(msr.readChan)
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
			close(msr.readChan)
			return
		}
		shouldContinue := msr.streamBucketContents(filepath, hash)
		if !shouldContinue {
			close(msr.readChan)
			return
		}
	}

	close(msr.readChan)
	return
}

// streamBucketContents pushes value onto the read channel, returning false when the channel needs to be closed otherwise true
func (msr *MemoryStateReader) streamBucketContents(filepath string, hash historyarchive.Hash) bool {
	bucketPath, e := getBucketPath(msr.bucketHashRegex, filepath)
	if e != nil {
		msr.readChan <- readResult{xdr.LedgerEntry{}, fmt.Errorf("cannot get bucket path for filepath '%s' with hash '%s': %s", filepath, hash, e)}
		return false
	}

	rdr, e := msr.archive.GetXdrStream(bucketPath)
	if e != nil {
		msr.readChan <- readResult{xdr.LedgerEntry{}, fmt.Errorf("cannot get xdr stream for file '%s': %s", filepath, e)}
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
			msr.readChan <- readResult{xdr.LedgerEntry{}, fmt.Errorf("Error on XDR record %d of filepath '%s': %s", n, filepath, e)}
			return false
		}

		liveEntry, ok := entry.GetLiveEntry()
		if ok {
			// since readChan is a buffered channel we block here until one item is consumed on the dequeue side.
			// this is our intended behavior, which ensures we only buffer exactly bufferSize results in the channel.
			msr.readChan <- readResult{liveEntry, nil}
		}
		// TODO should we do something if we don't have a live entry?

		n++
	}
}

// GetSequence impl.
func (msr *MemoryStateReader) GetSequence() uint32 {
	return msr.sequence
}

// Read returns a new ledger entry on each call, returning false when the stream ends
func (msr *MemoryStateReader) Read() (bool, xdr.LedgerEntry, error) {
	if !msr.active {
		return false, xdr.LedgerEntry{}, fmt.Errorf("memory state reader not active, need to call BufferReads() before calling Read()")
	}

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
