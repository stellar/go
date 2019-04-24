package io

import (
	"fmt"
	"io"
	"log"
	"regexp"

	"github.com/stellar/go/support/historyarchive"
	"github.com/stellar/go/xdr"
)

// MemoryStateReader is the in-memory representation of HistoryArchiveState
type MemoryStateReader struct {
	has             *historyarchive.HistoryArchiveState
	archive         *historyarchive.Archive
	sequence        uint32
	bucketHashRegex *regexp.Regexp
}

// enforce MemoryStateReader to implement StateReader
var _ StateReader = &MemoryStateReader{}

// MakeMemoryStateReader is a factory method for MemoryStateReader
func MakeMemoryStateReader(archive *historyarchive.Archive, sequence uint32) (*MemoryStateReader, error) {
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

// GetSequence placeholder
func (msr *MemoryStateReader) GetSequence() uint32 {
	return msr.sequence
}

// Read placeholder
func (msr *MemoryStateReader) Read() (bool, xdr.LedgerEntry, error) {
	leChan := make(chan xdr.LedgerEntry)
	for _, hash := range msr.has.Buckets() {
		if !msr.archive.BucketExists(hash) {
			return false, xdr.LedgerEntry{}, fmt.Errorf("bucket hash does not exist: %s", hash)
		}

		sChan, eChan := msr.archive.ListBucket(historyarchive.HashPrefix(hash))
		go func() {
			s := <-sChan
			// example: prd/core-testnet/core_testnet_001/bucket/be/3c/bf/bucket-be3cbfc2d7e4272c01a1a22084573a04dad96bf77aa7fc2be4ce2dec8777b4f9.xdr.gz
			log.Printf("filename: '%s'", s)

			bucketPath, e := getBucketPath(msr.bucketHashRegex, s)
			if e != nil {
				log.Fatalf("cannot get bucket path for file '%s': %s", s, e)
				// return false, xdr.LedgerEntry{}, log.Fatalf("cannot get bucket path for file '%s': %s", s, e)
			}

			rdr, e := msr.archive.GetXdrStream(bucketPath)
			if e != nil {
				log.Fatalf("cannot get xdr stream for file '%s': %s", s, e)
				// return false, xdr.LedgerEntry{}, fmt.Errorf("cannot get xdr stream: %s", e)
			}
			defer rdr.Close()

			n := 0
			for {
				var entry xdr.BucketEntry
				if e = rdr.ReadOne(&entry); e != nil {
					if e == io.EOF {
						break
					}
					log.Fatalf("Error on XDR record %d of %s: %s", n, s, e)
					// return false, xdr.LedgerEntry{}, fmt.Errorf("Error on XDR record %d of %s: %s", n, s, e)
				}
				n++

				le, ok := entry.GetLiveEntry()
				if !ok {
					log.Fatalf("cannot get live entry, n = %d, filename = %s", n, s)
					// return false, xdr.LedgerEntry{}, fmt.Errorf("cannot get live entry, n = %d, filename = %s", n, s)
				}

				leChan <- le
			}
		}()

		go func() {
			e := <-eChan
			if e != nil {
				log.Printf("%s", e.Error())
				panic(e)
			} else {
				log.Printf("nil error\n")
			}
		}()

		// break here for now so we only do first run
		break
	}

	le := <-leChan

	return true, le, nil
}
