// Copyright 2021 Stellar Development Foundation and contributors. Licensed
// under the Apache License, Version 2.0. See the COPYING file at the root
// of this distribution or at http://www.apache.org/licenses/LICENSE-2.0

package historyarchive

import (
	"math/rand"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// A PooledArchive is just a collection of `ArchiveInterface`s so that we can
// distribute requests fairly throughout the pool.
type PooledArchive []ArchiveInterface

// CreatePool tries connecting to each of the provided history archive URLs,
// returning a pool of valid archives.
//
// If none of the archives work, this returns the error message of the last
// failed archive. Note that the errors for each individual archive are hard to
// track if there's success overall.
//
// Possible FIXME for the above limitation: return []error instead? but then
// users need to check `len(pool) > 0` instead of `err == nil`.
func CreatePool(archiveURLs []string, config ConnectOptions) (PooledArchive, error) {
	if len(archiveURLs) <= 0 {
		return nil, errors.New("No history archives provided")
	}

	var lastErr error = nil

	// Try connecting to all of the listed archives, but only store valid ones.
	var validArchives PooledArchive
	for _, url := range archiveURLs {
		archive, err := Connect(
			url,
			ConnectOptions{
				NetworkPassphrase:   config.NetworkPassphrase,
				CheckpointFrequency: config.CheckpointFrequency,
				Context:             config.Context,
			},
		)

		if err != nil {
			lastErr = errors.Wrapf(err, "Error connecting to history archive (%s)", url)
			continue
		}

		validArchives = append(validArchives, archive)
	}

	if len(validArchives) == 0 {
		return nil, lastErr
	}

	return validArchives, nil
}

// Ensure the pool conforms to the ArchiveInterface
var _ ArchiveInterface = PooledArchive{}

// Below are the ArchiveInterface method implementations.

func (pa PooledArchive) GetAnyArchive() ArchiveInterface {
	return pa[rand.Intn(len(pa))]
}

func (pa PooledArchive) GetPathHAS(path string) (HistoryArchiveState, error) {
	return pa.GetAnyArchive().GetPathHAS(path)
}

func (pa PooledArchive) PutPathHAS(path string, has HistoryArchiveState, opts *CommandOptions) error {
	return pa.GetAnyArchive().PutPathHAS(path, has, opts)
}

func (pa PooledArchive) BucketExists(bucket Hash) (bool, error) {
	return pa.GetAnyArchive().BucketExists(bucket)
}

func (pa PooledArchive) CategoryCheckpointExists(cat string, chk uint32) (bool, error) {
	return pa.GetAnyArchive().CategoryCheckpointExists(cat, chk)
}

func (pa PooledArchive) GetLedgerHeader(chk uint32) (xdr.LedgerHeaderHistoryEntry, error) {
	return pa.GetAnyArchive().GetLedgerHeader(chk)
}

func (pa PooledArchive) GetRootHAS() (HistoryArchiveState, error) {
	return pa.GetAnyArchive().GetRootHAS()
}

func (pa PooledArchive) GetLedgers(start, end uint32) (map[uint32]*Ledger, error) {
	return pa.GetAnyArchive().GetLedgers(start, end)
}

func (pa PooledArchive) GetCheckpointHAS(chk uint32) (HistoryArchiveState, error) {
	return pa.GetAnyArchive().GetCheckpointHAS(chk)
}

func (pa PooledArchive) PutCheckpointHAS(chk uint32, has HistoryArchiveState, opts *CommandOptions) error {
	return pa.GetAnyArchive().PutCheckpointHAS(chk, has, opts)
}

func (pa PooledArchive) PutRootHAS(has HistoryArchiveState, opts *CommandOptions) error {
	return pa.GetAnyArchive().PutRootHAS(has, opts)
}

func (pa PooledArchive) ListBucket(dp DirPrefix) (chan string, chan error) {
	return pa.GetAnyArchive().ListBucket(dp)
}

func (pa PooledArchive) ListAllBuckets() (chan string, chan error) {
	return pa.GetAnyArchive().ListAllBuckets()
}

func (pa PooledArchive) ListAllBucketHashes() (chan Hash, chan error) {
	return pa.GetAnyArchive().ListAllBucketHashes()
}

func (pa PooledArchive) ListCategoryCheckpoints(cat string, pth string) (chan uint32, chan error) {
	return pa.GetAnyArchive().ListCategoryCheckpoints(cat, pth)
}

func (pa PooledArchive) GetXdrStreamForHash(hash Hash) (*XdrStream, error) {
	return pa.GetAnyArchive().GetXdrStreamForHash(hash)
}

func (pa PooledArchive) GetXdrStream(pth string) (*XdrStream, error) {
	return pa.GetAnyArchive().GetXdrStream(pth)
}

func (pa PooledArchive) GetCheckpointManager() CheckpointManager {
	return pa.GetAnyArchive().GetCheckpointManager()
}
