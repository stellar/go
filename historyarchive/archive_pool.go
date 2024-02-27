// Copyright 2021 Stellar Development Foundation and contributors. Licensed
// under the Apache License, Version 2.0. See the COPYING file at the root
// of this distribution or at http://www.apache.org/licenses/LICENSE-2.0

package historyarchive

import (
	"math/rand"

	log "github.com/sirupsen/logrus"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// An ArchivePool is just a collection of `ArchiveInterface`s so that we can
// distribute requests fairly throughout the pool, but with additional
// error-tracking to identify/avoid problematic archives in the pool.
type ArchivePool struct {
	pool []ArchiveInterface

	errors map[ArchiveInterface]*errStats
	curr   int
}

// NewArchivePool tries connecting to each of the provided history archive URLs,
// returning a pool of valid archives.
//
// If none of the archives work, this returns the error message of the last
// failed archive. Note that the errors for each individual archive are hard to
// track if there's success overall.
func NewArchivePool(archiveURLs []string, opts ArchiveOptions) (ArchiveInterface, error) {
	if len(archiveURLs) <= 0 {
		return nil, errors.New("No history archives provided")
	}

	ap := ArchivePool{
		pool:   make([]ArchiveInterface, 0, len(archiveURLs)),
		errors: make(map[ArchiveInterface]*errStats, len(archiveURLs)),
		curr:   0,
	}
	var lastErr error = nil

	// Try connecting to all of the listed archives, but only store valid ones.
	for _, url := range archiveURLs {
		archive, err := Connect(
			url,
			opts,
		)

		if err != nil {
			lastErr = errors.Wrapf(err, "Error connecting to history archive (%s)", url)
			continue
		}

		ap.pool = append(ap.pool, archive)
		ap.errors[archive] = &errStats{}
	}

	if len(ap.pool) == 0 {
		return nil, lastErr
	}

	ap.curr = rand.Intn(len(ap.pool)) // don't necessarily start at zero
	return &ap, nil
}

func (pa *ArchivePool) GetStats() []ArchiveStats {
	stats := []ArchiveStats{}
	for _, archive := range pa.pool {
		stats = append(stats, archive.GetStats()...)
	}
	return stats
}

// Ensure the pool conforms to the ArchiveInterface
var _ ArchiveInterface = &ArchivePool{}

//
// These are helpers to round-robin calls through archives.
//

// getNextIndex is a stateless way to iterate through the pool
func (pa *ArchivePool) getNextIndex(i int) int {
	return (i + 1) % len(pa.pool)
}

// getNextArchive statefully round-robins through the pool
func (pa *ArchivePool) getNextArchive() ArchiveInterface {
	// Round-robin through the archives
	pa.curr = pa.getNextIndex(pa.curr)
	return pa.pool[pa.curr]
}

// runOnEach is a helper method that will run a particular action on every
// archive in the pool until it succeeds or the pool is exhausted (whichever
// comes first). It will also track error stats against the pool to identify
// problematic archives.
func (pa *ArchivePool) runOnEach(runner func(ai ArchiveInterface) error) error {
	// track the first archive we'll use
	start := pa.getNextIndex(pa.curr)

	//
	// The error resilience here is fairly simple: we round-robin through
	// the list of archives in the pool until success or exhaustion.
	//
	for {
		ai := pa.getNextArchive()
		cycle := pa.getNextIndex(pa.curr) == start

		if err := runner(ai); err != nil {
			if statline, ok := pa.errors[ai]; ok /* shouldn't miss, but safer */ {
				// Periodically output accumulated issues
				if errCount := statline.addError(err); errCount%7 == 0 {
					log.WithError(err).Warnf(
						"The archive '%s' has had %d errors so far",
						ai.GetStats()[0].GetBackendName(), errCount)
				}
			}

			// TODO: Should these be handled in a special way? I think no,
			// because it will bubble up the same error after trying all of
			// the archives in the pool, anyway.
			// if err == context.Canceled || err == context.DeadlineExceeded {
			// }

			// If we're cycling around, we're all out of options and should
			// bubble up the last error we saw.
			if cycle {
				return err
			}

			continue
		}

		return nil
	}
}

//
// Below are the ArchiveInterface method implementations.
//

func (pa *ArchivePool) GetPathHAS(path string) (HistoryArchiveState, error) {
	has := HistoryArchiveState{}
	err := pa.runOnEach(func(ai ArchiveInterface) error {
		var innerErr error
		has, innerErr = ai.GetPathHAS(path)
		return innerErr
	})
	return has, err
}

func (pa *ArchivePool) PutPathHAS(path string, has HistoryArchiveState, opts *CommandOptions) error {
	return pa.runOnEach(func(ai ArchiveInterface) error {
		return ai.PutPathHAS(path, has, opts)
	})
}

func (pa *ArchivePool) BucketExists(bucket Hash) (bool, error) {
	status := false
	return status, pa.runOnEach(func(ai ArchiveInterface) error {
		var err error
		status, err = ai.BucketExists(bucket)
		return err
	})
}

func (pa *ArchivePool) BucketSize(bucket Hash) (int64, error) {
	var bsize int64
	return bsize, pa.runOnEach(func(ai ArchiveInterface) error {
		var err error
		bsize, err = ai.BucketSize(bucket)
		return err
	})
}

func (pa *ArchivePool) CategoryCheckpointExists(cat string, chk uint32) (bool, error) {
	return pa.getNextArchive().CategoryCheckpointExists(cat, chk)
}

func (pa *ArchivePool) GetLedgerHeader(chk uint32) (xdr.LedgerHeaderHistoryEntry, error) {
	return pa.getNextArchive().GetLedgerHeader(chk)
}

func (pa *ArchivePool) GetRootHAS() (HistoryArchiveState, error) {
	return pa.getNextArchive().GetRootHAS()
}

func (pa *ArchivePool) GetLedgers(start, end uint32) (map[uint32]*Ledger, error) {
	return pa.getNextArchive().GetLedgers(start, end)
}

func (pa *ArchivePool) GetCheckpointHAS(chk uint32) (HistoryArchiveState, error) {
	return pa.getNextArchive().GetCheckpointHAS(chk)
}

func (pa *ArchivePool) PutCheckpointHAS(chk uint32, has HistoryArchiveState, opts *CommandOptions) error {
	return pa.getNextArchive().PutCheckpointHAS(chk, has, opts)
}

func (pa *ArchivePool) PutRootHAS(has HistoryArchiveState, opts *CommandOptions) error {
	return pa.getNextArchive().PutRootHAS(has, opts)
}

func (pa *ArchivePool) ListBucket(dp DirPrefix) (chan string, chan error) {
	return pa.getNextArchive().ListBucket(dp)
}

func (pa *ArchivePool) ListAllBuckets() (chan string, chan error) {
	return pa.getNextArchive().ListAllBuckets()
}

func (pa *ArchivePool) ListAllBucketHashes() (chan Hash, chan error) {
	return pa.getNextArchive().ListAllBucketHashes()
}

func (pa *ArchivePool) ListCategoryCheckpoints(cat string, pth string) (chan uint32, chan error) {
	return pa.getNextArchive().ListCategoryCheckpoints(cat, pth)
}

func (pa *ArchivePool) GetXdrStreamForHash(hash Hash) (*XdrStream, error) {
	return pa.getNextArchive().GetXdrStreamForHash(hash)
}

func (pa *ArchivePool) GetXdrStream(pth string) (*XdrStream, error) {
	return pa.getNextArchive().GetXdrStream(pth)
}

func (pa *ArchivePool) GetCheckpointManager() CheckpointManager {
	return pa.getNextArchive().GetCheckpointManager()
}

// errStats is a way to track how many errors occurred.
type errStats struct {
	count   int
	lastErr error
}

// addError updates all tracking states
func (statline *errStats) addError(err error) int {
	statline.count++
	statline.lastErr = err
	return statline.count
}
