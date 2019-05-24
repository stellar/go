package io

import (
	"encoding/base64"
	"fmt"
	"io"
	"sync"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
	"github.com/stellar/go/xdr"
)

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

func hashToBucketPath(hash historyarchive.Hash) string {
	return fmt.Sprintf(
		"bucket/%s/bucket-%s.xdr.gz",
		historyarchive.HashPrefix(hash).Path(),
		hash.String(),
	)
}

func (msr *MemoryStateReader) bufferNext() {
	defer close(msr.readChan)

	seen := map[string]xdr.LedgerEntry{}

	// Process buckets from oldest to newest
	var buckets []string
	for i := len(msr.has.CurrentBuckets) - 1; i >= 0; i-- {
		b := msr.has.CurrentBuckets[i]
		buckets = append(buckets, b.Snap, b.Curr)
	}

	for _, hashString := range buckets {
		hash, err := historyarchive.DecodeHash(hashString)
		if err != nil {
			msr.readChan <- readResult{xdr.LedgerEntry{}, errors.Wrap(err, "Error decoding bucket hash")}
			return
		}

		if hash.IsZero() {
			continue
		}

		if !msr.archive.BucketExists(hash) {
			msr.readChan <- readResult{xdr.LedgerEntry{}, fmt.Errorf("bucket hash does not exist: %s", hash)}
			return
		}

		bucketPath := hashToBucketPath(hash)
		var shouldContinue bool
		shouldContinue = msr.streamBucketContents(bucketPath, seen)
		if !shouldContinue {
			break
		}
	}

	// Send map contents

	// since readChan is a buffered channel we block here until one item is consumed on the dequeue side.
	// this is our intended behavior, which ensures we only buffer exactly bufferSize results in the channel.
	for _, entry := range seen {
		msr.readChan <- readResult{entry, nil}
	}
}

// streamBucketContents pushes value onto the read channel, returning false when the channel needs to be closed otherwise true
func (msr *MemoryStateReader) streamBucketContents(
	bucketPath string,
	seen map[string]xdr.LedgerEntry,
) bool {
	rdr, e := msr.archive.GetXdrStream(bucketPath)
	if e != nil {
		msr.readChan <- readResult{xdr.LedgerEntry{}, fmt.Errorf("cannot get xdr stream for bucketPath '%s': %s", bucketPath, e)}
		return false
	}
	defer rdr.Close()

	n := -1
	for {
		n++

		var entry xdr.BucketEntry
		if e = rdr.ReadOne(&entry); e != nil {
			if e == io.EOF {
				// proceed to the next bucket hash
				return true
			}
			msr.readChan <- readResult{xdr.LedgerEntry{}, fmt.Errorf("Error on XDR record %d of bucketPath '%s': %s", n, bucketPath, e)}
			return false
		}

		var key xdr.LedgerKey

		switch entry.Type {
		case xdr.BucketEntryTypeLiveentry:
			liveEntry := entry.MustLiveEntry()
			key = liveEntry.LedgerKey()
		case xdr.BucketEntryTypeDeadentry:
			key = entry.MustDeadEntry()
		case xdr.BucketEntryTypeMetaentry:
			// Ignore
		default:
			panic(fmt.Sprintf("Shouldn't happen in protocol <=10: BucketEntryType=%d", entry.Type))
		}

		// Process accounts only for now
		if key.Type != xdr.LedgerEntryTypeAccount {
			continue
		}

		keyBytes, e := key.MarshalBinary()
		if e != nil {
			msr.readChan <- readResult{xdr.LedgerEntry{}, fmt.Errorf("Error marshaling XDR record %d of bucketPath '%s': %s", n, bucketPath, e)}
			return false
		}

		h := base64.StdEncoding.EncodeToString(keyBytes)

		switch entry.Type {
		case xdr.BucketEntryTypeLiveentry:
			seen[h] = entry.MustLiveEntry()
		case xdr.BucketEntryTypeDeadentry:
			delete(seen, h)
		}
	}

	panic("Shouldn't happen")
}

// GetSequence impl.
func (msr *MemoryStateReader) GetSequence() uint32 {
	return msr.sequence
}

// Read returns a new ledger entry on each call, returning false when the stream ends
func (msr *MemoryStateReader) Read() (xdr.LedgerEntry, error) {
	msr.once.Do(func() {
		go msr.bufferNext()
	})

	// blocking call. anytime we consume from this channel, the background goroutine will stream in the next value
	result, ok := <-msr.readChan
	if !ok {
		// when channel is closed then return false with empty values
		return xdr.LedgerEntry{}, EOF
	}

	if result.e != nil {
		return xdr.LedgerEntry{}, fmt.Errorf("error while reading from background channel: %s", result.e)
	}
	return result.entry, nil
}
