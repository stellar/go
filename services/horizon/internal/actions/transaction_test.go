package actions

import (
	"context"
	"testing"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
)

var defaultPage db2.PageQuery = db2.PageQuery{
	Order:  db2.OrderAscending,
	Limit:  db2.DefaultPageSize,
	Cursor: "",
}

func TestTransactionPage(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	ctx := context.Background()

	// filter by account
	page, err := TransactionPage(ctx, &history.Q{tt.HorizonSession()}, "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H", 0, true, defaultPage)
	tt.Assert.NoError(err)
	tt.Assert.Equal(3, len(page.Embedded.Records))

	page, err = TransactionPage(ctx, &history.Q{tt.HorizonSession()}, "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", 0, true, defaultPage)
	tt.Assert.NoError(err)
	tt.Assert.Equal(1, len(page.Embedded.Records))

	page, err = TransactionPage(ctx, &history.Q{tt.HorizonSession()}, "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", 0, true, defaultPage)
	tt.Assert.NoError(err)
	tt.Assert.Equal(2, len(page.Embedded.Records))

	// filter by ledger
	page, err = TransactionPage(ctx, &history.Q{tt.HorizonSession()}, "", 1, true, defaultPage)
	tt.Assert.NoError(err)
	tt.Assert.Equal(0, len(page.Embedded.Records))

	page, err = TransactionPage(ctx, &history.Q{tt.HorizonSession()}, "", 2, true, defaultPage)
	tt.Assert.NoError(err)
	tt.Assert.Equal(3, len(page.Embedded.Records))

	page, err = TransactionPage(ctx, &history.Q{tt.HorizonSession()}, "", 3, true, defaultPage)
	tt.Assert.NoError(err)
	tt.Assert.Equal(1, len(page.Embedded.Records))

	// conflict fields
	_, err = TransactionPage(ctx, &history.Q{tt.HorizonSession()}, "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H", 1, true, defaultPage)
	tt.Assert.Error(err)
}

func TestLoadTransactionRecords(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	// filter by account
	records, err := loadTransactionRecords(&history.Q{tt.HorizonSession()}, "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H", 0, true, defaultPage)
	tt.Assert.NoError(err)
	tt.Assert.Equal(3, len(records))

	records, err = loadTransactionRecords(&history.Q{tt.HorizonSession()}, "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", 0, true, defaultPage)
	tt.Assert.NoError(err)
	tt.Assert.Equal(1, len(records))

	records, err = loadTransactionRecords(&history.Q{tt.HorizonSession()}, "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", 0, true, defaultPage)
	tt.Assert.NoError(err)
	tt.Assert.Equal(2, len(records))

	// filter by ledger
	records, err = loadTransactionRecords(&history.Q{tt.HorizonSession()}, "", 1, true, defaultPage)
	tt.Assert.NoError(err)
	tt.Assert.Equal(0, len(records))

	records, err = loadTransactionRecords(&history.Q{tt.HorizonSession()}, "", 2, true, defaultPage)
	tt.Assert.NoError(err)
	tt.Assert.Equal(3, len(records))

	records, err = loadTransactionRecords(&history.Q{tt.HorizonSession()}, "", 3, true, defaultPage)
	tt.Assert.NoError(err)
	tt.Assert.Equal(1, len(records))

	// conflict fields
	_, err = loadTransactionRecords(&history.Q{tt.HorizonSession()}, "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", 1, true, defaultPage)
	tt.Assert.Error(err)
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

	page, err := TransactionPage(
		context.Background(),
		q,
		"",
		0,
		false,
		db2.PageQuery{Cursor: "", Limit: 10, Order: db2.OrderAscending},
	)
	tt.Assert.NoError(err)
	tt.Assert.Len(page.Embedded.Records, 2)

	feeBumpResponse := page.Embedded.Records[0].(horizon.Transaction)
	checkOuterHashResponse(tt, fixture, feeBumpResponse)

	normalTxResponse := page.Embedded.Records[1].(horizon.Transaction)
	tt.Assert.Equal(fixture.NormalTransaction.TransactionHash, normalTxResponse.ID)
}

func TestFeeBumpTransactionResource(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &history.Q{tt.HorizonSession()}
	fixture := history.FeeBumpScenario(tt, q, true)

	byOuterHash, err := TransactionResource(context.Background(), q, fixture.OuterHash)
	tt.Assert.NoError(err)

	checkOuterHashResponse(tt, fixture, byOuterHash)

	byInnerHash, err := TransactionResource(context.Background(), q, fixture.InnerHash)
	tt.Assert.NoError(err)

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
