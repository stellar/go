package tools

import (
	"path/filepath"
	"testing"

	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/log"
	"github.com/stretchr/testify/require"
)

const (
	freq = historyarchive.DefaultCheckpointFrequency
)

func TestIndexPurge(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	tempFile := "file://" + filepath.Join(t.TempDir(), "index-store")
	accounts := []string{keypair.MustRandom().Address()}

	idx, err := index.Connect(tempFile)
	require.NoError(t, err)

	for _, chk := range []uint32{14, 15, 16, 17, 20, 25, 123} {
		require.NoError(t, idx.AddParticipantsToIndexes(chk, "test", accounts))
	}

	idx.Flush() // saves to disk

	// Try purging the index
	err = purgeIndex(tempFile, historyarchive.Range{Low: 15 * freq, High: 22 * freq})
	require.NoError(t, err)

	// Check to make sure it worked.
	idx, err = index.Connect(tempFile)
	require.NoError(t, err)

	// Ensure that the index is in the expected state.
	indices, err := idx.Read(accounts[0])
	require.NoError(t, err)
	require.Contains(t, indices, "test")

	index := indices["test"]
	i, err := index.NextActiveBit(0)
	require.NoError(t, err)
	require.EqualValues(t, 14, i)

	i, err = index.NextActiveBit(15)
	require.NoError(t, err)
	require.EqualValues(t, 25, i)

	i, err = index.NextActiveBit(i + 1)
	require.NoError(t, err)
	require.EqualValues(t, 123, i)
}
