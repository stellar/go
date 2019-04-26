package txnbuild

import (
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/assert"
)

func newKeypair0() *keypair.Full {
	// Address: GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3
	return newKeypair("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
}

func newKeypair1() *keypair.Full {
	// Address: GAS4V4O2B7DW5T7IQRPEEVCRXMDZESKISR7DVIGKZQYYV3OSQ5SH5LVP
	return newKeypair("SBMSVD4KKELKGZXHBUQTIROWUAPQASDX7KEJITARP4VMZ6KLUHOGPTYW")
}

func newKeypair2() *keypair.Full {
	// Address: GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H
	return newKeypair("SBZVMB74Z76QZ3ZOY7UTDFYKMEGKW5XFJEB6PFKBF4UYSSWHG4EDH7PY")
}

func newKeypair(seed string) *keypair.Full {
	myKeypair, _ := keypair.Parse(seed)
	return myKeypair.(*keypair.Full)
}

func buildSignEncode(tx Transaction, kp *keypair.Full, t *testing.T) (txeBase64 string) {
	var err error
	err = tx.Build()
	assert.NoError(t, err)

	err = tx.Sign(kp)
	assert.NoError(t, err)

	txeBase64, err = tx.Base64()
	assert.NoError(t, err)

	return
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
