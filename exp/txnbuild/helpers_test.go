package txnbuild

import (
	"testing"

	"github.com/stellar/go/keypair"
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
