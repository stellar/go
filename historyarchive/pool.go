// Copyright 2016 Stellar Development Foundation and contributors. Licensed
// under the Apache License, Version 2.0. See the COPYING file at the root
// of this distribution or at http://www.apache.org/licenses/LICENSE-2.0

package historyarchive

import (
	"math/rand"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// Type PooledArchive forwards all API calls to a random ArchiveInterface within
// its internal pool.
type PooledArchive struct {
	pool []ArchiveInterface
}

var _ ArchiveInterface = &PooledArchive{}

func CreatePool(archives ...ArchiveInterface) (*PooledArchive, error) {
	if len(archives) <= 0 {
		return nil, errors.New("No history archives provided")
	}
	return &PooledArchive{pool: archives}, nil
}

func (pa *PooledArchive) GetAnyArchive() ArchiveInterface {
	return pa.pool[rand.Intn(len(pa.pool))]
}

// Below are the ArchiveInterface method implementations.

func (pa *PooledArchive) GetPathHAS(path string) (HistoryArchiveState, error) {
	return pa.GetAnyArchive().GetPathHAS(path)
}

func (pa *PooledArchive) PutPathHAS(path string, has HistoryArchiveState, opts *CommandOptions) error {
	return pa.GetAnyArchive().PutPathHAS(path, has, opts)
}

func (pa *PooledArchive) BucketExists(bucket Hash) (bool, error) {
	return pa.GetAnyArchive().BucketExists(bucket)
}

func (pa *PooledArchive) CategoryCheckpointExists(cat string, chk uint32) (bool, error) {
	return pa.GetAnyArchive().CategoryCheckpointExists(cat, chk)
}

func (pa *PooledArchive) GetLedgerHeader(chk uint32) (xdr.LedgerHeaderHistoryEntry, error) {
	return pa.GetAnyArchive().GetLedgerHeader(chk)
}

func (pa *PooledArchive) GetRootHAS() (HistoryArchiveState, error) {
	return pa.GetAnyArchive().GetRootHAS()
}

func (pa *PooledArchive) GetCheckpointHAS(chk uint32) (HistoryArchiveState, error) {
	return pa.GetAnyArchive().GetCheckpointHAS(chk)
}

func (pa *PooledArchive) PutCheckpointHAS(chk uint32, has HistoryArchiveState, opts *CommandOptions) error {
	return pa.GetAnyArchive().PutCheckpointHAS(chk, has, opts)
}

func (pa *PooledArchive) PutRootHAS(has HistoryArchiveState, opts *CommandOptions) error {
	return pa.GetAnyArchive().PutRootHAS(has, opts)
}

func (pa *PooledArchive) ListBucket(dp DirPrefix) (chan string, chan error) {
	return pa.GetAnyArchive().ListBucket(dp)
}

func (pa *PooledArchive) ListAllBuckets() (chan string, chan error) {
	return pa.GetAnyArchive().ListAllBuckets()
}

func (pa *PooledArchive) ListAllBucketHashes() (chan Hash, chan error) {
	return pa.GetAnyArchive().ListAllBucketHashes()
}

func (pa *PooledArchive) ListCategoryCheckpoints(cat string, pth string) (chan uint32, chan error) {
	return pa.GetAnyArchive().ListCategoryCheckpoints(cat, pth)
}

func (pa *PooledArchive) GetXdrStreamForHash(hash Hash) (*XdrStream, error) {
	return pa.GetAnyArchive().GetXdrStreamForHash(hash)
}

func (pa *PooledArchive) GetXdrStream(pth string) (*XdrStream, error) {
	return pa.GetAnyArchive().GetXdrStream(pth)
}

func (pa *PooledArchive) GetCheckpointManager() CheckpointManager {
	return pa.GetAnyArchive().GetCheckpointManager()
}
