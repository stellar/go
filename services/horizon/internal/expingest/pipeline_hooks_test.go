package expingest

import (
	"context"
	"testing"

	"github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	horizonProcessors "github.com/stellar/go/services/horizon/internal/expingest/processors"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func TestStatePreProcessingHook(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	session := tt.HorizonSession()
	defer session.Rollback()
	ctx := context.WithValue(
		context.Background(),
		pipeline.LedgerSequenceContextKey,
		uint32(0),
	)
	pipelineType := statePipeline
	historyQ := &history.Q{session}
	tt.Assert.Nil(historyQ.UpdateLastLedgerExpIngest(0))

	tt.Assert.Nil(session.GetTx())
	newCtx, err := preProcessingHook(ctx, pipelineType, session)
	tt.Assert.NoError(err)
	tt.Assert.NotNil(session.GetTx())
	tt.Assert.Nil(newCtx.Value(horizonProcessors.IngestUpdateDatabase))

	tt.Assert.Nil(session.Rollback())
	tt.Assert.Nil(session.GetTx())

	tt.Assert.Nil(session.Begin())
	tt.Assert.NotNil(session.GetTx())

	newCtx, err = preProcessingHook(ctx, pipelineType, session)
	tt.Assert.NoError(err)
	tt.Assert.NotNil(session.GetTx())
	tt.Assert.Nil(newCtx.Value(horizonProcessors.IngestUpdateDatabase))
}

func TestLedgerPreProcessingHook(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	session := tt.HorizonSession()
	defer session.Rollback()
	ctx := context.WithValue(
		context.Background(),
		pipeline.LedgerSequenceContextKey,
		uint32(2),
	)
	pipelineType := ledgerPipeline
	historyQ := &history.Q{session}
	tt.Assert.Nil(historyQ.UpdateLastLedgerExpIngest(1))

	tt.Assert.Nil(session.GetTx())
	newCtx, err := preProcessingHook(ctx, pipelineType, session)
	tt.Assert.NoError(err)
	tt.Assert.NotNil(session.GetTx())
	tt.Assert.Equal(newCtx.Value(horizonProcessors.IngestUpdateDatabase), true)

	tt.Assert.Nil(session.Rollback())
	tt.Assert.Nil(session.GetTx())

	tt.Assert.Nil(session.Begin())
	tt.Assert.NotNil(session.GetTx())
	newCtx, err = preProcessingHook(ctx, pipelineType, session)
	tt.Assert.NoError(err)
	tt.Assert.NotNil(session.GetTx())
	tt.Assert.Equal(newCtx.Value(horizonProcessors.IngestUpdateDatabase), true)

	tt.Assert.Nil(session.Rollback())
	tt.Assert.Nil(session.GetTx())

	tt.Assert.Nil(historyQ.UpdateLastLedgerExpIngest(2))
	newCtx, err = preProcessingHook(ctx, pipelineType, session)
	tt.Assert.NoError(err)
	tt.Assert.Nil(session.GetTx())
	tt.Assert.Nil(newCtx.Value(horizonProcessors.IngestUpdateDatabase))

	tt.Assert.Nil(session.Begin())
	tt.Assert.NotNil(session.GetTx())
	newCtx, err = preProcessingHook(ctx, pipelineType, session)
	tt.Assert.NoError(err)
	tt.Assert.Nil(session.GetTx())
	tt.Assert.Nil(newCtx.Value(horizonProcessors.IngestUpdateDatabase))
}

func TestPostProcessingHook(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	account := "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2"
	signer := "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"
	weight := int32(123)
	accountSigner := history.AccountSigner{
		Account: account,
		Signer:  signer,
		Weight:  weight,
	}

	session := tt.HorizonSession()
	defer session.Rollback()
	historyQ := &history.Q{session}
	for _, testCase := range []struct {
		name           string
		err            error
		lastLedger     uint32
		pipelineLedger uint32
		inTx           bool
		expectedError  string
	}{
		{
			"succeeds when last ledger in db is 0",
			nil,
			0,
			3,
			true,
			"",
		},
		{
			"succeeds when local latest sequence is equal to global sequence",
			nil,
			2,
			3,
			true,
			"",
		},
		{
			"succeeds when we're not in a tx",
			nil,
			1,
			3,
			false,
			"",
		},
		{
			"fails because of passed in error",
			errors.New("test case error"),
			2,
			3,
			false,
			"test case error",
		},
		{
			"fails because local latest sequence is not equal to global sequence",
			nil,
			1,
			3,
			true,
			"local latest sequence is not equal to global sequence",
		},
	} {
		t.Run(testCase.name, func(_ *testing.T) {
			tt.Assert.Nil(historyQ.UpdateLastLedgerExpIngest(testCase.lastLedger))
			tt.Assert.Nil(historyQ.UpdateExpIngestVersion(0))
			_, err := historyQ.RemoveAccountSigner(account, signer)
			tt.Assert.NoError(err)

			ctx := context.WithValue(
				context.Background(),
				pipeline.LedgerSequenceContextKey,
				testCase.pipelineLedger,
			)
			graph := orderbook.NewOrderBookGraph()
			// queue an offer on the orderbook so we can check if the post
			// processing hook applied it
			graph.AddOffer(eurOffer)

			if testCase.inTx {
				tt.Assert.Nil(session.Begin())
				// queue an insert on the transaction so we can check if the post
				// processing hook committed it to the db
				_, err = historyQ.CreateAccountSigner(account, signer, weight)
				tt.Assert.NoError(err)
			}

			err = postProcessingHook(ctx, testCase.err, statePipeline, nil, graph, session)
			if testCase.expectedError == "" {
				tt.Assert.NoError(err)
				tt.Assert.Equal(graph.Offers(), []xdr.OfferEntry{eurOffer})
			} else {
				tt.Assert.Contains(err.Error(), testCase.expectedError)
				tt.Assert.Equal(graph.Offers(), []xdr.OfferEntry{})
			}
			tt.Assert.Nil(session.GetTx())

			if testCase.inTx && testCase.expectedError == "" {
				// check that the ingest version and the ingest sequence was updated
				version, err := historyQ.GetExpIngestVersion()
				tt.Assert.NoError(err)
				tt.Assert.Equal(version, CurrentVersion)
				seq, err := historyQ.GetLastLedgerExpIngestNonBlocking()
				tt.Assert.NoError(err)
				tt.Assert.Equal(seq, testCase.pipelineLedger)

				// check that the transaction was committed
				accounts, err := historyQ.AccountsForSigner(signer, db2.PageQuery{Order: "asc", Limit: 10})
				tt.Assert.NoError(err)
				tt.Assert.Len(accounts, 1)
				tt.Assert.Equal(accountSigner, accounts[0])
			} else {
				// check that the transaction was rolled back and nothing was committed
				version, err := historyQ.GetExpIngestVersion()
				tt.Assert.NoError(err)
				tt.Assert.Equal(version, 0)
				seq, err := historyQ.GetLastLedgerExpIngestNonBlocking()
				tt.Assert.NoError(err)
				tt.Assert.Equal(seq, testCase.lastLedger)

				accounts, err := historyQ.AccountsForSigner(signer, db2.PageQuery{Order: "asc", Limit: 10})
				tt.Assert.NoError(err)
				tt.Assert.Len(accounts, 0)
			}
		})
	}
}
