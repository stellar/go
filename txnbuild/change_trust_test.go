package txnbuild

import (
	"testing"

	"github.com/stellar/go/network"
	"github.com/stretchr/testify/assert"
)

func TestChangeTrustMaxLimit(t *testing.T) {
	kp0 := newKeypair0()
	txSourceAccount := NewSimpleAccount(kp0.Address(), int64(9605939170639898))

	changeTrust := ChangeTrust{
		Line: CreditAsset{"ABCD", kp0.Address()},
	}

	tx := Transaction{
		SourceAccount: &txSourceAccount,
		Operations:    []Operation{&changeTrust},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}
	received := buildSignEncode(t, tx, kp0)

	expected := "AAAAAgAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAGQAIiCNAAAAGwAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAABgAAAAFBQkNEAAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFf/////////8AAAAAAAAAAeoucsUAAABAXp/gGvNtaqn2/gEh4QoNO+LpT3AmyLFDb81INsfdkf70USiBheUc7bzxgZJLVpFy2qw3ucqpQQPi986XFbPsAQ=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestChangeTrustValidateInvalidAsset(t *testing.T) {
	kp0 := newKeypair0()
	txSourceAccount := NewSimpleAccount(kp0.Address(), int64(9605939170639898))

	changeTrust := ChangeTrust{
		Line: NativeAsset{},
	}

	tx := Transaction{
		SourceAccount: &txSourceAccount,
		Operations:    []Operation{&changeTrust},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}
	err := tx.Build()
	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.ChangeTrust operation: Field: Line, Error: native (XLM) asset type is not allowed"
		assert.Contains(t, err.Error(), expected)
	}
}

func TestChangeTrustValidateInvalidLimit(t *testing.T) {
	kp0 := newKeypair0()
	txSourceAccount := NewSimpleAccount(kp0.Address(), int64(9605939170639898))

	changeTrust := ChangeTrust{
		Line:  CreditAsset{"ABCD", kp0.Address()},
		Limit: "-1",
	}

	tx := Transaction{
		SourceAccount: &txSourceAccount,
		Operations:    []Operation{&changeTrust},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}
	err := tx.Build()
	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.ChangeTrust operation: Field: Limit, Error: amount can not be negative"
		assert.Contains(t, err.Error(), expected)
	}
}
