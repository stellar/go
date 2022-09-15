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
	tmp := t.TempDir()
	store, err := index.NewFileStore(tmp,
		index.StoreConfig{
			URL:     "file://" + tmp,
			Workers: 4,
		},
	)
	require.NoError(t, err)

	for _, checkpoint := range []uint32{1, 5, 10, 12} {
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

	// cursor increments
	for i := int32(1); i < freq; i++ {
		nextCursor, err = cursorMgr.Advance(1)
		require.NoError(t, err)
		assert.EqualValues(t, 4*freq+i, getLedgerFromCursor(nextCursor))
	}

	// cursor jumps to next active checkpoint
	nextCursor, err = cursorMgr.Advance(1)
	require.NoError(t, err)
	assert.EqualValues(t, 9*freq, getLedgerFromCursor(nextCursor))

	// cursor skips
	nextCursor, err = cursorMgr.Advance(5)
	require.NoError(t, err)
	assert.EqualValues(t, 9*freq+5, getLedgerFromCursor(nextCursor))

	// cursor jumps to next active when skipping
	nextCursor, err = cursorMgr.Advance(uint(freq - 5))
	require.NoError(t, err)
	assert.EqualValues(t, 11*freq, getLedgerFromCursor(nextCursor))

	// cursor EOFs at the end
	nextCursor, err = cursorMgr.Advance(uint(freq - 1))
	require.NoError(t, err)
	assert.EqualValues(t, 12*freq-1, getLedgerFromCursor(nextCursor))
	_, err = cursorMgr.Advance(1)
	assert.ErrorIs(t, err, io.EOF)

	// cursor EOFs if skipping past the end
	rewind := toid.New(int32(getLedgerFromCursor(nextCursor)-5), 0, 0)
	nextCursor, err = cursorMgr.Begin(rewind.ToInt64())
	require.NoError(t, err)
	assert.EqualValues(t, rewind.LedgerSequence, getLedgerFromCursor(nextCursor))
	_, err = cursorMgr.Advance(uint(freq))
	assert.ErrorIs(t, err, io.EOF)
}
