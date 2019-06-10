package ingest

import (
	"github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/support/historyarchive"
)

type System struct {
	//
}

// LiveSession initializes the ledger state using `Archive` and `StatePipeline`,
// then starts processing ledger data using `ledgerbackend`.
type LiveSession struct {
	Archive *historyarchive.Archive

	// mutex is used to make sure queries across many stores are persistent
	// mutex         sync.RWMutex
	pipeline *pipeline.StatePipeline
	shutdown chan bool
}

// SingleLedgerSession initializes the ledger state using `Archive` and `StatePipeline`
// and terminates. Useful for finding stats for a single ledger. Set `LedgerSequence`
// to `0` to process the latest checkpoint.
type SingleLedgerSession struct {
	Archive        *historyarchive.Archive
	LedgerSequence uint32

	pipeline *pipeline.StatePipeline
	shutdown chan bool
}

type Session interface {
	Run() error
	AddPipeline(*pipeline.StatePipeline)
	Shutdown()
}
