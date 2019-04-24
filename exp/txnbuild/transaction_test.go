package txnbuild

import (
	"testing"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestAccount(kp *keypair.Full, seqnum string) horizon.Account {
	return horizon.Account{
		HistoryAccount: horizon.HistoryAccount{
			AccountID: kp.Address(),
		},
		Sequence: seqnum,
	}
}

func TestInflation(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := makeTestAccount(kp0, "3556091187167235")

	inflation := Inflation{}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&inflation},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	// https://www.stellar.org/laboratory/#xdr-viewer?input=AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAMoj8AAAAEAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAJAAAAAAAAAAHqLnLFAAAAQP3NHWXvzKIHB3%2BjjhHITdc%2FtBPntWYj3SoTjpON%2BdxjKqU5ohFamSHeqi5ONXkhE9Uajr5sVZXjQfUcTTzsWAA%3D&type=TransactionEnvelope
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAMoj8AAAAEAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAJAAAAAAAAAAHqLnLFAAAAQP3NHWXvzKIHB3+jjhHITdc/tBPntWYj3SoTjpON+dxjKqU5ohFamSHeqi5ONXkhE9Uajr5sVZXjQfUcTTzsWAA="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestCreateAccount(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := makeTestAccount(kp0, "9605939170639897")

	createAccount := CreateAccount{
		Destination: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:      "10",
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&createAccount},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAaAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAITg3tq8G0kvnvoIhZPMYJsY+9KVV8xAA6NxhtKxIXZUAAAAAAX14QAAAAAAAAAAAeoucsUAAABAHsyMojA0Q5MiNsR5X5AiNpCn9mlXmqluRsNpTniCR91M4U5TFmrrqVNLkU58/l+Y8hUPwidDTRSzLZKbMUL/Bw=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestPayment(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := makeTestAccount(kp0, "9605939170639898")

	payment := Payment{
		Destination: "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
		Amount:      "10",
		Asset:       NativeAsset{},
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&payment},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAbAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAABAAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAAAAAAAAF9eEAAAAAAAAAAAHqLnLFAAAAQNcGQpjNOFCLf9eEmobN+H8SNoDH/jMrfEFPX8kM212ST+TGfirEdXH77GJXvaWplfGKmE3B+UDwLuYLwO+KbQQ="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestPaymentFailsIfNoAssetSpecified(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := makeTestAccount(kp0, "9605939170639898")

	payment := Payment{
		Destination: "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
		Amount:      "10",
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&payment},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	err := tx.Build()
	expectedErrMsg := "failed to build operation *txnbuild.Payment: you must specify an asset for payment"
	require.EqualError(t, err, expectedErrMsg, "An asset is required")
}

func TestBumpSequence(t *testing.T) {
	kp1 := newKeypair1()
	sourceAccount := makeTestAccount(kp1, "9606132444168199")

	bumpSequence := BumpSequence{
		BumpTo: 9606132444168300,
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&bumpSequence},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp1, t)
	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAiILoAAAAIAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAALACIgugAAAGwAAAAAAAAAAdKHZH4AAABAndjSSWeACpbr0ROAEK6jw5CzHiL/rCDpa6AO05+raHDowSUJBckkwlEuCjbBoO/A06tZNRT1Per3liTQrc8fCg=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestAccountMerge(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := makeTestAccount(kp0, "40385577484298")

	accountMerge := AccountMerge{
		Destination: "GAS4V4O2B7DW5T7IQRPEEVCRXMDZESKISR7DVIGKZQYYV3OSQ5SH5LVP",
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&accountMerge},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAALAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAIAAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAAAAAAAHqLnLFAAAAQJ/UcOgE64+GQpwv0uXXa2jrKtFdmDsyZ6ZZ/udxryPS8cNCm2L784ixPYM4XRgkoQCdxC3YK8n5x5+CXLzrrwA="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestManageData(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := makeTestAccount(kp0, "3556091187167235")

	manageData := ManageData{
		Name:  "Fruit preference",
		Value: []byte("Apple"),
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&manageData},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	// https://www.stellar.org/laboratory/#txsigner?xdr=AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAMoj8AAAAEAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAKAAAAEEZydWl0IHByZWZlcmVuY2UAAAABAAAABUFwcGxlAAAAAAAAAAAAAAA%3D
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAMoj8AAAAEAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAKAAAAEEZydWl0IHByZWZlcmVuY2UAAAABAAAABUFwcGxlAAAAAAAAAAAAAAHqLnLFAAAAQO1ELJBEoqBDyIsS7uSJwe1LOimV/E+09MyF1G/+yrxSggFVPEjD5LXcm/6POze3IsMuIYJU1et5Q2Vt9f73zQo="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestManageDataRemoveDataEntry(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := makeTestAccount(kp0, "40385577484309")

	manageData := ManageData{
		Name: "Fruit preference",
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&manageData},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAWAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAKAAAAEEZydWl0IHByZWZlcmVuY2UAAAAAAAAAAAAAAAHqLnLFAAAAQB8rkFZgtffUTdCASzwJ3jRcMCzHpVbbuFbye7Ki2dLao6u5d2aSzz3M2ugNJjNFMfSu3io9adCqwVKKjk0UJQA="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsInflationDestination(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := makeTestAccount(kp0, "40385577484315")

	setOptions := SetOptions{
		InflationDestination: NewInflationDestination(kp1.Address()),
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&setOptions},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAcAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAFAAAAAQAAAAAlyvHaD8duz+iEXkJUUbsHkklIlH46oMrMMYrt0odkfgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHqLnLFAAAAQB0RLe9DjdHzLM22whFja3ZT97L/818lvWpk5EOTETr9lmDH7/A0/EAzeCkTBzZMCi3C6pV1PrGBr0NJdRrPowg="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsSetFlags(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := makeTestAccount(kp0, "40385577484318")

	setOptions := SetOptions{
		SetFlags: []AccountFlag{AuthRequired, AuthRevocable},
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&setOptions},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAfAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAAAAAABAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB6i5yxQAAAECfYTppxtp1A2zSbb6VzkOkyk9D/7xjaXRxR+ZIqgdK3lWkHQRkjyVBj2yaI61J3trdp7CswImptjkjLprt0WIO"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsClearFlags(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := makeTestAccount(kp0, "40385577484319")

	setOptions := SetOptions{
		ClearFlags: []AccountFlag{AuthRequired, AuthRevocable},
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&setOptions},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAgAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAEAAAADAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB6i5yxQAAAEANXPAN+RgvqjGF0kJ6MyNTiMnWaELw5vYNwxhv8+mi3KmGWMzojCxcmMAqni0zBMsEjl9z7H8JT9x05OlQ9nsD"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsMasterWeight(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := makeTestAccount(kp0, "40385577484320")

	setOptions := SetOptions{
		MasterWeight: NewThreshold(10),
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&setOptions},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAhAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAAAAAAAAAAAAQAAAAoAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB6i5yxQAAAECIxH2W4XZ5fMsG658hdIEys2nlVSAK1FEjT5GADF6sWEThGFc+Wrmlw6GwKn6ZNAmxVULEgircjQx48aYSgFYD"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsThresholds(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := makeTestAccount(kp0, "40385577484322")

	setOptions := SetOptions{
		LowThreshold:    NewThreshold(1),
		MediumThreshold: NewThreshold(2),
		HighThreshold:   NewThreshold(2),
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&setOptions},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAjAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAABAAAAAQAAAAIAAAABAAAAAgAAAAAAAAAAAAAAAAAAAAHqLnLFAAAAQFwRcFbzEtxoxZOtWlOQld3nURHZugNj5faEncpv0X/dcrfiQVU7k3fkTYDskiVExFiq78CBsYAr0uuvfH61IQs="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsHomeDomain(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := makeTestAccount(kp0, "40385577484325")

	setOptions := SetOptions{
		HomeDomain: NewHomeDomain("LovelyLumensLookLuminous.com"),
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&setOptions},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAmAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAcTG92ZWx5THVtZW5zTG9va0x1bWlub3VzLmNvbQAAAAAAAAAAAAAAAeoucsUAAABAtC4HZzvRfyphRg5jjmz5jzBn86SANXCZS59GejRE8L1uCOxgXSEVoh1b+UetUEi7JN/n1ECBEVJrXgj0c34eBg=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsHomeDomainTooLong(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := makeTestAccount(kp0, "40385577484323")

	setOptions := SetOptions{
		HomeDomain: NewHomeDomain("LovelyLumensLookLuminousLately.com"),
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&setOptions},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	err := tx.Build()
	assert.Error(t, err, "A validation error was expected (home domain > 32 chars)")
}

func TestSetOptionsSigner(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := makeTestAccount(kp0, "40385577484325")

	setOptions := SetOptions{
		Signer: &Signer{Address: kp1.Address(), Weight: Threshold(4)},
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&setOptions},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAmAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAABAAAAAAAAAAB6i5yxQAAAEBfgmUK+wNj8ROz78Sg0rQ2s7lmtvA4r5epHkqc9yoxLDr/GSkmgWneVqoKNxWF0JB9L+Gql1+f8M8p1McF4MsB"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestMultipleOperations(t *testing.T) {
	kp1 := newKeypair1()
	sourceAccount := makeTestAccount(kp1, "9606132444168199")

	inflation := Inflation{}
	bumpSequence := BumpSequence{
		BumpTo: 9606132444168300,
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&inflation, &bumpSequence},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp1, t)
	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAyAAiILoAAAAIAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAgAAAAAAAAAJAAAAAAAAAAsAIiC6AAAAbAAAAAAAAAAB0odkfgAAAEDmf3Ag2Hw5NdlvzJpph4Km+aNKy8kfzS1EAhIVdKJwUnMVWhOpfdXSh/aekEVdoxXh2+ioocrxdtkWAZfS3sMF"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestChangeTrust(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := makeTestAccount(kp0, "40385577484348")

	changeTrust := ChangeTrust{
		Line:  CreditAsset{"ABCD", kp1.Address()},
		Limit: "10",
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&changeTrust},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAA9AAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAGAAAAAUFCQ0QAAAAAJcrx2g/Hbs/ohF5CVFG7B5JJSJR+OqDKzDGK7dKHZH4AAAAABfXhAAAAAAAAAAAB6i5yxQAAAED7YSd1VdewEdtEURAYuyCy8dWbzALEf1vJn88/gCER4CNdIvojOEafJEhYhzZJhdG7oa+95UjfI9vMJO8qdWMK"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestChangeTrustNativeAssetNotAllowed(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := makeTestAccount(kp0, "40385577484348")

	changeTrust := ChangeTrust{
		Line:  NativeAsset{},
		Limit: "10",
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&changeTrust},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	err := tx.Build()
	expectedErrMsg := "failed to build operation *txnbuild.ChangeTrust: trustline cannot be extended to a native (XLM) asset"
	require.EqualError(t, err, expectedErrMsg, "No trustlines for native assets")
}

func TestChangeTrustDeleteTrustline(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := makeTestAccount(kp0, "40385577484354")

	issuedAsset := CreditAsset{"ABCD", kp1.Address()}
	removeTrust := RemoveTrustlineOp(issuedAsset)

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&removeTrust},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAABDAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAGAAAAAUFCQ0QAAAAAJcrx2g/Hbs/ohF5CVFG7B5JJSJR+OqDKzDGK7dKHZH4AAAAAAAAAAAAAAAAAAAAB6i5yxQAAAECgd2wkK35civvf6NKpsSFDyKpdyo/cs7wL+RYfZ2BCP7eGrUUpu2GfQFtf/Hm6aBwT6nJ+dONTSPXnyp7Dq18L"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestAllowTrust(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := makeTestAccount(kp0, "40385577484366")

	issuedAsset := CreditAsset{"ABCD", kp1.Address()}
	allowTrust := AllowTrust{
		Trustor:   kp1.Address(),
		Type:      issuedAsset,
		Authorize: true,
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&allowTrust},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAABPAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAHAAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAAUFCQ0QAAAABAAAAAAAAAAHqLnLFAAAAQGGBSKitYxpHNMaVVOE2CIylWFJgwqxjhwnIvWauSSkLapntD18G1pMahLbs8Lqcr3+cEs5WjLI4eBhy6WiJhAk="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestManageOfferNewOffer(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := makeTestAccount(kp1, "41137196761092")

	selling := NativeAsset{}
	buying := CreditAsset{"ABCD", kp0.Address()}
	sellAmount := "100"
	price := "0.01"
	createOffer := CreateOfferOp(selling, buying, sellAmount, price)

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&createOffer},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp1, t)
	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAAJWoAAAAFAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAADAAAAAAAAAAFBQkNEAAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAADuaygAAAAABAAAAZAAAAAAAAAAAAAAAAAAAAAHSh2R+AAAAQAmXf4BnH8bWhy+Tnxf+7zgsij7pV0b7XC4rqfYWi9ZIVUaidWPbrFhaWjiQbXYB1NKdx0XjidzkcAgMInLqDgs="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestManageOfferDeleteOffer(t *testing.T) {
	kp1 := newKeypair1()
	sourceAccount := makeTestAccount(kp1, "41137196761105")

	offerID := int64(2921622)
	deleteOffer := DeleteOfferOp(offerID)

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&deleteOffer},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp1, t)
	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAAJWoAAAASAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAADAAAAAAAAAAFGQUtFAAAAAEEHgGTElYZi82AkGiJdSja2OBaU2aEcwwp3AY3tFJ2xAAAAAAAAAAAAAAABAAAAAQAAAAAALJSWAAAAAAAAAAHSh2R+AAAAQBSjRfpyEAIMnRQOPf1BBOx8HFC6Lm6bxxdljaegnUts8SmWJGQbZN5a8PQGzOTwGdBKBk9X9d+BIrBVc3kyyQ4="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestManageOfferUpdateOffer(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := makeTestAccount(kp1, "41137196761097")

	selling := NativeAsset{}
	buying := CreditAsset{"ABCD", kp0.Address()}
	sellAmount := "50"
	price := "0.02"
	offerID := int64(2497628)
	updateOffer := UpdateOfferOp(selling, buying, sellAmount, price, offerID)

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&updateOffer},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp1, t)
	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAAJWoAAAAKAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAADAAAAAAAAAAFBQkNEAAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAAB3NZQAAAAABAAAAMgAAAAAAJhxcAAAAAAAAAAHSh2R+AAAAQAwqWg2C/oe/zH4D3Y7/yg5SlHqFvF6A3j6GQZ9NPh3ROqutovLyAE62+rvXxM7hqSNz1Rtx4frJaOhOabh6DAg="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestCreatePassiveOffer(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := makeTestAccount(kp1, "41137196761100")

	createPassiveOffer := CreatePassiveSellOffer{
		Selling: NativeAsset{},
		Buying:  CreditAsset{"ABCD", kp0.Address()},
		Amount:  "10",
		Price:   "1.0"}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&createPassiveOffer},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp1, t)
	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAAJWoAAAANAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAEAAAAAAAAAAFBQkNEAAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAAAX14QAAAAABAAAAAQAAAAAAAAAB0odkfgAAAEAgUD7M1UL7x2m2m26ySzcSHxIneOT7/r+s/HLsgWDj6CmpSi1GZrlvtBH+CNuegCwvW09TRZJhp7bLywkaFCoK"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestPathPayment(t *testing.T) {
	kp0 := newKeypair0()
	kp2 := newKeypair2()
	sourceAccount := makeTestAccount(kp2, "187316408680450")

	abcdAsset := CreditAsset{"ABCD", kp0.Address()}
	pathPayment := PathPayment{
		SendAsset:   NativeAsset{},
		SendMax:     "10",
		Destination: kp2.Address(),
		DestAsset:   NativeAsset{},
		DestAmount:  "1",
		Path:        []Asset{abcdAsset},
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&pathPayment},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp2, t)
	expected := "AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAAql0AAAADAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAACAAAAAAAAAAAF9eEAAAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAAAAAAAAAmJaAAAAAAQAAAAFBQkNEAAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAAAAAAAEuFVmYAAAAQF2kLUL/RoFIy1cmt+GXdWn2tDUjJYV3YwF4A82zIBhqYSO6ogOoLPNRt3w+IGCAgfR4Q9lpax+wCXWoQERHSw4="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestMemoText(t *testing.T) {
	kp2 := newKeypair2()
	sourceAccount := makeTestAccount(kp2, "3556099777101824")

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&BumpSequence{BumpTo: 1}},
		Memo:          MemoText("Twas brillig"),
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp2, t)
	// https://www.stellar.org/laboratory/#txsigner?xdr=AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAMokEAAAABAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAABAAAADFR3YXMgYnJpbGxpZwAAAAEAAAAAAAAACwAAAAAAAAABAAAAAAAAAAA%3D&network=test
	expected := "AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAMokEAAAABAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAABAAAADFR3YXMgYnJpbGxpZwAAAAEAAAAAAAAACwAAAAAAAAABAAAAAAAAAAEuFVmYAAAAQILT8/7MGTmWkfjMi6Y23n2cVWs+IMY67xOskTivSZehp7wWaDXLIdCbdijmG64+Nz+fPBT9HYMqSRDcLiZYDQ0="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestMemoID(t *testing.T) {
	kp2 := newKeypair2()
	sourceAccount := makeTestAccount(kp2, "3428320205078528")

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&BumpSequence{BumpTo: 1}},
		Memo:          MemoID(314159),
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp2, t)
	expected := "AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAMLgoAAAABAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAACAAAAAAAEyy8AAAABAAAAAAAAAAsAAAAAAAAAAQAAAAAAAAABLhVZmAAAAEA5P/V/Veh6pjXj7CnqtWDATh8II+ci1z3/zmNk374XLuVLzx7jRve59AKnPMwIPwDJ8cXwEKz8+fYOIkfEI9AJ"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestMemoHash(t *testing.T) {
	kp2 := newKeypair2()
	sourceAccount := makeTestAccount(kp2, "3428320205078528")

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&BumpSequence{BumpTo: 1}},
		Memo:          MemoHash([32]byte{0x01}),
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp2, t)
	expected := "AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAMLgoAAAABAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAADAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAsAAAAAAAAAAQAAAAAAAAABLhVZmAAAAEAgauaUpqEGF1VeXYtkYg0I19QC3GJVrCPOqDHPIdXvGkQ9N+3Vt6yfKIN0sE/X5NuD6FhArQ3adwvZeaNDilwN"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestMemoReturn(t *testing.T) {
	kp2 := newKeypair2()
	sourceAccount := makeTestAccount(kp2, "3428320205078528")

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&BumpSequence{BumpTo: 1}},
		Memo:          MemoReturn([32]byte{0x01}),
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp2, t)
	expected := "AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAMLgoAAAABAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAEAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAsAAAAAAAAAAQAAAAAAAAABLhVZmAAAAEAuLFTunY08pbWKompoepHdazLmr7uePUSOzA4P33+SVRKWiu+h2tngOsP8hga+wpLJXT9l/0uMQ3iziRVUrh0K"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}
