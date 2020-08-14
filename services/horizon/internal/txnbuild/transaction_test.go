package txnbuild

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMissingSourceAccount(t *testing.T) {
	_, err := NewTransaction(TransactionParams{})
	assert.EqualError(t, err, "transaction has no source account")
}

func TestIncrementSequenceNum(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), 1)
	inflation := Inflation{}

	_, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&inflation},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), sourceAccount.Sequence)

	_, err = NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&inflation},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), sourceAccount.Sequence)

	_, err = NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: false,
			Operations:           []Operation{&inflation},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), sourceAccount.Sequence)

	_, err = NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: false,
			Operations:           []Operation{&inflation},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), sourceAccount.Sequence)
}

func TestFeeNoOperations(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), 5938436531814403)

	_, err := NewTransaction(
		TransactionParams{
			SourceAccount: &sourceAccount,
			Operations:    []Operation{},
			BaseFee:       MinBaseFee,
			Timebounds:    NewInfiniteTimeout(),
		},
	)
	assert.EqualError(t, err, "transaction has no operations")
}

func TestInflation(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(3556091187167235))

	inflation := Inflation{}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&inflation},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0,
	)
	assert.NoError(t, err)

	// https://www.stellar.org/laboratory/#xdr-viewer?input=AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAMoj8AAAAEAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAJAAAAAAAAAAHqLnLFAAAAQP3NHWXvzKIHB3%2BjjhHITdc%2FtBPntWYj3SoTjpON%2BdxjKqU5ohFamSHeqi5ONXkhE9Uajr5sVZXjQfUcTTzsWAA%3D&type=TransactionEnvelope
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAMoj8AAAAEAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAJAAAAAAAAAAHqLnLFAAAAQP3NHWXvzKIHB3+jjhHITdc/tBPntWYj3SoTjpON+dxjKqU5ohFamSHeqi5ONXkhE9Uajr5sVZXjQfUcTTzsWAA="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestCreateAccount(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(9605939170639897))

	createAccount := CreateAccount{
		Destination: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:      "10",
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&createAccount},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0,
	)
	assert.NoError(t, err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAaAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAITg3tq8G0kvnvoIhZPMYJsY+9KVV8xAA6NxhtKxIXZUAAAAAAX14QAAAAAAAAAAAeoucsUAAABAHsyMojA0Q5MiNsR5X5AiNpCn9mlXmqluRsNpTniCR91M4U5TFmrrqVNLkU58/l+Y8hUPwidDTRSzLZKbMUL/Bw=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestPayment(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(9605939170639898))

	payment := Payment{
		Destination: "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
		Amount:      "10",
		Asset:       NativeAsset{},
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&payment},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0,
	)
	assert.NoError(t, err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAbAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAABAAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAAAAAAAAF9eEAAAAAAAAAAAHqLnLFAAAAQNcGQpjNOFCLf9eEmobN+H8SNoDH/jMrfEFPX8kM212ST+TGfirEdXH77GJXvaWplfGKmE3B+UDwLuYLwO+KbQQ="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestPaymentFailsIfNoAssetSpecified(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(9605939170639898))

	payment := Payment{
		Destination: "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
		Amount:      "10",
	}

	_, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&payment},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	expectedErrMsg := "validation failed for *txnbuild.Payment operation: Field: Asset, Error: asset is undefined"
	require.EqualError(t, err, expectedErrMsg, "An asset is required")
}

func TestBumpSequence(t *testing.T) {
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp1.Address(), int64(9606132444168199))

	bumpSequence := BumpSequence{
		BumpTo: 9606132444168300,
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&bumpSequence},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp1,
	)
	assert.NoError(t, err)

	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAiILoAAAAIAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAALACIgugAAAGwAAAAAAAAAAdKHZH4AAABAndjSSWeACpbr0ROAEK6jw5CzHiL/rCDpa6AO05+raHDowSUJBckkwlEuCjbBoO/A06tZNRT1Per3liTQrc8fCg=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestAccountMerge(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484298))

	accountMerge := AccountMerge{
		Destination: "GAS4V4O2B7DW5T7IQRPEEVCRXMDZESKISR7DVIGKZQYYV3OSQ5SH5LVP",
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&accountMerge},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0,
	)
	assert.NoError(t, err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAALAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAIAAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAAAAAAAHqLnLFAAAAQJ/UcOgE64+GQpwv0uXXa2jrKtFdmDsyZ6ZZ/udxryPS8cNCm2L784ixPYM4XRgkoQCdxC3YK8n5x5+CXLzrrwA="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestManageData(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(3556091187167235))

	manageData := ManageData{
		Name:  "Fruit preference",
		Value: []byte("Apple"),
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&manageData},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0,
	)
	assert.NoError(t, err)

	// https://www.stellar.org/laboratory/#txsigner?xdr=AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAMoj8AAAAEAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAKAAAAEEZydWl0IHByZWZlcmVuY2UAAAABAAAABUFwcGxlAAAAAAAAAAAAAAA%3D
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAMoj8AAAAEAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAKAAAAEEZydWl0IHByZWZlcmVuY2UAAAABAAAABUFwcGxlAAAAAAAAAAAAAAHqLnLFAAAAQO1ELJBEoqBDyIsS7uSJwe1LOimV/E+09MyF1G/+yrxSggFVPEjD5LXcm/6POze3IsMuIYJU1et5Q2Vt9f73zQo="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestManageDataRemoveDataEntry(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484309))

	manageData := ManageData{
		Name: "Fruit preference",
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&manageData},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0,
	)
	assert.NoError(t, err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAWAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAKAAAAEEZydWl0IHByZWZlcmVuY2UAAAAAAAAAAAAAAAHqLnLFAAAAQB8rkFZgtffUTdCASzwJ3jRcMCzHpVbbuFbye7Ki2dLao6u5d2aSzz3M2ugNJjNFMfSu3io9adCqwVKKjk0UJQA="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsInflationDestination(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484315))

	setOptions := SetOptions{
		InflationDestination: NewInflationDestination(kp1.Address()),
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&setOptions},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0,
	)
	assert.NoError(t, err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAcAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAFAAAAAQAAAAAlyvHaD8duz+iEXkJUUbsHkklIlH46oMrMMYrt0odkfgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHqLnLFAAAAQB0RLe9DjdHzLM22whFja3ZT97L/818lvWpk5EOTETr9lmDH7/A0/EAzeCkTBzZMCi3C6pV1PrGBr0NJdRrPowg="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsSetFlags(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484318))

	setOptions := SetOptions{
		SetFlags: []AccountFlag{AuthRequired, AuthRevocable},
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&setOptions},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0,
	)
	assert.NoError(t, err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAfAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAAAAAABAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB6i5yxQAAAECfYTppxtp1A2zSbb6VzkOkyk9D/7xjaXRxR+ZIqgdK3lWkHQRkjyVBj2yaI61J3trdp7CswImptjkjLprt0WIO"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsClearFlags(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484319))

	setOptions := SetOptions{
		ClearFlags: []AccountFlag{AuthRequired, AuthRevocable},
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&setOptions},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0,
	)
	assert.NoError(t, err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAgAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAEAAAADAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB6i5yxQAAAEANXPAN+RgvqjGF0kJ6MyNTiMnWaELw5vYNwxhv8+mi3KmGWMzojCxcmMAqni0zBMsEjl9z7H8JT9x05OlQ9nsD"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsMasterWeight(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484320))

	setOptions := SetOptions{
		MasterWeight: NewThreshold(10),
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&setOptions},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0,
	)
	assert.NoError(t, err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAhAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAAAAAAAAAAAAQAAAAoAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB6i5yxQAAAECIxH2W4XZ5fMsG658hdIEys2nlVSAK1FEjT5GADF6sWEThGFc+Wrmlw6GwKn6ZNAmxVULEgircjQx48aYSgFYD"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsThresholds(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484322))

	setOptions := SetOptions{
		LowThreshold:    NewThreshold(1),
		MediumThreshold: NewThreshold(2),
		HighThreshold:   NewThreshold(2),
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&setOptions},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0,
	)
	assert.NoError(t, err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAjAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAABAAAAAQAAAAIAAAABAAAAAgAAAAAAAAAAAAAAAAAAAAHqLnLFAAAAQFwRcFbzEtxoxZOtWlOQld3nURHZugNj5faEncpv0X/dcrfiQVU7k3fkTYDskiVExFiq78CBsYAr0uuvfH61IQs="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsHomeDomain(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484325))

	setOptions := SetOptions{
		HomeDomain: NewHomeDomain("LovelyLumensLookLuminous.com"),
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&setOptions},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0,
	)
	assert.NoError(t, err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAmAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAcTG92ZWx5THVtZW5zTG9va0x1bWlub3VzLmNvbQAAAAAAAAAAAAAAAeoucsUAAABAtC4HZzvRfyphRg5jjmz5jzBn86SANXCZS59GejRE8L1uCOxgXSEVoh1b+UetUEi7JN/n1ECBEVJrXgj0c34eBg=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsHomeDomainTooLong(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484323))

	setOptions := SetOptions{
		HomeDomain: NewHomeDomain("LovelyLumensLookLuminousLately.com"),
	}

	_, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&setOptions},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)

	assert.Error(t, err, "A validation error was expected (home domain > 32 chars)")
}

func TestSetOptionsSigner(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484325))

	setOptions := SetOptions{
		Signer: &Signer{Address: kp1.Address(), Weight: Threshold(4)},
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&setOptions},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0,
	)
	assert.NoError(t, err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAmAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAABAAAAAAAAAAB6i5yxQAAAEBfgmUK+wNj8ROz78Sg0rQ2s7lmtvA4r5epHkqc9yoxLDr/GSkmgWneVqoKNxWF0JB9L+Gql1+f8M8p1McF4MsB"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestMultipleOperations(t *testing.T) {
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp1.Address(), int64(9606132444168199))

	inflation := Inflation{}
	bumpSequence := BumpSequence{
		BumpTo: 9606132444168300,
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&inflation, &bumpSequence},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp1,
	)
	assert.NoError(t, err)

	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAyAAiILoAAAAIAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAgAAAAAAAAAJAAAAAAAAAAsAIiC6AAAAbAAAAAAAAAAB0odkfgAAAEDmf3Ag2Hw5NdlvzJpph4Km+aNKy8kfzS1EAhIVdKJwUnMVWhOpfdXSh/aekEVdoxXh2+ioocrxdtkWAZfS3sMF"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestChangeTrust(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484348))

	changeTrust := ChangeTrust{
		Line:  CreditAsset{"ABCD", kp1.Address()},
		Limit: "10",
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&changeTrust},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0,
	)
	assert.NoError(t, err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAA9AAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAGAAAAAUFCQ0QAAAAAJcrx2g/Hbs/ohF5CVFG7B5JJSJR+OqDKzDGK7dKHZH4AAAAABfXhAAAAAAAAAAAB6i5yxQAAAED7YSd1VdewEdtEURAYuyCy8dWbzALEf1vJn88/gCER4CNdIvojOEafJEhYhzZJhdG7oa+95UjfI9vMJO8qdWMK"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestChangeTrustNativeAssetNotAllowed(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484348))

	changeTrust := ChangeTrust{
		Line:  NativeAsset{},
		Limit: "10",
	}

	_, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&changeTrust},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)

	expectedErrMsg := "validation failed for *txnbuild.ChangeTrust operation: Field: Line, Error: native (XLM) asset type is not allowed"
	require.EqualError(t, err, expectedErrMsg, "No trustlines for native assets")
}

func TestChangeTrustDeleteTrustline(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484354))

	issuedAsset := CreditAsset{"ABCD", kp1.Address()}
	removeTrust := RemoveTrustlineOp(issuedAsset)

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&removeTrust},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0,
	)
	assert.NoError(t, err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAABDAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAGAAAAAUFCQ0QAAAAAJcrx2g/Hbs/ohF5CVFG7B5JJSJR+OqDKzDGK7dKHZH4AAAAAAAAAAAAAAAAAAAAB6i5yxQAAAECgd2wkK35civvf6NKpsSFDyKpdyo/cs7wL+RYfZ2BCP7eGrUUpu2GfQFtf/Hm6aBwT6nJ+dONTSPXnyp7Dq18L"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestAllowTrust(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484366))

	issuedAsset := CreditAsset{"ABCD", kp1.Address()}
	allowTrust := AllowTrust{
		Trustor:   kp1.Address(),
		Type:      issuedAsset,
		Authorize: true,
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&allowTrust},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0,
	)
	assert.NoError(t, err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAABPAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAHAAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAAUFCQ0QAAAABAAAAAAAAAAHqLnLFAAAAQGGBSKitYxpHNMaVVOE2CIylWFJgwqxjhwnIvWauSSkLapntD18G1pMahLbs8Lqcr3+cEs5WjLI4eBhy6WiJhAk="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestAllowTrustNoIssuer(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484366))

	issuedAsset := CreditAsset{Code: "XYZ"}
	allowTrust := AllowTrust{
		Trustor:   kp1.Address(),
		Type:      issuedAsset,
		Authorize: true,
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&allowTrust},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0,
	)
	assert.NoError(t, err)

	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAABPAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAHAAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAAVhZWgAAAAABAAAAAAAAAAHqLnLFAAAAQO8mcsi/+RObrKto8tABtN8RwUi6101FqBDTwqMQp4hNuujw+SGEFaBCYLNw/u40DHFRQoBNi6zcBKbBSg+gVwE="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestManageSellOfferNewOffer(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp1.Address(), int64(41137196761092))

	selling := NativeAsset{}
	buying := CreditAsset{"ABCD", kp0.Address()}
	sellAmount := "100"
	price := "0.01"
	createOffer, err := CreateOfferOp(selling, buying, sellAmount, price)
	check(err)

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&createOffer},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp1,
	)
	assert.NoError(t, err)

	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAAJWoAAAAFAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAADAAAAAAAAAAFBQkNEAAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAADuaygAAAAABAAAAZAAAAAAAAAAAAAAAAAAAAAHSh2R+AAAAQAmXf4BnH8bWhy+Tnxf+7zgsij7pV0b7XC4rqfYWi9ZIVUaidWPbrFhaWjiQbXYB1NKdx0XjidzkcAgMInLqDgs="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestManageSellOfferDeleteOffer(t *testing.T) {
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp1.Address(), int64(41137196761105))

	offerID := int64(2921622)
	deleteOffer, err := DeleteOfferOp(offerID)
	check(err)

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&deleteOffer},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp1,
	)
	assert.NoError(t, err)

	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAAJWoAAAASAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAADAAAAAAAAAAFGQUtFAAAAAEEHgGTElYZi82AkGiJdSja2OBaU2aEcwwp3AY3tFJ2xAAAAAAAAAAAAAAABAAAAAQAAAAAALJSWAAAAAAAAAAHSh2R+AAAAQBSjRfpyEAIMnRQOPf1BBOx8HFC6Lm6bxxdljaegnUts8SmWJGQbZN5a8PQGzOTwGdBKBk9X9d+BIrBVc3kyyQ4="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestManageSellOfferUpdateOffer(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp1.Address(), int64(41137196761097))

	selling := NativeAsset{}
	buying := CreditAsset{"ABCD", kp0.Address()}
	sellAmount := "50"
	price := "0.02"
	offerID := int64(2497628)
	updateOffer, err := UpdateOfferOp(selling, buying, sellAmount, price, offerID)
	check(err)

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&updateOffer},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp1,
	)
	assert.NoError(t, err)

	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAAJWoAAAAKAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAADAAAAAAAAAAFBQkNEAAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAAB3NZQAAAAABAAAAMgAAAAAAJhxcAAAAAAAAAAHSh2R+AAAAQAwqWg2C/oe/zH4D3Y7/yg5SlHqFvF6A3j6GQZ9NPh3ROqutovLyAE62+rvXxM7hqSNz1Rtx4frJaOhOabh6DAg="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestCreatePassiveSellOffer(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp1.Address(), int64(41137196761100))

	createPassiveOffer := CreatePassiveSellOffer{
		Selling: NativeAsset{},
		Buying:  CreditAsset{"ABCD", kp0.Address()},
		Amount:  "10",
		Price:   "1.0",
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&createPassiveOffer},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp1,
	)
	assert.NoError(t, err)

	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAAJWoAAAANAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAEAAAAAAAAAAFBQkNEAAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAAAX14QAAAAABAAAAAQAAAAAAAAAB0odkfgAAAEAgUD7M1UL7x2m2m26ySzcSHxIneOT7/r+s/HLsgWDj6CmpSi1GZrlvtBH+CNuegCwvW09TRZJhp7bLywkaFCoK"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestPathPayment(t *testing.T) {
	kp0 := newKeypair0()
	kp2 := newKeypair2()
	sourceAccount := NewSimpleAccount(kp2.Address(), int64(187316408680450))

	abcdAsset := CreditAsset{"ABCD", kp0.Address()}
	pathPayment := PathPayment{
		SendAsset:   NativeAsset{},
		SendMax:     "10",
		Destination: kp2.Address(),
		DestAsset:   NativeAsset{},
		DestAmount:  "1",
		Path:        []Asset{abcdAsset},
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&pathPayment},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp2,
	)
	assert.NoError(t, err)

	expected := "AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAAql0AAAADAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAACAAAAAAAAAAAF9eEAAAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAAAAAAAAAmJaAAAAAAQAAAAFBQkNEAAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAAAAAAAEuFVmYAAAAQF2kLUL/RoFIy1cmt+GXdWn2tDUjJYV3YwF4A82zIBhqYSO6ogOoLPNRt3w+IGCAgfR4Q9lpax+wCXWoQERHSw4="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestMemoText(t *testing.T) {
	kp2 := newKeypair2()
	sourceAccount := NewSimpleAccount(kp2.Address(), int64(3556099777101824))

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&BumpSequence{BumpTo: 1}},
			Memo:                 MemoText("Twas brillig"),
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp2,
	)
	assert.NoError(t, err)

	// https://www.stellar.org/laboratory/#txsigner?xdr=AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAMokEAAAABAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAABAAAADFR3YXMgYnJpbGxpZwAAAAEAAAAAAAAACwAAAAAAAAABAAAAAAAAAAA%3D&network=test
	expected := "AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAMokEAAAABAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAABAAAADFR3YXMgYnJpbGxpZwAAAAEAAAAAAAAACwAAAAAAAAABAAAAAAAAAAEuFVmYAAAAQILT8/7MGTmWkfjMi6Y23n2cVWs+IMY67xOskTivSZehp7wWaDXLIdCbdijmG64+Nz+fPBT9HYMqSRDcLiZYDQ0="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestMemoID(t *testing.T) {
	kp2 := newKeypair2()
	sourceAccount := NewSimpleAccount(kp2.Address(), int64(3428320205078528))

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&BumpSequence{BumpTo: 1}},
			Memo:                 MemoID(314159),
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp2,
	)
	assert.NoError(t, err)

	expected := "AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAMLgoAAAABAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAACAAAAAAAEyy8AAAABAAAAAAAAAAsAAAAAAAAAAQAAAAAAAAABLhVZmAAAAEA5P/V/Veh6pjXj7CnqtWDATh8II+ci1z3/zmNk374XLuVLzx7jRve59AKnPMwIPwDJ8cXwEKz8+fYOIkfEI9AJ"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestMemoHash(t *testing.T) {
	kp2 := newKeypair2()
	sourceAccount := NewSimpleAccount(kp2.Address(), int64(3428320205078528))

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&BumpSequence{BumpTo: 1}},
			Memo:                 MemoHash([32]byte{0x01}),
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp2,
	)
	assert.NoError(t, err)

	expected := "AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAMLgoAAAABAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAADAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAsAAAAAAAAAAQAAAAAAAAABLhVZmAAAAEAgauaUpqEGF1VeXYtkYg0I19QC3GJVrCPOqDHPIdXvGkQ9N+3Vt6yfKIN0sE/X5NuD6FhArQ3adwvZeaNDilwN"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestMemoReturn(t *testing.T) {
	kp2 := newKeypair2()
	sourceAccount := NewSimpleAccount(kp2.Address(), int64(3428320205078528))

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&BumpSequence{BumpTo: 1}},
			Memo:                 MemoReturn([32]byte{0x01}),
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp2,
	)
	assert.NoError(t, err)

	expected := "AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAMLgoAAAABAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAEAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAsAAAAAAAAAAQAAAAAAAAABLhVZmAAAAEAuLFTunY08pbWKompoepHdazLmr7uePUSOzA4P33+SVRKWiu+h2tngOsP8hga+wpLJXT9l/0uMQ3iziRVUrh0K"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestManageBuyOfferNewOffer(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp1.Address(), int64(41137196761092))

	buyOffer := ManageBuyOffer{
		Selling: NativeAsset{},
		Buying:  CreditAsset{"ABCD", kp0.Address()},
		Amount:  "100",
		Price:   "0.01",
		OfferID: 0,
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&buyOffer},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp1,
	)
	assert.NoError(t, err)

	// https://www.stellar.org/laboratory/#xdr-viewer?input=AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R%2BAAAAZAAAJWoAAAAFAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAMAAAAAAAAAAFBQkNEAAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAADuaygAAAAABAAAAZAAAAAAAAAAAAAAAAAAAAAHSh2R%2BAAAAQHwuorW7BvBwJAz%2BETSteeDZ9UKhox1y1BqJLvaIkWSr5rNbOpimjWQxrUNQoy%2B%2BwmtY8tiMSv3Jbz8Dd4QTaQU%3D&type=TransactionEnvelope&network=test
	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAAJWoAAAAFAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAMAAAAAAAAAAFBQkNEAAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAADuaygAAAAABAAAAZAAAAAAAAAAAAAAAAAAAAAHSh2R+AAAAQHwuorW7BvBwJAz+ETSteeDZ9UKhox1y1BqJLvaIkWSr5rNbOpimjWQxrUNQoy++wmtY8tiMSv3Jbz8Dd4QTaQU="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestManageBuyOfferDeleteOffer(t *testing.T) {
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp1.Address(), int64(41137196761105))

	buyOffer := ManageBuyOffer{
		Selling: NativeAsset{},
		Buying:  CreditAsset{"ABCD", kp1.Address()},
		Amount:  "0",
		Price:   "0.01",
		OfferID: int64(2921622),
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&buyOffer},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp1,
	)
	assert.NoError(t, err)

	// https://www.stellar.org/laboratory/#xdr-viewer?input=AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R%2BAAAAZAAAJWoAAAASAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAMAAAAAAAAAAFBQkNEAAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R%2BAAAAAAAAAAAAAAABAAAAZAAAAAAALJSWAAAAAAAAAAHSh2R%2BAAAAQItno%2BcpmUYFvxLcYVaDonTV3dmvzz%2B2SLzKRrYoXOqK8wCZjcP%2FkgzPMmXhTtF2tgQ9qb0rAIYpH9%2FrjtZPBgY%3D&type=TransactionEnvelope&network=test
	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAAJWoAAAASAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAMAAAAAAAAAAFBQkNEAAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAAAAAAAAAAAABAAAAZAAAAAAALJSWAAAAAAAAAAHSh2R+AAAAQItno+cpmUYFvxLcYVaDonTV3dmvzz+2SLzKRrYoXOqK8wCZjcP/kgzPMmXhTtF2tgQ9qb0rAIYpH9/rjtZPBgY="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestManageBuyOfferUpdateOffer(t *testing.T) {
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp1.Address(), int64(41137196761097))

	buyOffer := ManageBuyOffer{
		Selling: NativeAsset{},
		Buying:  CreditAsset{"ABCD", kp1.Address()},
		Amount:  "50",
		Price:   "0.02",
		OfferID: int64(2921622),
	}

	received, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&buyOffer},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp1,
	)
	assert.NoError(t, err)

	// https://www.stellar.org/laboratory/#xdr-viewer?input=AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R%2BAAAAZAAAJWoAAAAKAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAMAAAAAAAAAAFBQkNEAAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R%2BAAAAAB3NZQAAAAABAAAAMgAAAAAALJSWAAAAAAAAAAHSh2R%2BAAAAQK%2FsasTxgNqvkz3dGaDOyUgfa9UAAmUBmgiyaQU1dMlNNvTVH1D7PQKXkTooWmb6qK7Ee8vaTCFU6gGmShhA9wE%3D&type=TransactionEnvelope&network=test
	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAAJWoAAAAKAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAMAAAAAAAAAAFBQkNEAAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAAB3NZQAAAAABAAAAMgAAAAAALJSWAAAAAAAAAAHSh2R+AAAAQK/sasTxgNqvkz3dGaDOyUgfa9UAAmUBmgiyaQU1dMlNNvTVH1D7PQKXkTooWmb6qK7Ee8vaTCFU6gGmShhA9wE="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestBuildChallengeTx(t *testing.T) {
	kp0 := newKeypair0()

	{
		// 1 minute timebound
		tx, err := BuildChallengeTx(kp0.Seed(), kp0.Address(), "SDF", network.TestNetworkPassphrase, time.Minute)
		assert.NoError(t, err)
		txeBase64, err := tx.Base64()
		assert.NoError(t, err)
		var txXDR xdr.TransactionEnvelope
		err = xdr.SafeUnmarshalBase64(txeBase64, &txXDR)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), txXDR.SeqNum(), "sequence number should be 0")
		assert.Equal(t, uint32(100), txXDR.Fee(), "Fee should be 100")
		assert.Equal(t, 1, len(txXDR.Operations()), "number operations should be 1")
		timeDiff := txXDR.TimeBounds().MaxTime - txXDR.TimeBounds().MinTime
		assert.Equal(t, int64(60), int64(timeDiff), "time difference should be 300 seconds")
		op := txXDR.Operations()[0]
		assert.Equal(t, xdr.OperationTypeManageData, op.Body.Type, "operation type should be manage data")
		assert.Equal(t, xdr.String64("SDF auth"), op.Body.ManageDataOp.DataName, "DataName should be 'SDF auth'")
		assert.Equal(t, 64, len(*op.Body.ManageDataOp.DataValue), "DataValue should be 64 bytes")

	}

	{
		// 5 minutes timebound
		tx, err := BuildChallengeTx(kp0.Seed(), kp0.Address(), "SDF1", network.TestNetworkPassphrase, time.Duration(5*time.Minute))
		assert.NoError(t, err)
		txeBase64, err := tx.Base64()
		assert.NoError(t, err)
		var txXDR1 xdr.TransactionEnvelope
		err = xdr.SafeUnmarshalBase64(txeBase64, &txXDR1)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), txXDR1.SeqNum(), "sequence number should be 0")
		assert.Equal(t, uint32(100), txXDR1.Fee(), "Fee should be 100")
		assert.Equal(t, 1, len(txXDR1.Operations()), "number operations should be 1")

		timeDiff := txXDR1.TimeBounds().MaxTime - txXDR1.TimeBounds().MinTime
		assert.Equal(t, int64(300), int64(timeDiff), "time difference should be 300 seconds")
		op1 := txXDR1.Operations()[0]
		assert.Equal(t, xdr.OperationTypeManageData, op1.Body.Type, "operation type should be manage data")
		assert.Equal(t, xdr.String64("SDF1 auth"), op1.Body.ManageDataOp.DataName, "DataName should be 'SDF1 auth'")
		assert.Equal(t, 64, len(*op1.Body.ManageDataOp.DataValue), "DataValue should be 64 bytes")
	}

	//transaction with infinite timebound
	_, err := BuildChallengeTx(kp0.Seed(), kp0.Address(), "sdf", network.TestNetworkPassphrase, 0)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "provided timebound must be at least 1s (300s is recommended)")
	}
}

func TestHashHex(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(9605939170639897))

	createAccount := CreateAccount{
		Destination: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:      "10",
	}

	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&createAccount},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)

	tx, err = tx.Sign(network.TestNetworkPassphrase, kp0)
	assert.NoError(t, err)

	txeB64, err := tx.Base64()
	assert.NoError(t, err)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAaAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAITg3tq8G0kvnvoIhZPMYJsY+9KVV8xAA6NxhtKxIXZUAAAAAAX14QAAAAAAAAAAAeoucsUAAABAHsyMojA0Q5MiNsR5X5AiNpCn9mlXmqluRsNpTniCR91M4U5TFmrrqVNLkU58/l+Y8hUPwidDTRSzLZKbMUL/Bw=="
	assert.Equal(t, expected, txeB64, "Base 64 XDR should match")

	hashHex, err := tx.HashHex(network.TestNetworkPassphrase)
	assert.NoError(t, err)
	expected = "1b3905ba8c3c0ecc68ae812f2d77f27c697195e8daf568740fc0f5662f65f759"
	assert.Equal(t, expected, hashHex, "hex encoded hash should match")

	txEnv, err := tx.TxEnvelope()
	assert.NoError(t, err)
	assert.NotNil(t, txEnv, "transaction xdr envelope should not be nil")
	sourceAccountFromEnv := txEnv.SourceAccount().ToAccountId()
	assert.Equal(t, sourceAccount.AccountID, sourceAccountFromEnv.Address())
	assert.Equal(t, uint32(100), txEnv.Fee())
	assert.Equal(t, sourceAccount.Sequence, int64(txEnv.SeqNum()))
	assert.Equal(t, xdr.MemoTypeMemoNone, txEnv.Memo().Type)
	assert.Len(t, txEnv.Operations(), 1)
}

func TestTransactionFee(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(9605939170639897))

	createAccount := CreateAccount{
		Destination: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:      "10",
	}

	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&createAccount},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)

	assert.Equal(t, int64(100), tx.BaseFee(), "Transaction base fee should match")
	assert.Equal(t, int64(100), tx.MaxFee(), "Transaction fee should match")

	tx, err = NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&createAccount},
			BaseFee:              500,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)

	assert.Equal(t, int64(500), tx.BaseFee(), "Transaction base fee should match")
	assert.Equal(t, int64(500), tx.MaxFee(), "Transaction fee should match")

	tx, err = tx.Sign(network.TestNetworkPassphrase, kp0)
	assert.NoError(t, err)

	txeB64, err := tx.Base64()
	assert.NoError(t, err)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAB9AAiII0AAAAbAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAITg3tq8G0kvnvoIhZPMYJsY+9KVV8xAA6NxhtKxIXZUAAAAAAX14QAAAAAAAAAAAeoucsUAAABAnc69aKw6dg1LlHxkIetKZu8Ou8hgbj4mICV0tiOJeuiq8DvivSlAngnD+FlVIaotmg8i3dEzBg+LcLnG9UttBQ=="
	assert.Equal(t, expected, txeB64, "Base 64 XDR should match")
}

func TestPreAuthTransaction(t *testing.T) {
	// Address: GDK3YEHGI3ORGVO7ZEV2XF4SV5JU3BOKHMHPP4QFJ74ZRIIRROZ7ITOJ
	kp0 := newKeypair("SDY4PF6F6OWWERZT6OL2LVNREHUGHKALUI5W4U2JK4GAKPAC2RM43OAU")
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(4353383146192898)) // sequence number is in the future

	createAccount := CreateAccount{
		Destination: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:      "10",
	}

	// build transaction to be submitted in the future.
	txFuture, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&createAccount},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)

	// save the hash of the future transaction.
	txFutureHash, err := txFuture.Hash(network.TestNetworkPassphrase)
	assert.NoError(t, err)

	// sign transaction without keypairs, the hash of the future transaction on the account
	//  will be used for authorisation.
	txFuture, err = txFuture.Sign(network.TestNetworkPassphrase)
	assert.NoError(t, err)

	txeFutureB64, err := txFuture.Base64()
	assert.NoError(t, err)
	expected := "AAAAANW8EOZG3RNV38krq5eSr1NNhco7DvfyBU/5mKERi7P0AAAAZAAPd2EAAAADAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAITg3tq8G0kvnvoIhZPMYJsY+9KVV8xAA6NxhtKxIXZUAAAAAAX14QAAAAAAAAAAAA=="
	assert.Equal(t, expected, txeFutureB64, "Base 64 XDR should match")

	//encode the txFutureHash as a stellar HashTx signer key.
	preAuth, err := strkey.Encode(strkey.VersionByteHashTx, txFutureHash[:])
	assert.NoError(t, err)

	// set sequence number to the current number.
	sourceAccount.Sequence = int64(4353383146192897)

	// add hash of future transaction as signer to account
	setOptions := SetOptions{
		Signer: &Signer{Address: preAuth, Weight: Threshold(2)},
	}

	// build a transaction to add the hash of the future transaction as a signer on the account.
	txNow, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&setOptions},
			BaseFee:              500,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)

	assert.Equal(t, int64(500), txNow.MaxFee(), "Transaction fee should match")
	txNow, err = txNow.Sign(network.TestNetworkPassphrase, kp0)
	assert.NoError(t, err)

	txeNowB64, err := txNow.Base64()
	assert.NoError(t, err)
	expected = "AAAAANW8EOZG3RNV38krq5eSr1NNhco7DvfyBU/5mKERi7P0AAAB9AAPd2EAAAACAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAY9c66YpCPn8yMopKaNBd7gbiD2cr+aTLOaZE4whmeO1AAAAAgAAAAAAAAABEYuz9AAAAEC62tXQKDTcrB8VvOQIaI3ECV0uypBkpGNuyodnLLY27ii4QMdB4g4otYIvKY6nbWQqYYapNh6Q9dVsYfK6OHQM"
	assert.Equal(t, expected, txeNowB64, "Base 64 XDR should match")
	// Note: txeFutureB64 can be submitted to the network after txeNowB64 has been applied to the account
}

func TestHashXTransaction(t *testing.T) {
	// 256 bit preimage
	preimage := "this is a preimage for hashx transactions on the stellar network"

	preimageHash := sha256.Sum256([]byte(preimage))

	//encode preimageHash as a stellar HashX signer key
	hashx, err := strkey.Encode(strkey.VersionByteHashX, preimageHash[:])
	assert.NoError(t, err)

	// add hashx as signer to the account
	setOptions := SetOptions{
		Signer: &Signer{Address: hashx, Weight: Threshold(1)},
	}

	// Address: GDK3YEHGI3ORGVO7ZEV2XF4SV5JU3BOKHMHPP4QFJ74ZRIIRROZ7ITOJ
	kp0 := newKeypair("SDY4PF6F6OWWERZT6OL2LVNREHUGHKALUI5W4U2JK4GAKPAC2RM43OAU")
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(4353383146192899))

	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&setOptions},
			BaseFee:              500,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, int64(500), tx.MaxFee(), "Transaction fee should match")

	tx, err = tx.Sign(network.TestNetworkPassphrase, kp0)
	assert.NoError(t, err)

	txeB64, err := tx.Base64()
	assert.NoError(t, err)

	expected := "AAAAANW8EOZG3RNV38krq5eSr1NNhco7DvfyBU/5mKERi7P0AAAB9AAPd2EAAAAEAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAvslgb5oIfuISFP8FYvTqsciG1iSUerJB3Au6T2WLqMFAAAAAQAAAAAAAAABEYuz9AAAAECHBwfCbcOwFyoILLW7OZejvdbsVPEwB6z6ocAG4cRGu69vXKCrBFYD2mMdJRCeglgJgfgaFj2qshBgL8yQ14UH"
	assert.Equal(t, expected, txeB64, "Base 64 XDR should match")

	// build a transaction to test hashx signer
	payment := Payment{
		Destination: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:      "10",
		Asset:       NativeAsset{},
	}

	sourceAccount.Sequence = int64(4353383146192902)

	tx, err = NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&payment},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)

	// sign transaction with the preimage
	tx, err = tx.SignHashX([]byte(preimage))
	assert.NoError(t, err)

	txeB64, err = tx.Base64()
	assert.NoError(t, err)
	expected = "AAAAANW8EOZG3RNV38krq5eSr1NNhco7DvfyBU/5mKERi7P0AAAAZAAPd2EAAAAHAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAABAAAAAITg3tq8G0kvnvoIhZPMYJsY+9KVV8xAA6NxhtKxIXZUAAAAAAAAAAAF9eEAAAAAAAAAAAGWLqMFAAAAQHRoaXMgaXMgYSBwcmVpbWFnZSBmb3IgaGFzaHggdHJhbnNhY3Rpb25zIG9uIHRoZSBzdGVsbGFyIG5ldHdvcms="
	assert.Equal(t, expected, txeB64, "Base 64 XDR should match")

}

func TestFromXDR(t *testing.T) {
	txeB64 := "AAAAACYWIvM98KlTMs0IlQBZ06WkYpZ+gILsQN6ega0++I/sAAAAZAAXeEkAAAABAAAAAAAAAAEAAAAQMkExVjZKNTcwM0c0N1hIWQAAAAEAAAABAAAAACYWIvM98KlTMs0IlQBZ06WkYpZ+gILsQN6ega0++I/sAAAAAQAAAADMSEvcRKXsaUNna++Hy7gWm/CfqTjEA7xoGypfrFGUHAAAAAAAAAACCPHRAAAAAAAAAAABPviP7AAAAEBu6BCKf4WZHPum5+29Nxf6SsJNN8bgjp1+e1uNBaHjRg3rdFZYgUqEqbHxVEs7eze3IeRbjMZxS3zPf/xwJCEI"

	tx, err := TransactionFromXDR(txeB64)
	assert.NoError(t, err)
	newTx, ok := tx.Transaction()
	assert.True(t, ok)
	_, ok = tx.FeeBump()
	assert.False(t, ok)

	assert.Equal(t, "GATBMIXTHXYKSUZSZUEJKACZ2OS2IYUWP2AIF3CA32PIDLJ67CH6Y5UY", newTx.SourceAccount().AccountID, "source accounts should match")
	assert.Equal(t, int64(100), newTx.BaseFee(), "Base fee should match")
	sa := newTx.SourceAccount()
	assert.Equal(t, int64(6606179392290817), sa.Sequence, "Sequence number should match")
	assert.Equal(t, 1, len(newTx.Operations()), "Operations length should match")
	assert.IsType(t, newTx.Operations()[0], &Payment{}, "Operation types should match")
	paymentOp, ok1 := newTx.Operations()[0].(*Payment)
	assert.Equal(t, true, ok1)
	assert.Equal(t, "GATBMIXTHXYKSUZSZUEJKACZ2OS2IYUWP2AIF3CA32PIDLJ67CH6Y5UY", paymentOp.SourceAccount.GetAccountID(), "Operation source should match")
	assert.Equal(t, "GDGEQS64ISS6Y2KDM5V67B6LXALJX4E7VE4MIA54NANSUX5MKGKBZM5G", paymentOp.Destination, "Operation destination should match")
	assert.Equal(t, "874.0000000", paymentOp.Amount, "Operation amount should match")

	txeB64 = "AAAAAGigiN2q4qBXAERImNEncpaADylyBRtzdqpEsku6CN0xAAABkAAADXYAAAABAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAABAAAABm5ldyB0eAAAAAAAAgAAAAEAAAAA+Q2efEMLNGF4i+aYfutUXGMSlf8tNevKeS1Jl/oCVGkAAAAGAAAAAVVTRAAAAAAAaKCI3arioFcAREiY0SdyloAPKXIFG3N2qkSyS7oI3TF//////////wAAAAAAAAAKAAAABHRlc3QAAAABAAAABXZhbHVlAAAAAAAAAAAAAAA="

	tx2, err := TransactionFromXDR(txeB64)
	assert.NoError(t, err)
	newTx2, ok := tx2.Transaction()
	assert.True(t, ok)
	_, ok = tx2.FeeBump()
	assert.False(t, ok)

	assert.Equal(t, "GBUKBCG5VLRKAVYAIREJRUJHOKLIADZJOICRW43WVJCLES52BDOTCQZU", newTx2.SourceAccount().AccountID, "source accounts should match")
	assert.Equal(t, int64(200), newTx2.BaseFee(), "Base fee should match")
	assert.Equal(t, int64(14800457302017), newTx2.SourceAccount().Sequence, "Sequence number should match")

	memo, ok := newTx2.Memo().(MemoText)
	assert.Equal(t, true, ok)
	assert.Equal(t, MemoText("new tx"), memo, "memo should match")
	assert.Equal(t, 2, len(newTx2.Operations()), "Operations length should match")
	assert.IsType(t, newTx2.Operations()[0], &ChangeTrust{}, "Operation types should match")
	assert.IsType(t, newTx2.Operations()[1], &ManageData{}, "Operation types should match")
	op1, ok1 := newTx2.Operations()[0].(*ChangeTrust)
	assert.Equal(t, true, ok1)
	assert.Equal(t, "GD4Q3HT4IMFTIYLYRPTJQ7XLKROGGEUV74WTL26KPEWUTF72AJKGSJS7", op1.SourceAccount.GetAccountID(), "Operation source should match")
	assetType, err := op1.Line.GetType()
	assert.NoError(t, err)

	assert.Equal(t, AssetTypeCreditAlphanum4, assetType, "Asset type should match")
	assert.Equal(t, "USD", op1.Line.GetCode(), "Asset code should match")
	assert.Equal(t, "GBUKBCG5VLRKAVYAIREJRUJHOKLIADZJOICRW43WVJCLES52BDOTCQZU", op1.Line.GetIssuer(), "Asset issuer should match")
	assert.Equal(t, "922337203685.4775807", op1.Limit, "trustline limit should match")

	op2, ok2 := newTx2.Operations()[1].(*ManageData)
	assert.Equal(t, true, ok2)
	assert.Equal(t, nil, op2.SourceAccount, "Operation source should match")
	assert.Equal(t, "test", op2.Name, "Name should match")
	assert.Equal(t, "value", string(op2.Value), "Value should match")
}

func TestBuild(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(9605939170639897))
	createAccount := CreateAccount{
		Destination: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:      "10",
	}

	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&createAccount},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)

	expectedUnsigned := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAaAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAITg3tq8G0kvnvoIhZPMYJsY+9KVV8xAA6NxhtKxIXZUAAAAAAX14QAAAAAAAAAAAA=="

	expectedSigned := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAaAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAITg3tq8G0kvnvoIhZPMYJsY+9KVV8xAA6NxhtKxIXZUAAAAAAX14QAAAAAAAAAAAeoucsUAAABAHsyMojA0Q5MiNsR5X5AiNpCn9mlXmqluRsNpTniCR91M4U5TFmrrqVNLkU58/l+Y8hUPwidDTRSzLZKbMUL/Bw=="

	txeB64, err := tx.Base64()
	assert.NoError(t, err)
	assert.Equal(t, expectedUnsigned, txeB64, "tx envelope should match")
	tx, err = tx.Sign(network.TestNetworkPassphrase, kp0)
	assert.NoError(t, err)
	txeB64, err = tx.Base64()
	assert.NoError(t, err)
	assert.Equal(t, expectedSigned, txeB64, "tx envelope should match")
}

func TestFromXDRBuildSignEncode(t *testing.T) {
	expectedUnsigned := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAaAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAITg3tq8G0kvnvoIhZPMYJsY+9KVV8xAA6NxhtKxIXZUAAAAAAX14QAAAAAAAAAAAA=="

	expectedSigned := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAaAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAITg3tq8G0kvnvoIhZPMYJsY+9KVV8xAA6NxhtKxIXZUAAAAAAX14QAAAAAAAAAAAeoucsUAAABAHsyMojA0Q5MiNsR5X5AiNpCn9mlXmqluRsNpTniCR91M4U5TFmrrqVNLkU58/l+Y8hUPwidDTRSzLZKbMUL/Bw=="

	kp0 := newKeypair0()

	// test signing transaction  without modification
	tx, err := TransactionFromXDR(expectedUnsigned)
	assert.NoError(t, err)
	newTx, ok := tx.Transaction()
	assert.True(t, ok)
	_, ok = tx.FeeBump()
	assert.False(t, ok)

	//passphrase is needed for signing
	newTx, err = newTx.Sign(network.TestNetworkPassphrase, kp0)
	assert.NoError(t, err)
	txeB64, err := newTx.Base64()
	assert.NoError(t, err)
	assert.Equal(t, expectedSigned, txeB64, "tx envelope should match")

	// test signing transaction  with modification
	expectedSigned2 := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAbAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAABAAAABW5ld3R4AAAAAAAAAQAAAAAAAAAAAAAAAITg3tq8G0kvnvoIhZPMYJsY+9KVV8xAA6NxhtKxIXZUAAAAAAX14QAAAAAAAAAAAeoucsUAAABADPbbXNzpC408WyYGQszN3VA9e41sNpsyZ2HcS62RXvUDsN0A+IXMPRMaCb+Wgn1OM6Ikam9ol0MJYNeK0BPxCg=="
	tx, err = TransactionFromXDR(expectedUnsigned)
	assert.NoError(t, err)
	newTx, ok = tx.Transaction()
	assert.True(t, ok)
	_, ok = tx.FeeBump()
	assert.False(t, ok)

	txeB64, err = newSignedTransaction(
		TransactionParams{
			SourceAccount: &SimpleAccount{
				AccountID: newTx.SourceAccount().AccountID,
				Sequence:  newTx.SourceAccount().Sequence + 1,
			},
			IncrementSequenceNum: false,
			Operations:           newTx.Operations(),
			BaseFee:              newTx.BaseFee(),
			Memo:                 MemoText("newtx"),
			Timebounds:           newTx.Timebounds(),
		},
		network.TestNetworkPassphrase,
		kp0,
	)
	assert.NoError(t, err)
	assert.Equal(t, expectedSigned2, txeB64, "tx envelope should match")
}

func TestSignWithSecretKey(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	txSource := NewSimpleAccount(kp0.Address(), int64(9605939170639897))
	tx1Source := NewSimpleAccount(kp0.Address(), int64(9605939170639897))
	opSource := NewSimpleAccount(kp1.Address(), 0)
	createAccount := CreateAccount{
		Destination:   "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:        "10",
		SourceAccount: &opSource,
	}

	expected, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&createAccount},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0, kp1,
	)
	assert.NoError(t, err)

	tx1, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &tx1Source,
			IncrementSequenceNum: true,
			Operations:           []Operation{&createAccount},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)

	tx1, err = tx1.SignWithKeyString(
		network.TestNetworkPassphrase,
		"SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R", ""+
			"SBMSVD4KKELKGZXHBUQTIROWUAPQASDX7KEJITARP4VMZ6KLUHOGPTYW",
	)
	assert.NoError(t, err)

	actual, err := tx1.Base64()
	assert.NoError(t, err)
	assert.Equal(t, expected, actual, "base64 xdr should match")
}

func TestAddSignatureBase64(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	txSource := NewSimpleAccount(kp0.Address(), int64(9605939170639897))
	tx1Source := NewSimpleAccount(kp0.Address(), int64(9605939170639897))
	opSource := NewSimpleAccount(kp1.Address(), 0)
	createAccount := CreateAccount{
		Destination:   "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:        "10",
		SourceAccount: &opSource,
	}

	expected, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&createAccount},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
		network.TestNetworkPassphrase,
		kp0, kp1,
	)
	assert.NoError(t, err)

	tx1, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &tx1Source,
			IncrementSequenceNum: true,
			Operations:           []Operation{&createAccount},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)

	tx1, err = tx1.AddSignatureBase64(
		network.TestNetworkPassphrase,
		"GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3",
		"TVogR6tbrWLnOc1BsP/j+Qrxpja2NWNgeRIwujECYscRdMG7AMtnb3dkCT7sqlbSM0TTzlRh7G+BcVocYBtqBw==",
	)
	assert.NoError(t, err)

	tx1, err = tx1.AddSignatureBase64(
		network.TestNetworkPassphrase,
		"GAS4V4O2B7DW5T7IQRPEEVCRXMDZESKISR7DVIGKZQYYV3OSQ5SH5LVP",
		"Iy77JteoW/FbeiuViZpgTyvrHP4BnBOeyVOjrdb5O/MpEMwcSlYXAkCBqPt4tBDil4jIcDDLhm7TsN6aUBkIBg==",
	)

	actual, err := tx1.Base64()
	assert.NoError(t, err)
	assert.Equal(t, expected, actual, "base64 xdr should match")
}

func TestReadChallengeTx_validSignedByServerAndClient(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}

	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
	)
	assert.NoError(t, err)

	tx, err = tx.Sign(network.TestNetworkPassphrase, serverKP, clientKP)
	assert.NoError(t, err)
	tx64, err := tx.Base64()
	require.NoError(t, err)
	readTx, readClientAccountID, err := ReadChallengeTx(tx64, serverKP.Address(), network.TestNetworkPassphrase)
	assert.Equal(t, tx, readTx)
	assert.Equal(t, clientKP.Address(), readClientAccountID)
	assert.NoError(t, err)
}

func TestReadChallengeTx_validSignedByServer(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
	)
	assert.NoError(t, err)

	tx, err = tx.Sign(network.TestNetworkPassphrase, serverKP)
	assert.NoError(t, err)
	tx64, err := tx.Base64()
	require.NoError(t, err)
	readTx, readClientAccountID, err := ReadChallengeTx(tx64, serverKP.Address(), network.TestNetworkPassphrase)
	assert.Equal(t, tx, readTx)
	assert.Equal(t, clientKP.Address(), readClientAccountID)
	assert.NoError(t, err)
}

func TestReadChallengeTx_invalidNotSignedByServer(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
	)
	assert.NoError(t, err)

	tx64, err := tx.Base64()
	require.NoError(t, err)
	readTx, readClientAccountID, err := ReadChallengeTx(tx64, serverKP.Address(), network.TestNetworkPassphrase)
	assert.Equal(t, tx, readTx)
	assert.Equal(t, clientKP.Address(), readClientAccountID)
	assert.EqualError(t, err, "transaction not signed by "+serverKP.Address())
}

func TestReadChallengeTx_invalidCorrupted(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
	)
	assert.NoError(t, err)

	tx, err = tx.Sign(network.TestNetworkPassphrase, serverKP)
	assert.NoError(t, err)
	tx64, err := tx.Base64()
	require.NoError(t, err)
	tx64 = strings.ReplaceAll(tx64, "A", "B")
	readTx, readClientAccountID, err := ReadChallengeTx(tx64, serverKP.Address(), network.TestNetworkPassphrase)
	assert.Nil(t, readTx)
	assert.Equal(t, "", readClientAccountID)
	assert.EqualError(
		t,
		err,
		"could not parse challenge: unable to unmarshal transaction envelope: "+
			"xdr:decode: switch '68174084' is not valid enum value for union",
	)
}

func TestReadChallengeTx_invalidServerAccountIDMismatch(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(newKeypair2().Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
	)
	assert.NoError(t, err)

	tx, err = tx.Sign(network.TestNetworkPassphrase, serverKP)
	assert.NoError(t, err)
	tx64, err := tx.Base64()
	require.NoError(t, err)
	readTx, readClientAccountID, err := ReadChallengeTx(tx64, serverKP.Address(), network.TestNetworkPassphrase)
	assert.Equal(t, tx, readTx)
	assert.Equal(t, "", readClientAccountID)
	assert.EqualError(t, err, "transaction source account is not equal to server's account")
}

func TestReadChallengeTx_invalidSeqNoNotZero(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(serverKP.Address(), 1234)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
	)
	assert.NoError(t, err)

	tx, err = tx.Sign(network.TestNetworkPassphrase, serverKP)
	assert.NoError(t, err)
	tx64, err := tx.Base64()
	require.NoError(t, err)
	readTx, readClientAccountID, err := ReadChallengeTx(tx64, serverKP.Address(), network.TestNetworkPassphrase)
	assert.Equal(t, tx, readTx)
	assert.Equal(t, "", readClientAccountID)
	assert.EqualError(t, err, "transaction sequence number must be 0")
}

func TestReadChallengeTx_invalidTimeboundsInfinite(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)

	tx, err = tx.Sign(network.TestNetworkPassphrase, serverKP)
	assert.NoError(t, err)
	tx64, err := tx.Base64()
	require.NoError(t, err)
	readTx, readClientAccountID, err := ReadChallengeTx(tx64, serverKP.Address(), network.TestNetworkPassphrase)
	assert.Equal(t, tx, readTx)
	assert.Equal(t, "", readClientAccountID)
	assert.EqualError(t, err, "transaction requires non-infinite timebounds")
}

func TestReadChallengeTx_invalidTimeboundsOutsideRange(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimebounds(0, 100),
		},
	)
	assert.NoError(t, err)

	tx, err = tx.Sign(network.TestNetworkPassphrase, serverKP)
	assert.NoError(t, err)
	tx64, err := tx.Base64()
	require.NoError(t, err)
	readTx, readClientAccountID, err := ReadChallengeTx(tx64, serverKP.Address(), network.TestNetworkPassphrase)
	assert.Equal(t, tx, readTx)
	assert.Equal(t, "", readClientAccountID)
	assert.Error(t, err)
	assert.Regexp(t, "transaction is not within range of the specified timebounds", err.Error())
}

func TestReadChallengeTx_invalidTooManyOperations(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op, &op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(300),
		},
	)
	assert.NoError(t, err)

	tx, err = tx.Sign(network.TestNetworkPassphrase, serverKP)
	assert.NoError(t, err)
	tx64, err := tx.Base64()
	require.NoError(t, err)
	_, _, err = ReadChallengeTx(tx64, serverKP.Address(), network.TestNetworkPassphrase)
	assert.EqualError(t, err, "transaction requires a single manage_data operation")
}

func TestReadChallengeTx_invalidOperationWrongType(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := BumpSequence{
		SourceAccount: &opSource,
		BumpTo:        0,
	}
	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(300),
		},
	)
	assert.NoError(t, err)

	tx, err = tx.Sign(network.TestNetworkPassphrase, serverKP)
	assert.NoError(t, err)
	tx64, err := tx.Base64()
	require.NoError(t, err)
	readTx, readClientAccountID, err := ReadChallengeTx(tx64, serverKP.Address(), network.TestNetworkPassphrase)
	assert.Equal(t, tx, readTx)
	assert.Equal(t, "", readClientAccountID)
	assert.EqualError(t, err, "operation type should be manage_data")
}

func TestReadChallengeTx_invalidOperationNoSourceAccount(t *testing.T) {
	serverKP := newKeypair0()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	op := ManageData{
		Name:  "testserver auth",
		Value: []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(300),
		},
	)
	assert.NoError(t, err)

	tx, err = tx.Sign(network.TestNetworkPassphrase, serverKP)
	assert.NoError(t, err)
	tx64, err := tx.Base64()
	require.NoError(t, err)
	_, _, err = ReadChallengeTx(tx64, serverKP.Address(), network.TestNetworkPassphrase)
	assert.EqualError(t, err, "operation should have a source account")
}

func TestReadChallengeTx_invalidDataValueWrongEncodedLength(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 45))),
	}
	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(300),
		},
	)
	assert.NoError(t, err)

	tx, err = tx.Sign(network.TestNetworkPassphrase, serverKP)
	assert.NoError(t, err)
	tx64, err := tx.Base64()
	require.NoError(t, err)
	readTx, readClientAccountID, err := ReadChallengeTx(tx64, serverKP.Address(), network.TestNetworkPassphrase)
	assert.Equal(t, tx, readTx)
	assert.Equal(t, clientKP.Address(), readClientAccountID)
	assert.EqualError(t, err, "random nonce encoded as base64 should be 64 bytes long")
}

func TestReadChallengeTx_invalidDataValueCorruptBase64(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA?AAAAAAAAAAAAAAAAAAAAAAAAAA"),
	}
	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(300),
		},
	)
	assert.NoError(t, err)

	tx, err = tx.Sign(network.TestNetworkPassphrase, serverKP)
	assert.NoError(t, err)
	tx64, err := tx.Base64()
	require.NoError(t, err)
	readTx, readClientAccountID, err := ReadChallengeTx(tx64, serverKP.Address(), network.TestNetworkPassphrase)
	assert.Equal(t, tx, readTx)
	assert.Equal(t, clientKP.Address(), readClientAccountID)
	assert.EqualError(t, err, "failed to decode random nonce provided in manage_data operation: illegal base64 data at input byte 37")
}

func TestReadChallengeTx_invalidDataValueWrongByteLength(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 47))),
	}
	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
	)
	assert.NoError(t, err)

	tx, err = tx.Sign(network.TestNetworkPassphrase, serverKP)
	assert.NoError(t, err)
	tx64, err := tx.Base64()
	assert.NoError(t, err)

	readTx, readClientAccountID, err := ReadChallengeTx(tx64, serverKP.Address(), network.TestNetworkPassphrase)
	assert.Equal(t, tx, readTx)
	assert.Equal(t, clientKP.Address(), readClientAccountID)
	assert.EqualError(t, err, "random nonce before encoding as base64 should be 48 bytes long")
}

func TestReadChallengeTx_acceptsV0AndV1Transactions(t *testing.T) {
	kp0 := newKeypair0()
	tx, err := BuildChallengeTx(
		kp0.Seed(),
		kp0.Address(),
		"SDF",
		network.TestNetworkPassphrase,
		time.Hour,
	)
	assert.NoError(t, err)

	originalHash, err := tx.HashHex(network.TestNetworkPassphrase)
	assert.NoError(t, err)

	tx.envelope.V1 = &xdr.TransactionV1Envelope{
		Tx: xdr.Transaction{
			SourceAccount: tx.envelope.SourceAccount(),
			Fee:           xdr.Uint32(tx.envelope.Fee()),
			SeqNum:        xdr.SequenceNumber(tx.envelope.SeqNum()),
			TimeBounds:    tx.envelope.V0.Tx.TimeBounds,
			Memo:          tx.envelope.Memo(),
			Operations:    tx.envelope.Operations(),
		},
	}
	tx.envelope.Type = xdr.EnvelopeTypeEnvelopeTypeTx
	v1Challenge, err := marshallBase64(tx.envelope, tx.Signatures())
	assert.NoError(t, err)

	tx.envelope.V0 = &xdr.TransactionV0Envelope{
		Tx: xdr.TransactionV0{
			SourceAccountEd25519: *tx.envelope.SourceAccount().Ed25519,
			Fee:                  xdr.Uint32(tx.envelope.Fee()),
			SeqNum:               xdr.SequenceNumber(tx.envelope.SeqNum()),
			TimeBounds:           tx.envelope.V0.Tx.TimeBounds,
			Memo:                 tx.envelope.Memo(),
			Operations:           tx.envelope.Operations(),
		},
	}
	tx.envelope.Type = xdr.EnvelopeTypeEnvelopeTypeTxV0
	v0Challenge, err := marshallBase64(tx.envelope, tx.Signatures())
	assert.NoError(t, err)

	for _, challenge := range []string{v1Challenge, v0Challenge} {
		parsedTx, clientAccountID, err := ReadChallengeTx(
			challenge,
			kp0.Address(),
			network.TestNetworkPassphrase,
		)
		assert.NoError(t, err)

		assert.Equal(t, kp0.Address(), clientAccountID)

		hash, err := parsedTx.HashHex(network.TestNetworkPassphrase)
		assert.NoError(t, err)
		assert.Equal(t, originalHash, hash)
	}
}

func TestReadChallengeTx_forbidsFeeBumpTransactions(t *testing.T) {
	kp0 := newKeypair0()
	tx, err := BuildChallengeTx(
		kp0.Seed(),
		kp0.Address(),
		"SDF",
		network.TestNetworkPassphrase,
		time.Hour,
	)
	assert.NoError(t, err)

	convertToV1Tx(tx)
	kp1 := newKeypair1()
	feeBumpTx, err := NewFeeBumpTransaction(
		FeeBumpTransactionParams{
			Inner:      tx,
			FeeAccount: kp1.Address(),
			BaseFee:    3 * MinBaseFee,
		},
	)
	assert.NoError(t, err)

	feeBumpTx, err = feeBumpTx.Sign(network.TestNetworkPassphrase, kp1)
	assert.NoError(t, err)

	challenge, err := feeBumpTx.Base64()
	assert.NoError(t, err)
	_, _, err = ReadChallengeTx(
		challenge,
		kp0.Address(),
		network.TestNetworkPassphrase,
	)
	assert.EqualError(t, err, "challenge cannot be a fee bump transaction")
}

func TestReadChallengeTx_forbidsMuxedAccounts(t *testing.T) {
	kp0 := newKeypair0()
	tx, err := BuildChallengeTx(
		kp0.Seed(),
		kp0.Address(),
		"SDF",
		network.TestNetworkPassphrase,
		time.Hour,
	)

	env, err := tx.TxEnvelope()
	assert.NoError(t, err)
	aid := xdr.MustAddress(kp0.Address())
	muxedAccount := xdr.MuxedAccount{
		Type: xdr.CryptoKeyTypeKeyTypeMuxedEd25519,
		Med25519: &xdr.MuxedAccountMed25519{
			Id:      0xcafebabe,
			Ed25519: *aid.Ed25519,
		},
	}
	*env.V0.Tx.Operations[0].SourceAccount = muxedAccount

	challenge, err := marshallBase64(env, env.Signatures())
	assert.NoError(t, err)

	_, _, err = ReadChallengeTx(
		challenge,
		kp0.Address(),
		network.TestNetworkPassphrase,
	)
	errorMessage := "only valid Ed25519 accounts are allowed in challenge transactions"
	assert.Contains(t, err.Error(), errorMessage)
}

func TestVerifyChallengeTxThreshold_invalidServer(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}

	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		clientKP,
	)
	assert.NoError(t, err)

	threshold := Threshold(1)
	signerSummary := SignerSummary{
		clientKP.Address(): 1,
	}
	signersFound, err := VerifyChallengeTxThreshold(tx64, serverKP.Address(), network.TestNetworkPassphrase, threshold, signerSummary)
	assert.Empty(t, signersFound)
	assert.EqualError(t, err, "transaction not signed by "+serverKP.Address())
}

func TestVerifyChallengeTxThreshold_validServerAndClientKeyMeetingThreshold(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP,
	)
	assert.NoError(t, err)

	threshold := Threshold(1)
	signerSummary := SignerSummary{
		clientKP.Address(): 1,
	}
	wantSigners := []string{
		clientKP.Address(),
	}

	signersFound, err := VerifyChallengeTxThreshold(tx64, serverKP.Address(), network.TestNetworkPassphrase, threshold, signerSummary)
	assert.ElementsMatch(t, wantSigners, signersFound)
	assert.NoError(t, err)
}

func TestVerifyChallengeTxThreshold_validServerAndMultipleClientKeyMeetingThreshold(t *testing.T) {
	serverKP := newKeypair0()
	clientKP1 := newKeypair1()
	clientKP2 := newKeypair2()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP1.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP1, clientKP2,
	)
	assert.NoError(t, err)

	threshold := Threshold(3)
	signerSummary := map[string]int32{
		clientKP1.Address(): 1,
		clientKP2.Address(): 2,
	}
	wantSigners := []string{
		clientKP1.Address(),
		clientKP2.Address(),
	}

	signersFound, err := VerifyChallengeTxThreshold(tx64, serverKP.Address(), network.TestNetworkPassphrase, threshold, signerSummary)
	assert.ElementsMatch(t, wantSigners, signersFound)
	assert.NoError(t, err)
}

func TestVerifyChallengeTxThreshold_validServerAndMultipleClientKeyMeetingThresholdSomeUnused(t *testing.T) {
	serverKP := newKeypair0()
	clientKP1 := newKeypair1()
	clientKP2 := newKeypair2()
	clientKP3 := keypair.MustRandom()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP1.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	threshold := Threshold(3)
	signerSummary := SignerSummary{
		clientKP1.Address(): 1,
		clientKP2.Address(): 2,
		clientKP3.Address(): 2,
	}
	wantSigners := []string{
		clientKP1.Address(),
		clientKP2.Address(),
	}

	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP1, clientKP2,
	)
	assert.NoError(t, err)

	signersFound, err := VerifyChallengeTxThreshold(tx64, serverKP.Address(), network.TestNetworkPassphrase, threshold, signerSummary)
	assert.ElementsMatch(t, wantSigners, signersFound)
	assert.NoError(t, err)
}

func TestVerifyChallengeTxThreshold_validServerAndMultipleClientKeyMeetingThresholdSomeUnusedIgnorePreauthTxHashAndXHash(t *testing.T) {
	serverKP := newKeypair0()
	clientKP1 := newKeypair1()
	clientKP2 := newKeypair2()
	clientKP3 := keypair.MustRandom()
	preauthTxHash := "TAQCSRX2RIDJNHFIFHWD63X7D7D6TRT5Y2S6E3TEMXTG5W3OECHZ2OG4"
	xHash := "XDRPF6NZRR7EEVO7ESIWUDXHAOMM2QSKIQQBJK6I2FB7YKDZES5UCLWD"
	unknownSignerType := "?ARPF6NZRR7EEVO7ESIWUDXHAOMM2QSKIQQBJK6I2FB7YKDZES5UCLWD"
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP1.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	threshold := Threshold(3)
	signerSummary := SignerSummary{
		clientKP1.Address(): 1,
		clientKP2.Address(): 2,
		clientKP3.Address(): 2,
		preauthTxHash:       10,
		xHash:               10,
		unknownSignerType:   10,
	}
	wantSigners := []string{
		clientKP1.Address(),
		clientKP2.Address(),
	}

	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP1, clientKP2,
	)
	assert.NoError(t, err)

	signersFound, err := VerifyChallengeTxThreshold(tx64, serverKP.Address(), network.TestNetworkPassphrase, threshold, signerSummary)
	assert.ElementsMatch(t, wantSigners, signersFound)
	assert.NoError(t, err)
}

func TestVerifyChallengeTxThreshold_invalidServerAndMultipleClientKeyNotMeetingThreshold(t *testing.T) {
	serverKP := newKeypair0()
	clientKP1 := newKeypair1()
	clientKP2 := newKeypair2()
	clientKP3 := keypair.MustRandom()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP1.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	threshold := Threshold(10)
	signerSummary := SignerSummary{
		clientKP1.Address(): 1,
		clientKP2.Address(): 2,
		clientKP3.Address(): 2,
	}

	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP1, clientKP2,
	)
	assert.NoError(t, err)

	_, err = VerifyChallengeTxThreshold(tx64, serverKP.Address(), network.TestNetworkPassphrase, threshold, signerSummary)
	assert.EqualError(t, err, "signers with weight 3 do not meet threshold 10")
}

func TestVerifyChallengeTxThreshold_invalidClientKeyUnrecognized(t *testing.T) {
	serverKP := newKeypair0()
	clientKP1 := newKeypair1()
	clientKP2 := newKeypair2()
	clientKP3 := keypair.MustRandom()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP1.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	threshold := Threshold(10)
	signerSummary := map[string]int32{
		clientKP1.Address(): 1,
		clientKP2.Address(): 2,
	}

	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP1, clientKP2, clientKP3,
	)
	assert.NoError(t, err)

	_, err = VerifyChallengeTxThreshold(tx64, serverKP.Address(), network.TestNetworkPassphrase, threshold, signerSummary)
	assert.EqualError(t, err, "transaction has unrecognized signatures")
}

func TestVerifyChallengeTxThreshold_invalidNoSigners(t *testing.T) {
	serverKP := newKeypair0()
	clientKP1 := newKeypair1()
	clientKP2 := newKeypair2()
	clientKP3 := keypair.MustRandom()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP1.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	threshold := Threshold(10)
	signerSummary := SignerSummary{}

	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP1, clientKP2, clientKP3,
	)
	assert.NoError(t, err)

	_, err = VerifyChallengeTxThreshold(tx64, serverKP.Address(), network.TestNetworkPassphrase, threshold, signerSummary)
	assert.EqualError(t, err, "no verifiable signers provided, at least one G... address must be provided")
}

func TestVerifyChallengeTxThreshold_weightsAddToMoreThan8Bits(t *testing.T) {
	serverKP := newKeypair0()
	clientKP1 := newKeypair1()
	clientKP2 := newKeypair2()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP1.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP1, clientKP2,
	)
	assert.NoError(t, err)

	threshold := Threshold(1)
	signerSummary := SignerSummary{
		clientKP1.Address(): 255,
		clientKP2.Address(): 1,
	}
	wantSigners := []string{
		clientKP1.Address(),
		clientKP2.Address(),
	}

	signersFound, err := VerifyChallengeTxThreshold(tx64, serverKP.Address(), network.TestNetworkPassphrase, threshold, signerSummary)
	assert.ElementsMatch(t, wantSigners, signersFound)
	assert.NoError(t, err)
}

func TestVerifyChallengeTxSigners_invalidServer(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		clientKP,
	)
	assert.NoError(t, err)

	signersFound, err := VerifyChallengeTxSigners(tx64, serverKP.Address(), network.TestNetworkPassphrase, clientKP.Address())
	assert.Empty(t, signersFound)
	assert.EqualError(t, err, "transaction not signed by "+serverKP.Address())
}

func TestVerifyChallengeTxSigners_validServerAndClientMasterKey(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP,
	)
	assert.NoError(t, err)

	signersFound, err := VerifyChallengeTxSigners(tx64, serverKP.Address(), network.TestNetworkPassphrase, clientKP.Address())
	assert.Equal(t, []string{clientKP.Address()}, signersFound)
	assert.NoError(t, err)
}

func TestVerifyChallengeTxSigners_invalidServerAndNoClient(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP,
	)
	assert.NoError(t, err)

	signersFound, err := VerifyChallengeTxSigners(tx64, serverKP.Address(), network.TestNetworkPassphrase, clientKP.Address())
	assert.Empty(t, signersFound)
	assert.EqualError(t, err, "transaction not signed by "+clientKP.Address())
}

func TestVerifyChallengeTxSigners_invalidServerAndUnrecognizedClient(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	unrecognizedKP := newKeypair2()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, unrecognizedKP,
	)
	assert.NoError(t, err)

	signersFound, err := VerifyChallengeTxSigners(tx64, serverKP.Address(), network.TestNetworkPassphrase, clientKP.Address())
	assert.Empty(t, signersFound)
	assert.EqualError(t, err, "transaction not signed by "+clientKP.Address())
}

func TestVerifyChallengeTxSigners_validServerAndMultipleClientSigners(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	clientKP2 := newKeypair2()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP, clientKP2,
	)
	assert.NoError(t, err)

	signersFound, err := VerifyChallengeTxSigners(tx64, serverKP.Address(), network.TestNetworkPassphrase, clientKP.Address(), clientKP2.Address())
	assert.Equal(t, []string{clientKP.Address(), clientKP2.Address()}, signersFound)
	assert.NoError(t, err)
}

func TestVerifyChallengeTxSigners_validServerAndMultipleClientSignersReverseOrder(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	clientKP2 := newKeypair2()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP2, clientKP,
	)
	assert.NoError(t, err)

	signersFound, err := VerifyChallengeTxSigners(tx64, serverKP.Address(), network.TestNetworkPassphrase, clientKP.Address(), clientKP2.Address())
	assert.Equal(t, []string{clientKP.Address(), clientKP2.Address()}, signersFound)
	assert.NoError(t, err)
}

func TestVerifyChallengeTxSigners_validServerAndClientSignersNotMasterKey(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	clientKP2 := newKeypair2()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP2,
	)
	assert.NoError(t, err)

	signersFound, err := VerifyChallengeTxSigners(tx64, serverKP.Address(), network.TestNetworkPassphrase, clientKP2.Address())
	assert.Equal(t, []string{clientKP2.Address()}, signersFound)
	assert.NoError(t, err)
}

func TestVerifyChallengeTxSigners_validServerAndClientSignersIgnoresServerSigner(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	clientKP2 := newKeypair2()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP2,
	)
	assert.NoError(t, err)

	signersFound, err := VerifyChallengeTxSigners(tx64, serverKP.Address(), network.TestNetworkPassphrase, serverKP.Address(), clientKP2.Address())
	assert.Equal(t, []string{clientKP2.Address()}, signersFound)
	assert.NoError(t, err)
}

func TestVerifyChallengeTxSigners_invalidServerNoClientSignersIgnoresServerSigner(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	clientKP2 := newKeypair2()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP,
	)
	assert.NoError(t, err)

	signersFound, err := VerifyChallengeTxSigners(tx64, serverKP.Address(), network.TestNetworkPassphrase, serverKP.Address(), clientKP2.Address())
	assert.Empty(t, signersFound)
	assert.EqualError(t, err, "transaction not signed by "+clientKP2.Address())
}

func TestVerifyChallengeTxSigners_validServerAndClientSignersIgnoresDuplicateSigner(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	clientKP2 := newKeypair2()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP2,
	)
	assert.NoError(t, err)

	signersFound, err := VerifyChallengeTxSigners(tx64, serverKP.Address(), network.TestNetworkPassphrase, clientKP2.Address(), clientKP2.Address())
	assert.Equal(t, []string{clientKP2.Address()}, signersFound)
	assert.NoError(t, err)
}

func TestVerifyChallengeTxSigners_validIgnorePreauthTxHashAndXHash(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	clientKP2 := newKeypair2()
	preauthTxHash := "TAQCSRX2RIDJNHFIFHWD63X7D7D6TRT5Y2S6E3TEMXTG5W3OECHZ2OG4"
	xHash := "XDRPF6NZRR7EEVO7ESIWUDXHAOMM2QSKIQQBJK6I2FB7YKDZES5UCLWD"
	unknownSignerType := "?ARPF6NZRR7EEVO7ESIWUDXHAOMM2QSKIQQBJK6I2FB7YKDZES5UCLWD"
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP2,
	)
	assert.NoError(t, err)

	signersFound, err := VerifyChallengeTxSigners(tx64, serverKP.Address(), network.TestNetworkPassphrase, clientKP2.Address(), preauthTxHash, xHash, unknownSignerType)
	assert.Equal(t, []string{clientKP2.Address()}, signersFound)
	assert.NoError(t, err)
}

func TestVerifyChallengeTxSigners_invalidServerAndClientSignersIgnoresDuplicateSignerInError(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	clientKP2 := newKeypair2()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP2,
	)
	assert.NoError(t, err)

	signersFound, err := VerifyChallengeTxSigners(tx64, serverKP.Address(), network.TestNetworkPassphrase, clientKP.Address(), clientKP.Address())
	assert.Empty(t, signersFound)
	assert.EqualError(t, err, "transaction not signed by "+clientKP.Address())
}

func TestVerifyChallengeTxSigners_invalidServerAndClientSignersFailsDuplicateSignatures(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	clientKP2 := newKeypair2()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP2, clientKP2,
	)
	assert.NoError(t, err)

	signersFound, err := VerifyChallengeTxSigners(tx64, serverKP.Address(), network.TestNetworkPassphrase, clientKP2.Address())
	assert.Equal(t, []string{clientKP2.Address()}, signersFound)
	assert.EqualError(t, err, "transaction has unrecognized signatures")
}

func TestVerifyChallengeTxSigners_invalidServerAndClientSignersFailsSignerSeed(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	clientKP2 := newKeypair2()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP2,
	)
	assert.NoError(t, err)

	signersFound, err := VerifyChallengeTxSigners(tx64, serverKP.Address(), network.TestNetworkPassphrase, clientKP2.Seed())
	assert.Empty(t, signersFound)
	assert.EqualError(t, err, "no verifiable signers provided, at least one G... address must be provided")
}

func TestVerifyChallengeTxSigners_invalidNoSigners(t *testing.T) {
	serverKP := newKeypair0()
	clientKP := newKeypair1()
	txSource := NewSimpleAccount(serverKP.Address(), -1)
	opSource := NewSimpleAccount(clientKP.Address(), 0)
	op := ManageData{
		SourceAccount: &opSource,
		Name:          "testserver auth",
		Value:         []byte(base64.StdEncoding.EncodeToString(make([]byte, 48))),
	}
	tx64, err := newSignedTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
		network.TestNetworkPassphrase,
		serverKP, clientKP,
	)
	assert.NoError(t, err)

	_, err = VerifyChallengeTxSigners(tx64, serverKP.Address(), network.TestNetworkPassphrase)
	assert.EqualError(t, err, "no verifiable signers provided, at least one G... address must be provided")
}

func TestVerifyTxSignatureUnsignedTx(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	txSource := NewSimpleAccount(kp0.Address(), int64(9605939170639897))
	opSource := NewSimpleAccount(kp1.Address(), 0)
	createAccount := CreateAccount{
		Destination:   "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:        "10",
		SourceAccount: &opSource,
	}
	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&createAccount},
			BaseFee:              MinBaseFee,
			Timebounds:           NewTimeout(1000),
		},
	)
	assert.NoError(t, err)

	err = verifyTxSignature(tx, network.TestNetworkPassphrase, kp0.Address())
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "transaction not signed by GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3")
	}
}

func TestVerifyTxSignatureSingle(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	txSource := NewSimpleAccount(kp0.Address(), int64(9605939170639897))
	opSource := NewSimpleAccount(kp1.Address(), 0)
	createAccount := CreateAccount{
		Destination:   "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:        "10",
		SourceAccount: &opSource,
	}
	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&createAccount},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)

	tx, err = tx.Sign(network.TestNetworkPassphrase, kp0)
	assert.NoError(t, err)
	err = verifyTxSignature(tx, network.TestNetworkPassphrase, kp0.Address())
	assert.NoError(t, err)
}

func TestVerifyTxSignatureMultiple(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	txSource := NewSimpleAccount(kp0.Address(), int64(9605939170639897))
	opSource := NewSimpleAccount(kp1.Address(), 0)
	createAccount := CreateAccount{
		Destination:   "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:        "10",
		SourceAccount: &opSource,
	}
	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&createAccount},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)

	// verify tx with multiple signature
	tx, err = tx.Sign(network.TestNetworkPassphrase, kp0, kp1)
	assert.NoError(t, err)
	err = verifyTxSignature(tx, network.TestNetworkPassphrase, kp0.Address())
	assert.NoError(t, err)
	err = verifyTxSignature(tx, network.TestNetworkPassphrase, kp1.Address())
	assert.NoError(t, err)
}
func TestVerifyTxSignatureInvalid(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	txSource := NewSimpleAccount(kp0.Address(), int64(9605939170639897))
	opSource := NewSimpleAccount(kp1.Address(), 0)
	createAccount := CreateAccount{
		Destination:   "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:        "10",
		SourceAccount: &opSource,
	}
	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&createAccount},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)

	// verify invalid signer
	tx, err = tx.Sign(network.TestNetworkPassphrase, kp0, kp1)
	assert.NoError(t, err)
	err = verifyTxSignature(tx, network.TestNetworkPassphrase, "GATBMIXTHXYKSUZSZUEJKACZ2OS2IYUWP2AIF3CA32PIDLJ67CH6Y5UY")
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "transaction not signed by GATBMIXTHXYKSUZSZUEJKACZ2OS2IYUWP2AIF3CA32PIDLJ67CH6Y5UY")
	}
}
