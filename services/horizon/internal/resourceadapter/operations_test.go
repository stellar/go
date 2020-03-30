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
		val    bool
		ledger = history.Ledger{}
	)

	dest = operations.Base{}
	row = history.Operation{TransactionSuccessful: nil}

	PopulateBaseOperation(ctx, &dest, row, nil, ledger)
	assert.True(t, dest.TransactionSuccessful)
	assert.Nil(t, dest.Transaction)

	dest = operations.Base{}
	val = true
	row = history.Operation{TransactionSuccessful: &val}

	PopulateBaseOperation(ctx, &dest, row, nil, ledger)
	assert.True(t, dest.TransactionSuccessful)
	assert.Nil(t, dest.Transaction)

	dest = operations.Base{}
	val = false
	row = history.Operation{TransactionSuccessful: &val}

	PopulateBaseOperation(ctx, &dest, row, nil, ledger)
	assert.False(t, dest.TransactionSuccessful)
	assert.Nil(t, dest.Transaction)
}

// TestPopulateOperation_WithTransaction tests PopulateBaseOperation when passing both an operation and a transaction.
func TestPopulateOperation_WithTransaction(t *testing.T) {
	ctx, _ := test.ContextWithLogBuffer()

	var (
		dest           operations.Base
		operationsRow  history.Operation
		val            bool
		ledger         = history.Ledger{}
		transactionRow history.Transaction
	)

	dest = operations.Base{}
	val = true
	operationsRow = history.Operation{TransactionSuccessful: &val}
	transactionRow = history.Transaction{Successful: &val, MaxFee: 10000, FeeCharged: 100}

	PopulateBaseOperation(ctx, &dest, operationsRow, &transactionRow, ledger)
	assert.True(t, dest.TransactionSuccessful)
	assert.True(t, dest.Transaction.Successful)
	assert.Equal(t, int32(100), dest.Transaction.FeeCharged)
	assert.Equal(t, int32(10000), dest.Transaction.MaxFee)
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
	txSuccessful := true
	transactionRow := history.Transaction{Successful: &txSuccessful, MaxFee: 10000, FeeCharged: 100}
	operationsRow := history.Operation{
		TransactionSuccessful: &txSuccessful,
		Type:                  xdr.OperationTypeAllowTrust,
		DetailsString:         null.StringFrom(details),
	}
	resource, err := NewOperation(ctx, operationsRow, &transactionRow, history.Ledger{})
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
