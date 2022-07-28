package services

import (
	"io"
	"testing"

	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/toid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	checkpointMgr = historyarchive.NewCheckpointManager(0)
)

func TestAccountTransactionCursorManager(t *testing.T) {
	freq := int32(checkpointMgr.GetCheckpointFrequency())
	accountId := keypair.MustRandom().Address()

	// Create an index and fill it with some checkpoint details.
	store, err := index.NewFileStore(index.StoreConfig{
		Url:     "file://" + t.TempDir(),
		Workers: 4,
	})
	require.NoError(t, err)

	for _, checkpoint := range []uint32{1, 5, 10} {
		require.NoError(t, store.AddParticipantsToIndexes(
			checkpoint, allTransactionsIndex, []string{accountId}))
	}

	cursorMgr := NewCursorManagerForAccountActivity(store, accountId)

	cursor := toid.New(1, 1, 1)
	var nextCursor int64

	// first checkpoint works
	nextCursor, err = cursorMgr.Begin(cursor.ToInt64())
	require.NoError(t, err)
	assert.EqualValues(t, 1, getLedgerFromCursor(nextCursor))

	// cursor is preserved if mid-active-range
	cursor.LedgerSequence = freq / 2
	nextCursor, err = cursorMgr.Begin(cursor.ToInt64())
	require.NoError(t, err)
	assert.EqualValues(t, cursor.LedgerSequence, getLedgerFromCursor(nextCursor))

	// cursor jumps ahead if not active
	cursor.LedgerSequence = 2 * freq
	nextCursor, err = cursorMgr.Begin(cursor.ToInt64())
	require.NoError(t, err)
	assert.EqualValues(t, 4*freq, getLedgerFromCursor(nextCursor))

	for i := int32(1); i < freq; i++ {
		nextCursor, err = cursorMgr.Advance()
		require.NoError(t, err)
		assert.EqualValues(t, 4*freq+i, getLedgerFromCursor(nextCursor))
	}

	// cursor jumps to next active checkpoint
	nextCursor, err = cursorMgr.Advance()
	require.NoError(t, err)
	assert.EqualValues(t, 9*freq, getLedgerFromCursor(nextCursor))

	// cursor increments
	for i := int32(1); i < freq; i++ {
		nextCursor, err = cursorMgr.Advance()
		require.NoError(t, err)
		assert.EqualValues(t, 9*freq+i, getLedgerFromCursor(nextCursor))
	}

	// cursor stops when no more actives
	_, err = cursorMgr.Advance()
	assert.ErrorIs(t, err, io.EOF)
}
