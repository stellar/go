package txnbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClawbackValidateFrom(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(9605939170639898))

	clawback := Clawback{
		From:   "",
		Amount: "10",
		Asset:  CreditAsset{"", kp0.Address()},
	}

	_, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: false,
			Operations:           []Operation{&clawback},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.Clawback operation: Field: From"
		assert.Contains(t, err.Error(), expected)
	}
}

func TestClawbackValidateAmount(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(9605939170639898))

	clawback := Clawback{
		From:   "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
		Amount: "ten",
		Asset:  CreditAsset{"", kp0.Address()},
	}

	_, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: false,
			Operations:           []Operation{&clawback},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.Clawback operation: Field: Amount, Error: invalid amount format: ten"
		assert.Contains(t, err.Error(), expected)
	}
}

func TestClawbackValidateAsset(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(9605939170639898))

	clawback := Clawback{
		From:   "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
		Amount: "10",
		Asset:  CreditAsset{},
	}

	_, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: false,
			Operations:           []Operation{&clawback},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.Clawback operation: Field: Asset, Error: asset code length must be between 1 and 12 characters"
		assert.Contains(t, err.Error(), expected)
	}
}

func TestClawbackRoundTrip(t *testing.T) {
	clawback := Clawback{
		From:   "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
		Amount: "10.0000000",
		Asset:  CreditAsset{"USD", ""},
	}

	testOperationsMarshallingRoundtrip(t, []Operation{&clawback})
}
