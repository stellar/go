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

// MemoryStateReader is an in-memory streaming implementation that reads ledger entries
// from buckets for a given HistoryArchiveState.
// MemoryStateReader hides internal structure of buckets from the user so entries returned
// by `Read()` are exactly the ledger entries present at the given ledger.
type MemoryStateReader struct {
	has        *historyarchive.HistoryArchiveState
	archive    *historyarchive.Archive
	sequence   uint32
	readChan   chan readResult
	streamOnce sync.Once
	closeOnce  sync.Once
	done       chan bool
}

// enforce MemoryStateReader to implement StateReadCloser
var _ StateReadCloser = &MemoryStateReader{}

// MakeMemoryStateReader is a factory method for MemoryStateReader
func MakeMemoryStateReader(archive *historyarchive.Archive, sequence uint32, bufferSize uint16) (*MemoryStateReader, error) {
	has, e := archive.GetCheckpointHAS(sequence)
	if e != nil {
		return nil, fmt.Errorf("unable to get checkpoint HAS at ledger sequence %d: %s", sequence, e)
	}

	return &MemoryStateReader{
		has:        &has,
		archive:    archive,
		sequence:   sequence,
		readChan:   make(chan readResult, bufferSize),
		streamOnce: sync.Once{},
		closeOnce:  sync.Once{},
		done:       make(chan bool),
	}, nil
}

// streamBuckets is internal method that streams buckets from the given HAS.
//
// Buckets should be processed from oldest to newest, `snap` and then `curr` at
// each level. The correct value of ledger entry is the latest seen `LIVEENTRY`
// except the case when there's a `DEADENTRY` later which removes the entry.
//
// We can implement trivial algorithm (processing from oldest to newest buckets)
// but it requires to keep map of all entries in memory and stream what's left
// when all buckets are processed.
//
// However, we can modify this algorithm to work from newest to oldest ledgers:
//
//   1. For each `LIVEENTRY` we check if we've seen it before (`seen` map) or
//      if we've seen `DEADENTRY` for it (`removed` map). If both conditions are
//      false, we write that bucket entry to the stream and mark it as `seen`.
//   2. For each `DEADENTRY` we keep track of removed bucket entries in
//      `removed` map.
//
// In such algorithm we just need to keep 2 maps with `bool` values that require
// much less memory space.  The memory requirements will be lowered when CAP-0020
// is live. Finally, we can require `ingest/pipeline.StateProcessor` to return
// entry types it needs so that `MemoryStateReader` will only stream entry types
// required by a given pipeline.
func (msr *MemoryStateReader) streamBuckets() {
	defer close(msr.readChan)
	defer msr.closeOnce.Do(msr.close)

	removed := map[string]bool{}
	seen := map[string]bool{}

	var buckets []string
	for i := 0; i < len(msr.has.CurrentBuckets); i++ {
		b := msr.has.CurrentBuckets[i]
		buckets = append(buckets, b.Curr, b.Snap)
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

		var shouldContinue bool
		shouldContinue = msr.streamBucketContents(hash, seen, removed)
		if !shouldContinue {
			break
		}
	}
}

// streamBucketContents pushes value onto the read channel, returning false when the channel needs to be closed otherwise true
func (msr *MemoryStateReader) streamBucketContents(
	hash historyarchive.Hash,
	seen map[string]bool,
	removed map[string]bool,
) bool {
	rdr, e := msr.archive.GetXdrStreamForHash(hash)
	if e != nil {
		msr.readChan <- readResult{xdr.LedgerEntry{}, fmt.Errorf("cannot get xdr stream for hash '%s': %s", hash.String(), e)}
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
			msr.readChan <- readResult{xdr.LedgerEntry{}, fmt.Errorf("Error on XDR record %d of hash '%s': %s", n, hash.String(), e)}
			return false
		}

		var key xdr.LedgerKey

		switch entry.Type {
		case xdr.BucketEntryTypeLiveentry:
			liveEntry := entry.MustLiveEntry()
			key = liveEntry.LedgerKey()
		case xdr.BucketEntryTypeDeadentry:
			key = entry.MustDeadEntry()
		default:
			panic(fmt.Sprintf("Shouldn't happen in protocol <=10: BucketEntryType=%d", entry.Type))
		}

		keyBytes, e := key.MarshalBinary()
		if e != nil {
			msr.readChan <- readResult{xdr.LedgerEntry{}, fmt.Errorf("Error marshaling XDR record %d of hash '%s': %s", n, hash.String(), e)}
			return false
		}

		h := base64.StdEncoding.EncodeToString(keyBytes)

		switch entry.Type {
		case xdr.BucketEntryTypeLiveentry:
			if !seen[h] && !removed[h] {
				msr.readChan <- readResult{entry.MustLiveEntry(), nil}
				seen[h] = true
			}
		case xdr.BucketEntryTypeDeadentry:
			removed[h] = true
		}

		select {
		case <-msr.done:
			// Close() called: stop processing buckets.
			return false
		default:
			continue
		}
	}

	panic("Shouldn't happen")
}

// GetSequence impl.
func (msr *MemoryStateReader) GetSequence() uint32 {
	return msr.sequence
}

// Read returns a new ledger entry on each call, returning io.EOF when the stream ends.
func (msr *MemoryStateReader) Read() (xdr.LedgerEntry, error) {
	msr.streamOnce.Do(func() {
		go msr.streamBuckets()
	})

	// blocking call. anytime we consume from this channel, the background goroutine will stream in the next value
	result, ok := <-msr.readChan
	if !ok {
		// when channel is closed then return io.EOF
		return xdr.LedgerEntry{}, EOF
	}

	if result.e != nil {
		return xdr.LedgerEntry{}, errors.Wrap(result.e, "Error while reading from buckets")
	}
	return result.entry, nil
}

func (msr *MemoryStateReader) close() {
	close(msr.done)
}

// Close should be called when reading is finished.
func (msr *MemoryStateReader) Close() error {
	msr.closeOnce.Do(msr.close)
	return nil
}
