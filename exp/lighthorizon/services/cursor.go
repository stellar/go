package services

import (
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/toid"
)

// CursorManager describes a way to control how a cursor advances for a
// particular indexing strategy.
type CursorManager interface {
	Begin(cursor int64) (int64, error)
	Advance() (int64, error)
}

type AccountActivityCursorManager struct {
	AccountId string

	store      index.Store
	lastCursor *toid.ID
}

func NewCursorManagerForAccountActivity(store index.Store, accountId string) *AccountActivityCursorManager {
	return &AccountActivityCursorManager{AccountId: accountId, store: store}
}

func (c *AccountActivityCursorManager) Begin(cursor int64) (int64, error) {
	freq := checkpointManager.GetCheckpointFrequency()
	id := toid.Parse(cursor)
	lastCheckpoint := uint32(0)
	if id.LedgerSequence >= int32(checkpointManager.GetCheckpointFrequency()) {
		lastCheckpoint = index.GetCheckpointNumber(uint32(id.LedgerSequence))
	}

	// We shouldn't take the provided cursor for granted: instead, we should
	// skip ahead to the first active ledger that's >= the given cursor.
	//
	// For example, someone might say ?cursor=0 but the first active checkpoint
	// is actually 40M ledgers in.
	firstCheckpoint, err := c.store.NextActive(c.AccountId, allTransactionsIndex, lastCheckpoint)
	if err != nil {
		return cursor, err
	}

	nextLedger := (firstCheckpoint - 1) * freq

	// However, if the given cursor is actually *more* specific than the index
	// can give us (e.g. somewhere *within* an active checkpoint range), prefer
	// it rather than starting over.
	if nextLedger < uint32(id.LedgerSequence) {
		better := toid.Parse(cursor)
		c.lastCursor = &better
		return cursor, nil
	}

	c.lastCursor = toid.New(int32(nextLedger), 1, 1)
	return c.lastCursor.ToInt64(), nil
}

func (c *AccountActivityCursorManager) Advance() (int64, error) {
	if c.lastCursor == nil {
		panic("invalid cursor, call Begin() first")
	}

	// Advancing the cursor means deciding whether or not we need to query the
	// index.

	lastLedger := uint32(c.lastCursor.LedgerSequence)
	freq := checkpointManager.GetCheckpointFrequency()

	if checkpointManager.IsCheckpoint(lastLedger) {
		// If the last cursor we looked at was a checkpoint ledger, then we need
		// to jump ahead to the next checkpoint. Note that NextActive() is
		// "inclusive" so if the parameter is an active checkpoint it will
		// return itself.
		checkpoint := index.GetCheckpointNumber(uint32(c.lastCursor.LedgerSequence))
		checkpoint, err := c.store.NextActive(c.AccountId, allTransactionsIndex, checkpoint+1)
		if err != nil {
			return c.lastCursor.ToInt64(), err
		}

		// We add a -1 here because an active checkpoint indicates that an
		// account had activity in the *previous* 64 ledgers, so we need to
		// backtrack to that ledger range.
		c.lastCursor = toid.New(int32((checkpoint-1)*freq), 1, 1)
	} else {
		// Otherwise, we can just bump the ledger number.
		c.lastCursor = toid.New(int32(lastLedger+1), 1, 1)
	}

	return c.lastCursor.ToInt64(), nil
}

var _ CursorManager = (*AccountActivityCursorManager)(nil) // ensure conformity to the interface

// getLedgerFromCursor is a helpful way to turn a cursor into a ledger number
func getLedgerFromCursor(cursor int64) uint32 {
	return uint32(toid.Parse(cursor).LedgerSequence)
}
