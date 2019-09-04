package txnbuild

import (
	"testing"

	"github.com/stellar/go/network"
	"github.com/stretchr/testify/assert"
)

func TestCreatePassiveSellOfferValidateBuyingAsset(t *testing.T) {
	// kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp1.Address(), int64(41137196761100))

	createPassiveOffer := CreatePassiveSellOffer{
		Selling: NativeAsset{},
		Buying:  CreditAsset{"ABCD", ""},
		Amount:  "10",
		Price:   "1.0",
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&createPassiveOffer},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	err := tx.Build()
	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.CreatePassiveSellOffer struct: Field: Buying, Error: asset issuer is not a valid stellar public key"
		assert.Contains(t, err.Error(), expected)
	}
}

func TestCreatePassiveSellOfferValidateSellingAsset(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp1.Address(), int64(41137196761100))

	createPassiveOffer := CreatePassiveSellOffer{
		Selling: CreditAsset{"ABCD0123456789", kp0.Address()},
		Buying:  NativeAsset{},
		Amount:  "10",
		Price:   "1.0",
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&createPassiveOffer},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	err := tx.Build()
	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.CreatePassiveSellOffer struct: Field: Selling, Error: asset code length must be between 1 and 12 characters"
		assert.Contains(t, err.Error(), expected)
	}
}

func TestCreatePassiveSellOfferValidateAmount(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp1.Address(), int64(41137196761100))

	createPassiveOffer := CreatePassiveSellOffer{
		Selling: CreditAsset{"ABCD", kp0.Address()},
		Buying:  NativeAsset{},
		Amount:  "-3",
		Price:   "1.0",
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&createPassiveOffer},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	err := tx.Build()
	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.CreatePassiveSellOffer struct: Field: Amount, Error: value should be positve or zero"
		assert.Contains(t, err.Error(), expected)
	}
}

func TestCreatePassiveSellOfferValidatePrice(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp1.Address(), int64(41137196761100))

	createPassiveOffer := CreatePassiveSellOffer{
		Selling: CreditAsset{"ABCD", kp0.Address()},
		Buying:  NativeAsset{},
		Amount:  "3",
		Price:   "-1.0",
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&createPassiveOffer},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	err := tx.Build()
	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.CreatePassiveSellOffer struct: Field: Price, Error: value should be positve or zero"
		assert.Contains(t, err.Error(), expected)
	}
}
