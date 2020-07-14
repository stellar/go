package actions

import (
	"net/http/httptest"
	"testing"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
	supportProblem "github.com/stellar/go/support/render/problem"
)

func TestGetTransactionsHandler(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	q := &history.Q{tt.HorizonSession()}
	handler := GetTransactionsHandler{}

	// filter by account
	records, err := handler.GetResourcePage(
		httptest.NewRecorder(),
		makeRequest(
			t, map[string]string{
				"account_id":     "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
				"include_failed": "true",
			}, map[string]string{}, q.Session,
		),
	)
	tt.Assert.NoError(err)
	tt.Assert.Len(records, 3)

	records, err = handler.GetResourcePage(
		httptest.NewRecorder(),
		makeRequest(
			t, map[string]string{
				"account_id":     "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
				"include_failed": "true",
			}, map[string]string{}, q.Session,
		),
	)
	tt.Assert.NoError(err)
	tt.Assert.Len(records, 1)

	records, err = handler.GetResourcePage(
		httptest.NewRecorder(),
		makeRequest(
			t, map[string]string{
				"account_id":     "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
				"include_failed": "true",
			}, map[string]string{}, q.Session,
		),
	)
	tt.Assert.NoError(err)
	tt.Assert.Len(records, 2)

	// // filter by ledger
	records, err = handler.GetResourcePage(
		httptest.NewRecorder(),
		makeRequest(
			t, map[string]string{
				"ledger_id":      "1",
				"include_failed": "true",
			}, map[string]string{}, q.Session,
		),
	)
	tt.Assert.NoError(err)
	tt.Assert.Len(records, 0)

	records, err = handler.GetResourcePage(
		httptest.NewRecorder(),
		makeRequest(
			t, map[string]string{
				"ledger_id":      "2",
				"include_failed": "true",
			}, map[string]string{}, q.Session,
		),
	)
	tt.Assert.NoError(err)
	tt.Assert.Len(records, 3)

	records, err = handler.GetResourcePage(
		httptest.NewRecorder(),
		makeRequest(
			t, map[string]string{
				"ledger_id":      "3",
				"include_failed": "true",
			}, map[string]string{}, q.Session,
		),
	)
	tt.Assert.NoError(err)
	tt.Assert.Len(records, 1)

	records, err = handler.GetResourcePage(
		httptest.NewRecorder(),
		makeRequest(
			t, map[string]string{
				"account_id":     "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
				"ledger_id":      "3",
				"include_failed": "true",
			}, map[string]string{}, q.Session,
		),
	)
	tt.Assert.IsType(&supportProblem.P{}, err)
	p := err.(*supportProblem.P)
	tt.Assert.Equal("bad_request", p.Type)
	tt.Assert.Equal("filters", p.Extras["invalid_field"])
	tt.Assert.Equal(
		"Use a single filter for transaction, you can only use one of account_id or ledger_id",
		p.Extras["reason"],
	)
}

func checkOuterHashResponse(
	tt *test.T,
	fixture history.FeeBumpFixture,
	transactionResponse horizon.Transaction,
) {
	tt.Assert.Equal(fixture.Transaction.Account, transactionResponse.Account)
	tt.Assert.Equal(fixture.Transaction.AccountSequence, transactionResponse.AccountSequence)
	tt.Assert.Equal(fixture.Transaction.FeeAccount.String, transactionResponse.FeeAccount)
	tt.Assert.Equal(fixture.Transaction.FeeCharged, transactionResponse.FeeCharged)
	tt.Assert.Equal(fixture.Transaction.TransactionHash, transactionResponse.ID)
	tt.Assert.Equal(fixture.Transaction.MaxFee, transactionResponse.InnerTransaction.MaxFee)
	tt.Assert.Equal(
		[]string(fixture.Transaction.InnerSignatures),
		transactionResponse.InnerTransaction.Signatures,
	)
	tt.Assert.Equal(
		fixture.Transaction.InnerTransactionHash.String,
		transactionResponse.InnerTransaction.Hash,
	)
	tt.Assert.Equal(fixture.Transaction.NewMaxFee.Int64, transactionResponse.MaxFee)
	tt.Assert.Equal(fixture.Transaction.Memo.String, transactionResponse.Memo)
	tt.Assert.Equal(fixture.Transaction.MemoType, transactionResponse.MemoType)
	tt.Assert.Equal(fixture.Transaction.OperationCount, transactionResponse.OperationCount)
	tt.Assert.Equal(
		[]string(fixture.Transaction.Signatures),
		transactionResponse.Signatures,
	)
	tt.Assert.Equal(fixture.Transaction.Successful, transactionResponse.Successful)
	tt.Assert.Equal(fixture.Transaction.TotalOrderID.PagingToken(), transactionResponse.PT)
	tt.Assert.Equal(fixture.Transaction.TransactionHash, transactionResponse.Hash)
	tt.Assert.Equal(fixture.Transaction.TxEnvelope, transactionResponse.EnvelopeXdr)
	tt.Assert.Equal(fixture.Transaction.TxFeeMeta, transactionResponse.FeeMetaXdr)
	tt.Assert.Equal(fixture.Transaction.TxMeta, transactionResponse.ResultMetaXdr)
	tt.Assert.Equal(fixture.Transaction.TxResult, transactionResponse.ResultXdr)
}

func TestFeeBumpTransactionPage(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &history.Q{tt.HorizonSession()}
	fixture := history.FeeBumpScenario(tt, q, true)
	handler := GetTransactionsHandler{}

	records, err := handler.GetResourcePage(
		httptest.NewRecorder(),
		makeRequest(
			t, map[string]string{}, map[string]string{}, q.Session,
		),
	)
	tt.Assert.NoError(err)
	tt.Assert.Len(records, 2)

	feeBumpResponse := records[0].(horizon.Transaction)
	checkOuterHashResponse(tt, fixture, feeBumpResponse)

	normalTxResponse := records[1].(horizon.Transaction)
	tt.Assert.Equal(fixture.NormalTransaction.TransactionHash, normalTxResponse.ID)
}

func TestFeeBumpTransactionResource(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &history.Q{tt.HorizonSession()}
	fixture := history.FeeBumpScenario(tt, q, true)

	handler := GetTransactionByHashHandler{}
	resource, err := handler.GetResource(
		httptest.NewRecorder(),
		makeRequest(
			t, map[string]string{}, map[string]string{
				"tx_id": fixture.OuterHash,
			}, q.Session,
		),
	)
	tt.Assert.NoError(err)
	byOuterHash := resource.(horizon.Transaction)
	checkOuterHashResponse(tt, fixture, byOuterHash)

	resource, err = handler.GetResource(
		httptest.NewRecorder(),
		makeRequest(
			t, map[string]string{}, map[string]string{
				"tx_id": fixture.InnerHash,
			}, q.Session,
		),
	)
	tt.Assert.NoError(err)

	byInnerHash := resource.(horizon.Transaction)

	tt.Assert.NotEqual(byOuterHash.Hash, byInnerHash.Hash)
	tt.Assert.NotEqual(byOuterHash.ID, byInnerHash.ID)
	tt.Assert.NotEqual(byOuterHash.Signatures, byInnerHash.Signatures)

	tt.Assert.Equal(fixture.InnerHash, byInnerHash.Hash)
	tt.Assert.Equal(fixture.InnerHash, byInnerHash.ID)
	tt.Assert.Equal(
		[]string(fixture.Transaction.InnerSignatures),
		byInnerHash.Signatures,
	)

	byInnerHash.Hash = byOuterHash.Hash
	byInnerHash.ID = byOuterHash.ID
	byInnerHash.Signatures = byOuterHash.Signatures
	byInnerHash.Links = byOuterHash.Links
	tt.Assert.Equal(byOuterHash, byInnerHash)
}
