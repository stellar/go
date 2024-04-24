package ledgerbackend

import (
	"context"
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func MockGCSBackend() GCSBackend {
	lcmCache := &LCMCache{lcm: make(map[uint32]xdr.LedgerCloseMeta)}
	return GCSBackend{
		fileSuffix: ".xdr.gz",
		lcmCache:   lcmCache,
	}
}

func TestGCSBackendGetLedger(t *testing.T) {
	gcsb := MockGCSBackend()
	gcsb.lcmCache.lcm[1] = xdr.LedgerCloseMeta{V: 0}
	gcsb.lcmCache.lcm[2] = xdr.LedgerCloseMeta{V: 1}
	ledgerRange := BoundedRange(1, 2)
	gcsb.prepared = &ledgerRange
	ctx := context.Background()

	lcm, err := gcsb.GetLedger(ctx, 1)

	assert.NoError(t, err)
	assert.Equal(t, xdr.LedgerCloseMeta{V: 0}, lcm)
}

func TestGetLatestFileNameLedgerSequence(t *testing.T) {
	gcsb := MockGCSBackend()
	directory := "ledgers/pubnet/21-30"
	filenames := []string{
		"ledgers/pubnet/21-30/21.xdr.gz",
		"ledgers/pubnet/21-30/22.xdr.gz",
		"ledgers/pubnet/21-30/23.xdr.gz",
	}
	latestLedgerSequence, _ := gcsb.GetLatestFileNameLedgerSequence(filenames, directory)

	assert.Equal(t, uint32(23), latestLedgerSequence)
}

func TestGetLatestDirectory(t *testing.T) {
	gcsb := MockGCSBackend()
	directories := []string{"ledgers/pubnet/1-10", "ledgers/pubnet/11-20", "ledgers/pubnet/21-30"}
	latestDirectory, _ := gcsb.GetLatestDirectory(directories)

	assert.Equal(t, "ledgers/pubnet/21-30", latestDirectory)
}
