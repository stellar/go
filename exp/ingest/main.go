package ingest

import (
	"sync"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/support/historyarchive"
)

// standardSession contains common methods used by all sessions.
type standardSession struct {
	shutdown chan bool

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
	// Run() start the session and works as long as the session is active.
	Run() error
	// Shutdown() gracefully stops running session and stops all internal
	// objects.
	Shutdown()
}
