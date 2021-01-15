package ingest

import (
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/support/errors"
)

// ErrNotFound is returned when the requested ledger is not found
var ErrNotFound = errors.New("ledger not found")

// ErrNotCheckpoint is returned when the requested ledger sequence number does
// not correspond to a checkpoint ledger.
//
// You can use the fields within the parent `CheckpointError` object (below) to
// resolve to an accurate checkpoint.
//
// The nth ledger n is a checkpoint ledger iff: n+1 mod f == 0, where f is the
// checkpoint frequency (64 by default).
var ErrNotCheckpoint = errors.New("sequence number is not a checkpoint ledger")

type CheckpointError struct {
	error
	precedingCheckpoint uint32
	followingCheckpoint uint32
}

func NewCheckpointError(sequence uint32, checkpointMgr historyarchive.CheckpointManager) CheckpointError {
	return CheckpointError{
		ErrNotCheckpoint,
		checkpointMgr.PrevCheckpoint(sequence),
		checkpointMgr.NextCheckpoint(sequence),
	}
}

// StateError is a fatal error indicating that the Change stream
// produced a result which violates fundamental invariants (e.g. an account
// transferred more XLM than the account held in its balance).
type StateError struct {
	error
}

// NewStateError creates a new StateError.
func NewStateError(err error) StateError {
	return StateError{err}
}
