package hubble

import (
	"github.com/stellar/go/exp/ingest"
	"github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/support/historyarchive"
)

const archivesURL string = "s3://history.stellar.org/prd/core-live/core_live_001/"
const archivesRegion string = "eu-west-1"

// RunStatePipelineSession runs a single ledger session.
func RunStatePipelineSession() error {
	archive, err := historyarchive.Connect(
		archivesURL,
		historyarchive.ConnectOptions{
			S3Region:         archivesRegion,
			UnsignedRequests: true,
		},
	)
	if err != nil {
		return err
	}

	statePipeline := GetStatePipeline()
	session := &ingest.SingleLedgerSession{
		Archive:       archive,
		StatePipeline: statePipeline,
	}
	err = session.Run()
	return err
}

// GetStatePipeline returns a state pipeline.
func GetStatePipeline() *pipeline.StatePipeline {
	sp := &pipeline.StatePipeline{}
	prettyPrintEntryProcessor := &PrettyPrintEntryProcessor{}

	sp.SetRoot(
		pipeline.StateNode(prettyPrintEntryProcessor),
	)
	return sp
}
