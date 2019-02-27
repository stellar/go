package txnbuild

import (
	"testing"

	"github.com/stellar/go/clients/horizon"
	"github.com/stretchr/testify/assert"
)

func TestInflation(t *testing.T) {
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
	assert.Nil(t, err)

	err = tx.Sign(secretSeed)
	assert.Nil(t, err)

	txeBase64, err := tx.Base64()
	assert.Nil(t, err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAaAAAAAAAAAAAAAAABAAAAAAAAAAkAAAAAAAAAAeoucsUAAABAWqznvTxLfn6Q+zIloGmLDXCJQWsFPlfIf/EVFF+FfpL/gNbsvTC/U2G/ZtxMTgvqTLsBJfZAailGvPS04rfYCw=="

	assert.Equal(t, expected, txeBase64, "Base 64 XDR should match")
}

func TestCreateAccount(t *testing.T) {
	var err error

	secretSeed := "SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R"
	sourceAddress := "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3"
	sourceAccount := horizon.Account{
		HistoryAccount: horizon.HistoryAccount{ID: sourceAddress},
		Sequence:       "9605939170639897",
	}

	createAccount := CreateAccount{
		Destination: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:      "10",
		Asset:       "native",
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&createAccount},
	}

	err = tx.Build()
	assert.Nil(t, err)

	err = tx.Sign(secretSeed)
	assert.Nil(t, err)

	txeBase64, err := tx.Base64()
	assert.Nil(t, err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAaAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAhODe2rwbSS+e+giFk8xgmxj70pVXzEADo3GG0rEhdlQAAAAABfXhAAAAAAAAAAAB6i5yxQAAAEBa4swhXSxQ2SYXoT0FcwIrrslFrv/Q/pnXK2+f6XigqjxW0yjNQwIrpVZuNz4zNGXB3DULxyYkUi8wDwwbiKIB"

	assert.Equal(t, expected, txeBase64, "Base 64 XDR should match")
}

func TestPayment(t *testing.T) {
	var err error

	secretSeed := "SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R"
	sourceAddress := "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3"
	sourceAccount := horizon.Account{
		HistoryAccount: horizon.HistoryAccount{ID: sourceAddress},
		Sequence:       "9605939170639898",
	}

	payment := Payment{
		Destination: "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
		Amount:      "10",
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&payment},
	}

	err = tx.Build()
	assert.Nil(t, err)

	err = tx.Sign(secretSeed)
	assert.Nil(t, err)

	txeBase64, err := tx.Base64()
	assert.Nil(t, err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAbAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAfhHLNNY19eGrAtSgLD3VpaRm2AjNjxIBWQg9zS4VWZgAAAAAAAAAAAX14QAAAAAAAAAAAeoucsUAAABA5rSL7gy8OGiMq2Rocvv6l6HwOdePwhIMw2aJ2j5mVumAmeADjMeeCcGQIj3A7bISo6eWoF49w3qcd7uBS4j6AQ=="

	assert.Equal(t, expected, txeBase64, "Base 64 XDR should match")
}

func TestBumpSequence(t *testing.T) {
	var err error

	secretSeed := "SBMSVD4KKELKGZXHBUQTIROWUAPQASDX7KEJITARP4VMZ6KLUHOGPTYW"
	sourceAddress := "GAS4V4O2B7DW5T7IQRPEEVCRXMDZESKISR7DVIGKZQYYV3OSQ5SH5LVP"
	sourceAccount := horizon.Account{
		HistoryAccount: horizon.HistoryAccount{ID: sourceAddress},
		Sequence:       "9606132444168199",
	}

	BumpSequence := BumpSequence{
		BumpTo: 9606132444168300,
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&BumpSequence},
	}

	err = tx.Build()
	assert.Nil(t, err)

	err = tx.Sign(secretSeed)
	assert.Nil(t, err)

	txeBase64, err := tx.Base64()
	assert.Nil(t, err)

	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAiILoAAAAIAAAAAAAAAAAAAAABAAAAAAAAAAsAIiC6AAAAbAAAAAAAAAAB0odkfgAAAEDLsgDc3tPETqlKxVMF16UePDbSXQ1X0i5b3U3DRHDEchU91YwsDb4oMZrCj0mwKhkiXzCUyg9pPmUG/vKtQVQD"

	assert.Equal(t, expected, txeBase64, "Base 64 XDR should match")
}
