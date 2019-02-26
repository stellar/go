package txnbuild

import (
	"testing"

	"github.com/stellar/go/clients/horizon"
	"github.com/stretchr/testify/assert"
)

func TestInflation(t *testing.T) {
	assert := assert.New(t)
	var err error

	secretSeed := "SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R"
	sourceAddress := "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3"
	sourceAccount := horizon.Account{
		HistoryAccount: horizon.HistoryAccount{ID: sourceAddress},
		Sequence:       "9605939170639897",
	}

	inflation := Inflation{}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&inflation},
	}

	err = tx.Build()
	assert.Nil(err)

	err = tx.Sign(secretSeed)
	assert.Nil(err)

	txeBase64, err := tx.Base64()
	assert.Nil(err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAaAAAAAAAAAAAAAAABAAAAAAAAAAkAAAAAAAAAAeoucsUAAABAWqznvTxLfn6Q+zIloGmLDXCJQWsFPlfIf/EVFF+FfpL/gNbsvTC/U2G/ZtxMTgvqTLsBJfZAailGvPS04rfYCw=="

	assert.Equal(expected, txeBase64, "Base 64 XDR should match")
}
