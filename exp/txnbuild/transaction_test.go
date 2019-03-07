package txnbuild

import (
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stretchr/testify/assert"
)

func newKeypair0() *keypair.Full {
	return newKeypair("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
}

func newKeypair1() *keypair.Full {
	return newKeypair("SBMSVD4KKELKGZXHBUQTIROWUAPQASDX7KEJITARP4VMZ6KLUHOGPTYW")
}

func newKeypair(seed string) *keypair.Full {
	myKeypair, _ := keypair.Parse(seed)
	return myKeypair.(*keypair.Full)
}

func buildSignEncode(tx Transaction, kp *keypair.Full, t *testing.T) (txeBase64 string) {
	var err error
	err = tx.Build()
	assert.Nil(t, err)

	err = tx.Sign(kp)
	assert.Nil(t, err)

	txeBase64, err = tx.Base64()
	assert.Nil(t, err)

	return
}

func TestInflation(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := Account{
		ID:             kp0.Address(),
		SequenceNumber: 9605939170639897,
	}

	inflation := Inflation{}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&inflation},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAaAAAAAAAAAAAAAAABAAAAAAAAAAkAAAAAAAAAAeoucsUAAABAWqznvTxLfn6Q+zIloGmLDXCJQWsFPlfIf/EVFF+FfpL/gNbsvTC/U2G/ZtxMTgvqTLsBJfZAailGvPS04rfYCw=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestCreateAccount(t *testing.T) {
	kp0 := newKeypair0()

	sourceAccount := Account{
		ID:             kp0.Address(),
		SequenceNumber: 9605939170639897,
	}

	createAccount := CreateAccount{
		Destination: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:      "10",
		Asset:       "native",
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&createAccount},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAaAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAhODe2rwbSS+e+giFk8xgmxj70pVXzEADo3GG0rEhdlQAAAAABfXhAAAAAAAAAAAB6i5yxQAAAEBa4swhXSxQ2SYXoT0FcwIrrslFrv/Q/pnXK2+f6XigqjxW0yjNQwIrpVZuNz4zNGXB3DULxyYkUi8wDwwbiKIB"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestPayment(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := Account{
		ID:             kp0.Address(),
		SequenceNumber: 9605939170639898,
	}

	payment := Payment{
		Destination: "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
		Amount:      "10",
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&payment},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAbAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAfhHLNNY19eGrAtSgLD3VpaRm2AjNjxIBWQg9zS4VWZgAAAAAAAAAAAX14QAAAAAAAAAAAeoucsUAAABA5rSL7gy8OGiMq2Rocvv6l6HwOdePwhIMw2aJ2j5mVumAmeADjMeeCcGQIj3A7bISo6eWoF49w3qcd7uBS4j6AQ=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestBumpSequence(t *testing.T) {
	kp1 := newKeypair1()
	sourceAccount := Account{
		ID:             kp1.Address(),
		SequenceNumber: 9606132444168199,
	}

	bumpSequence := BumpSequence{
		BumpTo: 9606132444168300,
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&bumpSequence},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp1, t)
	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAiILoAAAAIAAAAAAAAAAAAAAABAAAAAAAAAAsAIiC6AAAAbAAAAAAAAAAB0odkfgAAAEDLsgDc3tPETqlKxVMF16UePDbSXQ1X0i5b3U3DRHDEchU91YwsDb4oMZrCj0mwKhkiXzCUyg9pPmUG/vKtQVQD"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestMultipleOperations(t *testing.T) {
	kp1 := newKeypair1()
	sourceAccount := Account{
		ID:             kp1.Address(),
		SequenceNumber: 9606132444168199,
	}

	inflation := Inflation{}
	bumpSequence := BumpSequence{
		BumpTo: 9606132444168300,
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&inflation, &bumpSequence},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp1, t)
	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAyAAiILoAAAAIAAAAAAAAAAAAAAACAAAAAAAAAAkAAAAAAAAACwAiILoAAABsAAAAAAAAAAHSh2R+AAAAQGx5xAPuF3rH3/KSHXduYYvE/Qw4CAseF2F0oSacIYi8e320OW07lr9VF8XEcDqMSVNhkFopoh5P0ZSixcTxyQI="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}
