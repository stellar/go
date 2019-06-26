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

// LiveSession initializes the ledger state using `Archive` and `StatePipeline`,
// then starts processing ledger data using `ledgerbackend`.
type LiveSession struct {
	standardSession

	Archive       historyarchive.ArchiveInterface
	LedgerBackend ledgerbackend.LedgerBackend

	// mutex is used to make sure queries across many stores are persistent
	// mutex         sync.RWMutex

	statePipeline  *pipeline.StatePipeline
	ledgerPipeline *pipeline.LedgerPipeline

	currentLedger uint32
}

// SingleLedgerSession initializes the ledger state using `Archive` and `StatePipeline`
// and terminates. Useful for finding stats for a single ledger. Set `LedgerSequence`
// to `0` to process the latest checkpoint.
type SingleLedgerSession struct {
	standardSession

	Archive        *historyarchive.Archive
	LedgerSequence uint32

	statePipeline *pipeline.StatePipeline
}

type Session interface {
	Run() error
	SetStatePipeline(*pipeline.StatePipeline)
	SetLedgerPipeline(*pipeline.LedgerPipeline)
	Shutdown()
}
