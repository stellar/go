// Copyright 2021 Stellar Development Foundation and contributors. Licensed
// under the Apache License, Version 2.0. See the COPYING file at the root
// of this distribution or at http://www.apache.org/licenses/LICENSE-2.0

package historyarchive

import (
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

const (
	requestBackoffMs = 250
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
		curr:   rand.Intn(len(archiveURLs)), // don't necessarily start at zero
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

	for { // loops until success or cycle
		ai := pa.getNextArchive()
		cycle := pa.getNextIndex(pa.curr) == start

		//
		// The error-redundancy logic here is admittedly a little
		// over-engineered and I'd appreciate feedback on it:
		//
		// If the archive is "problematic" (i.e. needs backoff), we should skip
		// it until the backoff period is reached, so we find an archive that
		// needs the least amount of back-off.
		//
		// Finally, if we've cycled through the pool and haven't returned still,
		// then this is just a lost cause and we return the latest error.
		//
		// This might put more strain on livelier archives in the pool but it
		// does ensure we eventually retry failing ones. This is a fine
		// trade-off because we're preferring low latency rather than perfect
		// redundancy. It also means if everything in the pool is failing, we
		// back off for the smallest amount of time possible.
		//

		shouldBackoff := pa.errors[ai].getBackoff()
		if shouldBackoff > 0 {
			// If we're going to back off, we should sleep for the lowest amount
			// of time possible.
			bestBackoff, bestPool := shouldBackoff, ai

			// Start our search from the next interface since we *know* this one
			// requires a back-off.
			loopStart := pa.getNextIndex(pa.curr)
			for i := loopStart; pa.getNextIndex(i) != loopStart; i = pa.getNextIndex(i) {
				pool := pa.pool[i]
				errors := pa.errors[pool]
				if backoff := errors.getBackoff(); backoff < bestBackoff {
					bestBackoff = backoff
					bestPool = pool
				}
			}

			// Safe without a check because:
			//
			// > A negative or zero duration causes Sleep to return immediately.
			//
			// https://pkg.go.dev/time#Sleep
			time.Sleep(bestBackoff)
			ai = bestPool
		}

		// Reaching this point means either no back-off or `ai` was backed-off
		// from, so we can actually do execution now.
		if err := runner(ai); err != nil {
			if statline, ok := pa.errors[ai]; ok {
				statline.backoffs++ // increase backoff duration

				// Periodically output accumulated delays
				if errCount := statline.addError(err); errCount%7 == 0 {
					log.WithError(err).Warnf(
						"The archive '%s' has had %d errors so far",
						ai.GetStats()[0].GetBackendName(), errCount)
				}
			}

			// If we're cycling around, we're all out of options and should
			// bubble up the last error we saw.
			if cycle {
				return err
			}
			continue
		}

		// Always reset backoff counter in non-error case.
		pa.errors[ai].backoffs = 0
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

// errStats is a way to track when, how many, and the latest error occurred.
type errStats struct {
	count   int
	latest  time.Time
	lastErr error

	backoffs int // tracked by caller: how many errors since last success?
}

// addError updates all tracking states
func (statline *errStats) addError(err error) int {
	statline.count++
	statline.lastErr = err
	statline.latest = time.Now()
	return statline.count
}

// getBackoff suggests a linear backoff (+250ms each step) from the time of
// the last error
func (statline *errStats) getBackoff() time.Duration {
	if statline.backoffs == 0 {
		return time.Duration(0)
	}

	// Given the time of the last error, when would it be okay to request again?
	backoffUntil := statline.latest.Add(
		time.Duration(statline.backoffs) * requestBackoffMs * time.Millisecond,
	)

	// How long is it until then? If it's in the past, you can fire away.
	backoffFor := -time.Since(backoffUntil)
	if backoffFor > 0 {
		return backoffFor
	}
	return time.Duration(0)
}
