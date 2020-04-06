package resourceadapter

import (
	"github.com/guregu/null"
	"testing"

	. "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/test"
	"github.com/stretchr/testify/assert"
)

// TestPopulateTransaction_Successful tests transaction object population.
func TestPopulateTransaction_Successful(t *testing.T) {
	ctx, _ := test.ContextWithLogBuffer()

	var (
		dest Transaction
		row  history.Transaction
	)

	dest = Transaction{}
	row = history.Transaction{Successful: true}

	PopulateTransaction(ctx, row.TransactionHash, &dest, row)
	assert.True(t, dest.Successful)

	dest = Transaction{}
	row = history.Transaction{Successful: false}

	PopulateTransaction(ctx, row.TransactionHash, &dest, row)
	assert.False(t, dest.Successful)
}

// TestPopulateTransaction_Fee tests transaction object population.
func TestPopulateTransaction_Fee(t *testing.T) {
	ctx, _ := test.ContextWithLogBuffer()

	var (
		dest Transaction
		row  history.Transaction
	)

	dest = Transaction{}
	row = history.Transaction{MaxFee: 10000, FeeCharged: 100}

	PopulateTransaction(ctx, row.TransactionHash, &dest, row)
	assert.Equal(t, int64(100), dest.FeeCharged)
	assert.Equal(t, int64(10000), dest.MaxFee)
}

func TestFeeBumpTransaction(t *testing.T) {
	ctx, _ := test.ContextWithLogBuffer()
	dest := Transaction{}
	row := history.Transaction{
		MaxFee:               123,
		FeeCharged:           100,
		TransactionHash:      "cebb875a00ff6e1383aef0fd251a76f22c1f9ab2a2dffcb077855736ade2659a",
		SignatureString:      "a,b,c",
		FeeAccount:           null.StringFrom("GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU"),
		Account:              "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		NewMaxFee:            null.IntFrom(10000),
		InnerSignatureString: null.StringFrom("d,e,f"),
		InnerTransactionHash: null.StringFrom("2374e99349b9ef7dba9a5db3339b78fda8f34777b1af33ba468ad5c0df946d4d"),
	}

	PopulateTransaction(ctx, row.TransactionHash, &dest, row)
	assert.Equal(t, row.TransactionHash, dest.Hash)
	assert.Equal(t, row.TransactionHash, dest.ID)
	assert.Equal(t, row.FeeAccount.String, dest.FeeAccount)
	assert.Equal(t, row.Account, dest.Account)
	assert.Equal(t, row.FeeCharged, dest.FeeCharged)
	assert.Equal(t, row.NewMaxFee.Int64, dest.MaxFee)
	assert.Equal(t, []string{"a", "b", "c"}, dest.Signatures)
	assert.Equal(t, row.InnerTransactionHash.String, dest.InnerTransaction.Hash)
	assert.Equal(t, row.MaxFee, dest.InnerTransaction.MaxFee)
	assert.Equal(t, []string{"d", "e", "f"}, dest.InnerTransaction.Signatures)
	assert.Equal(t, row.TransactionHash, dest.FeeBumpTransaction.Hash)
	assert.Equal(t, []string{"a", "b", "c"}, dest.FeeBumpTransaction.Signatures)
	assert.Equal(t, "/transactions/"+row.TransactionHash, dest.Links.Transaction.Href)

	PopulateTransaction(ctx, row.InnerTransactionHash.String, &dest, row)
	assert.Equal(t, row.InnerTransactionHash.String, dest.Hash)
	assert.Equal(t, row.InnerTransactionHash.String, dest.ID)
	assert.Equal(t, row.FeeAccount.String, dest.FeeAccount)
	assert.Equal(t, row.Account, dest.Account)
	assert.Equal(t, row.FeeCharged, dest.FeeCharged)
	assert.Equal(t, row.NewMaxFee.Int64, dest.MaxFee)
	assert.Equal(t, []string{"d", "e", "f"}, dest.Signatures)
	assert.Equal(t, row.InnerTransactionHash.String, dest.InnerTransaction.Hash)
	assert.Equal(t, row.MaxFee, dest.InnerTransaction.MaxFee)
	assert.Equal(t, []string{"d", "e", "f"}, dest.InnerTransaction.Signatures)
	assert.Equal(t, row.TransactionHash, dest.FeeBumpTransaction.Hash)
	assert.Equal(t, []string{"a", "b", "c"}, dest.FeeBumpTransaction.Signatures)
	assert.Equal(t, "/transactions/"+row.InnerTransactionHash.String, dest.Links.Transaction.Href)
}
