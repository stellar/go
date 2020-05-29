package resourceadapter

import (
	"encoding/json"
	"testing"

	"github.com/guregu/null"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/test"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

// TestPopulateOperation_Successful tests operation object population.
func TestPopulateOperation_Successful(t *testing.T) {
	ctx, _ := test.ContextWithLogBuffer()

	var (
		dest   operations.Base
		row    history.Operation
		ledger = history.Ledger{}
	)

	dest = operations.Base{}
	row = history.Operation{TransactionSuccessful: true}

	assert.NoError(
		t,
		PopulateBaseOperation(ctx, &dest, row, "", nil, ledger),
	)
	assert.True(t, dest.TransactionSuccessful)
	assert.Nil(t, dest.Transaction)

	dest = operations.Base{}
	row = history.Operation{TransactionSuccessful: false}

	assert.NoError(
		t,
		PopulateBaseOperation(ctx, &dest, row, "", nil, ledger),
	)
	assert.False(t, dest.TransactionSuccessful)
	assert.Nil(t, dest.Transaction)
}

// TestPopulateOperation_WithTransaction tests PopulateBaseOperation when passing both an operation and a transaction.
func TestPopulateOperation_WithTransaction(t *testing.T) {
	ctx, _ := test.ContextWithLogBuffer()

	var (
		dest           operations.Base
		operationsRow  history.Operation
		ledger         = history.Ledger{}
		transactionRow history.Transaction
	)

	dest = operations.Base{}
	operationsRow = history.Operation{TransactionSuccessful: true}
	transactionRow = history.Transaction{
		TransactionWithoutLedger: history.TransactionWithoutLedger{
			Successful: true,
			MaxFee:     10000,
			FeeCharged: 100,
		},
	}

	assert.NoError(
		t,
		PopulateBaseOperation(
			ctx,
			&dest,
			operationsRow,
			transactionRow.TransactionHash,
			&transactionRow,
			ledger,
		),
	)
	assert.True(t, dest.TransactionSuccessful)
	assert.True(t, dest.Transaction.Successful)
	assert.Equal(t, int64(100), dest.Transaction.FeeCharged)
	assert.Equal(t, int64(10000), dest.Transaction.MaxFee)
}

func TestPopulateOperation_AllowTrust(t *testing.T) {
	tt := assert.New(t)

	details := `{
		"asset_code":                        "COP",
		"asset_issuer":                      "GDRW375MAYR46ODGF2WGANQC2RRZL7O246DYHHCGWTV2RE7IHE2QUQLD",
		"asset_type":                        "credit_alphanum4",
		"authorize":                         false,
		"authorize_to_maintain_liabilities": true,
		"trustee":                           "GDRW375MAYR46ODGF2WGANQC2RRZL7O246DYHHCGWTV2RE7IHE2QUQLD",
		"trustor":                           "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3"
	}`

	rsp, err := getJSONResponse(details)
	tt.NoError(err)
	tt.Equal(false, rsp["authorize"])
	tt.Equal(true, rsp["authorize_to_maintain_liabilities"])

	details = `{
		"asset_code":                        "COP",
		"asset_issuer":                      "GDRW375MAYR46ODGF2WGANQC2RRZL7O246DYHHCGWTV2RE7IHE2QUQLD",
		"asset_type":                        "credit_alphanum4",
		"authorize":                         true,
		"authorize_to_maintain_liabilities": true,
		"trustee":                           "GDRW375MAYR46ODGF2WGANQC2RRZL7O246DYHHCGWTV2RE7IHE2QUQLD",
		"trustor":                           "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3"
	}`

	rsp, err = getJSONResponse(details)
	tt.NoError(err)
	tt.Equal(true, rsp["authorize"])
	tt.Equal(true, rsp["authorize_to_maintain_liabilities"])

	details = `{
		"asset_code":                        "COP",
		"asset_issuer":                      "GDRW375MAYR46ODGF2WGANQC2RRZL7O246DYHHCGWTV2RE7IHE2QUQLD",
		"asset_type":                        "credit_alphanum4",
		"authorize":                         false,
		"authorize_to_maintain_liabilities": false,
		"trustee":                           "GDRW375MAYR46ODGF2WGANQC2RRZL7O246DYHHCGWTV2RE7IHE2QUQLD",
		"trustor":                           "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3"
	}`

	rsp, err = getJSONResponse(details)
	tt.NoError(err)
	tt.Equal(false, rsp["authorize"])
	tt.Equal(false, rsp["authorize_to_maintain_liabilities"])
}

func getJSONResponse(details string) (rsp map[string]interface{}, err error) {
	ctx, _ := test.ContextWithLogBuffer()
	transactionRow := history.Transaction{
		TransactionWithoutLedger: history.TransactionWithoutLedger{
			Successful: true,
			MaxFee:     10000,
			FeeCharged: 100,
		},
	}
	operationsRow := history.Operation{
		TransactionSuccessful: true,
		Type:                  xdr.OperationTypeAllowTrust,
		DetailsString:         null.StringFrom(details),
	}
	resource, err := NewOperation(ctx, operationsRow, "", &transactionRow, history.Ledger{})
	if err != nil {
		return
	}

	data, err := json.Marshal(resource)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &rsp)
	return
}

func TestFeeBumpOperation(t *testing.T) {
	ctx, _ := test.ContextWithLogBuffer()
	dest := operations.Base{}
	operationsRow := history.Operation{TransactionSuccessful: true}
	transactionRow := history.Transaction{
		TransactionWithoutLedger: history.TransactionWithoutLedger{
			Successful:           true,
			MaxFee:               123,
			FeeCharged:           100,
			TransactionHash:      "cebb875a00ff6e1383aef0fd251a76f22c1f9ab2a2dffcb077855736ade2659a",
			FeeAccount:           null.StringFrom("GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU"),
			Account:              "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
			NewMaxFee:            null.IntFrom(10000),
			InnerTransactionHash: null.StringFrom("2374e99349b9ef7dba9a5db3339b78fda8f34777b1af33ba468ad5c0df946d4d"),
			Signatures:           []string{"a", "b", "c"},
			InnerSignatures:      []string{"d", "e", "f"},
		},
	}

	assert.NoError(
		t,
		PopulateBaseOperation(
			ctx,
			&dest,
			operationsRow,
			transactionRow.TransactionHash,
			nil,
			history.Ledger{},
		),
	)
	assert.Equal(t, transactionRow.TransactionHash, dest.TransactionHash)

	assert.NoError(
		t,
		PopulateBaseOperation(
			ctx,
			&dest,
			operationsRow,
			transactionRow.InnerTransactionHash.String,
			nil,
			history.Ledger{},
		),
	)
	assert.Equal(t, transactionRow.InnerTransactionHash.String, dest.TransactionHash)

	assert.NoError(
		t,
		PopulateBaseOperation(
			ctx,
			&dest,
			operationsRow,
			transactionRow.TransactionHash,
			&transactionRow,
			history.Ledger{},
		),
	)

	assert.Equal(t, transactionRow.TransactionHash, dest.TransactionHash)
	assert.Equal(t, transactionRow.TransactionHash, dest.Transaction.Hash)
	assert.Equal(t, transactionRow.TransactionHash, dest.Transaction.ID)
	assert.Equal(t, transactionRow.FeeAccount.String, dest.Transaction.FeeAccount)
	assert.Equal(t, transactionRow.Account, dest.Transaction.Account)
	assert.Equal(t, transactionRow.FeeCharged, dest.Transaction.FeeCharged)
	assert.Equal(t, transactionRow.NewMaxFee.Int64, dest.Transaction.MaxFee)
	assert.Equal(t, []string{"a", "b", "c"}, dest.Transaction.Signatures)
	assert.Equal(t, transactionRow.InnerTransactionHash.String, dest.Transaction.InnerTransaction.Hash)
	assert.Equal(t, transactionRow.MaxFee, dest.Transaction.InnerTransaction.MaxFee)
	assert.Equal(t, []string{"d", "e", "f"}, dest.Transaction.InnerTransaction.Signatures)
	assert.Equal(t, transactionRow.TransactionHash, dest.Transaction.FeeBumpTransaction.Hash)
	assert.Equal(t, []string{"a", "b", "c"}, dest.Transaction.FeeBumpTransaction.Signatures)

	assert.NoError(
		t,
		PopulateBaseOperation(
			ctx,
			&dest,
			operationsRow,
			transactionRow.InnerTransactionHash.String,
			&transactionRow,
			history.Ledger{},
		),
	)
	assert.Equal(t, transactionRow.InnerTransactionHash.String, dest.TransactionHash)
	assert.Equal(t, transactionRow.InnerTransactionHash.String, dest.Transaction.Hash)
	assert.Equal(t, transactionRow.InnerTransactionHash.String, dest.Transaction.ID)
	assert.Equal(t, transactionRow.FeeAccount.String, dest.Transaction.FeeAccount)
	assert.Equal(t, transactionRow.Account, dest.Transaction.Account)
	assert.Equal(t, transactionRow.FeeCharged, dest.Transaction.FeeCharged)
	assert.Equal(t, transactionRow.NewMaxFee.Int64, dest.Transaction.MaxFee)
	assert.Equal(t, []string{"d", "e", "f"}, dest.Transaction.Signatures)
	assert.Equal(t, transactionRow.InnerTransactionHash.String, dest.Transaction.InnerTransaction.Hash)
	assert.Equal(t, transactionRow.MaxFee, dest.Transaction.InnerTransaction.MaxFee)
	assert.Equal(t, []string{"d", "e", "f"}, dest.Transaction.InnerTransaction.Signatures)
	assert.Equal(t, transactionRow.TransactionHash, dest.Transaction.FeeBumpTransaction.Hash)
	assert.Equal(t, []string{"a", "b", "c"}, dest.Transaction.FeeBumpTransaction.Signatures)
}
