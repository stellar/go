package txnbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathPaymentValidateSendAsset(t *testing.T) {
	kp0 := newKeypair0()
	kp2 := newKeypair2()
	sourceAccount := NewSimpleAccount(kp2.Address(), int64(187316408680450))

	abcdAsset := CreditAsset{"ABCD", kp0.Address()}
	pathPayment := PathPayment{
		SendAsset:   CreditAsset{"ABCD", ""},
		SendMax:     "10",
		Destination: kp2.Address(),
		DestAsset:   NativeAsset{},
		DestAmount:  "1",
		Path:        []Asset{abcdAsset},
	}

	_, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: false,
			Operations:           []Operation{&pathPayment},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)

	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.PathPaymentStrictReceive operation: Field: SendAsset, Error: asset issuer: public key is undefined"
		assert.Contains(t, err.Error(), expected)
	}
}

func TestPathPaymentValidateDestAsset(t *testing.T) {
	kp0 := newKeypair0()
	kp2 := newKeypair2()
	sourceAccount := NewSimpleAccount(kp2.Address(), int64(187316408680450))

	abcdAsset := CreditAsset{"ABCD", kp0.Address()}
	pathPayment := PathPayment{
		SendAsset:   NativeAsset{},
		SendMax:     "10",
		Destination: kp2.Address(),
		DestAsset:   CreditAsset{"", kp0.Address()},
		DestAmount:  "1",
		Path:        []Asset{abcdAsset},
	}

	_, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: false,
			Operations:           []Operation{&pathPayment},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.PathPaymentStrictReceive operation: Field: DestAsset, Error: asset code length must be between 1 and 12 characters"
		assert.Contains(t, err.Error(), expected)
	}
}

func TestPathPaymentValidateDestination(t *testing.T) {
	kp0 := newKeypair0()
	kp2 := newKeypair2()
	sourceAccount := NewSimpleAccount(kp2.Address(), int64(187316408680450))

	abcdAsset := CreditAsset{"ABCD", kp0.Address()}
	pathPayment := PathPayment{
		SendAsset:   NativeAsset{},
		SendMax:     "10",
		Destination: "SASND3NRUY5K43PN3H3HOP5JNTIDXJFLOKKNSCZQQAFBRSEIRD5OJKXZ",
		DestAsset:   CreditAsset{"ABCD", kp0.Address()},
		DestAmount:  "1",
		Path:        []Asset{abcdAsset},
	}

	_, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: false,
			Operations:           []Operation{&pathPayment},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.PathPaymentStrictReceive operation: Field: Destination"
		assert.Contains(t, err.Error(), expected)
	}
}

func TestPathPaymentValidateSendMax(t *testing.T) {
	kp0 := newKeypair0()
	kp2 := newKeypair2()
	sourceAccount := NewSimpleAccount(kp2.Address(), int64(187316408680450))

	abcdAsset := CreditAsset{"ABCD", kp0.Address()}
	pathPayment := PathPayment{
		SendAsset:   NativeAsset{},
		SendMax:     "abc",
		Destination: kp2.Address(),
		DestAsset:   CreditAsset{"ABCD", kp0.Address()},
		DestAmount:  "1",
		Path:        []Asset{abcdAsset},
	}

	_, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: false,
			Operations:           []Operation{&pathPayment},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.PathPaymentStrictReceive operation: Field: SendMax, Error: invalid amount format: abc"
		assert.Contains(t, err.Error(), expected)
	}
}

func TestPathPaymentValidateDestAmount(t *testing.T) {
	kp0 := newKeypair0()
	kp2 := newKeypair2()
	sourceAccount := NewSimpleAccount(kp2.Address(), int64(187316408680450))

	abcdAsset := CreditAsset{"ABCD", kp0.Address()}
	pathPayment := PathPayment{
		SendAsset:   NativeAsset{},
		SendMax:     "10",
		Destination: kp2.Address(),
		DestAsset:   CreditAsset{"ABCD", kp0.Address()},
		DestAmount:  "-1",
		Path:        []Asset{abcdAsset},
	}

	_, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: false,
			Operations:           []Operation{&pathPayment},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.PathPaymentStrictReceive operation: Field: DestAmount, Error: amount can not be negative"
		assert.Contains(t, err.Error(), expected)
	}
}
