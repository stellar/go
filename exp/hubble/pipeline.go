// +build go1.13

package hubble

import (
	"context"

	"github.com/olivere/elastic/v7"
	"github.com/stellar/go/exp/ingest"
	"github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
)

const archivesURL = "http://history.stellar.org/prd/core-live/core_live_001/"

// NewStatePipelineSession runs a single ledger session.
func NewStatePipelineSession(esUrl, esIndex string) (*ingest.SingleLedgerSession, error) {
	archive, err := newArchive()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create archive")
	}
	statePipeline, err := newStatePipeline(esUrl, esIndex)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create state pipeline")
	}
	session := &ingest.SingleLedgerSession{
		Archive:       archive,
		StatePipeline: statePipeline,
	}
	return session, nil
}

func newArchive() (*historyarchive.Archive, error) {
	archive, err := historyarchive.Connect(
		archivesURL,
		historyarchive.ConnectOptions{},
	)
	if err != nil {
		return nil, err
	}
	return archive, nil
}

func newStatePipeline(esUrl, esIndex string) (*pipeline.StatePipeline, error) {
	sp := &pipeline.StatePipeline{}
	client, err := newClientWithIndex(esUrl, esIndex)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create new es client and index")
	}
	esProcessor := &ESProcessor{
		client: client,
		index:  esIndex,
	}
	sp.SetRoot(
		pipeline.StateNode(esProcessor),
	)
	return sp, nil
}

func newClientWithIndex(esUrl, esIndex string) (*elastic.Client, error) {
	client, err := elastic.NewClient(
		elastic.SetURL(esUrl),
	)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create es client")
	}

	ctx := context.Background()
	_, _, err = client.Ping(esUrl).Do(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't ping es server")
	}

	exists, err := client.IndexExists(esIndex).Do(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't check es index existence")
	}

	if !exists {
		_, err = client.CreateIndex(esIndex).Do(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't create es index")
		}
	}
	return client, nil
}
