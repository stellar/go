package ingest

import (
	"sync"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/support/historyarchive"
)

// standardSession contains common methods used by all sessions.
type standardSession struct {
	shutdown              chan bool
	rwLock                sync.RWMutex
	latestProcessedLedger uint32

	doneMutex sync.Mutex
	done      bool
}

// LiveSession initializes the ledger state using `Archive` and `statePipeline`,
// then starts processing ledger data using `LedgerBackend` and `ledgerPipeline`.
type LiveSession struct {
	standardSession

	Archive        historyarchive.ArchiveInterface
	LedgerBackend  ledgerbackend.LedgerBackend
	StatePipeline  *pipeline.StatePipeline
	LedgerPipeline *pipeline.LedgerPipeline
}

// SingleLedgerSession initializes the ledger state using `Archive` and `StatePipeline`
// and terminates. Useful for finding stats for a single ledger. Set `LedgerSequence`
// to `0` to process the latest checkpoint.
type SingleLedgerSession struct {
	standardSession

	Archive        *historyarchive.Archive
	LedgerSequence uint32
	StatePipeline  *pipeline.StatePipeline
}

// Session is an implementation of a ingesting scenario. Some useful sessions
// can be found in this package.
type Session interface {
	// Run start the session and works as long as the session is active. If
	// you want to resume a session use Resume().
	Run() error
	// Resume resumes the session at ledger with a given sequence. It's up to
	// Session user to determine what was the last ledger processed by a
	// Session as it's stateless (or if Run() should be called first).
	Resume(ledgerSequence uint32) error
	// GetLatestProcessedLedger returns the latest ledger sequence processed in
	// the session. Return 0 if no ledgers were processed yet.
	// Please note that this value is not synchronized with pipeline hooks.
	GetLatestProcessedLedger() uint32
	// Shutdown gracefully stops running session and stops all internal
	// objects.
	// Calling Shutdown() does not trigger post processing hooks.
	Shutdown()
	// QueryLock locks the session for sending a query. This ensures a consistent
	// view when data is saved to multiple stores (ex. postgres/redis/memory).
	// TODO this is fine for a demo but we need to check if it works for systems
	// that don't provide strong consistency. This may also slow down readers
	// if commiting data to stores take longer time. This probably doesn't work
	// in distributed apps.
	QueryLock()
	// QueryUnlock unlocks read lock of the session.
	QueryUnlock()
	// UpdateLock locks the session for updating data. This ensures a consistent
	// view when data is saved to multiple stores (ex. postgres/redis/memory).
	// TODO this is fine for a demo but we need to check if it works for systems
	// that don't provide strong consistency. This may also slow down readers
	// if commiting data to stores take longer time. This probably doesn't work
	// in distributed apps.
	UpdateLock()
	// UpdateUnlock unlocks write lock of the session.
	UpdateUnlock()
}
