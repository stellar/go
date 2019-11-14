package hubble

import (
	"github.com/stellar/go/exp/ingest"
	"github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/support/historyarchive"
)

const archivesURL = "http://history.stellar.org/prd/core-live/core_live_001/"

// NewStatePipelineSession runs a single ledger session.
func NewStatePipelineSession() (*ingest.SingleLedgerSession, error) {
	archive, err := historyarchive.Connect(
		archivesURL,
		historyarchive.ConnectOptions{},
	)
	if err != nil {
		return nil, err
	}

	statePipeline := NewStatePipeline()
	session := &ingest.SingleLedgerSession{
		Archive:       archive,
		StatePipeline: statePipeline,
	}
	return session, nil
}

// NewStatePipeline returns a state pipeline.
func NewStatePipeline() *pipeline.StatePipeline {
	sp := &pipeline.StatePipeline{}
	prettyPrintEntryProcessor := &PrettyPrintEntryProcessor{}

	sp.SetRoot(
		pipeline.StateNode(prettyPrintEntryProcessor),
	)
	return sp
}
