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
	sourceAccount := makeTestAccount(kp0, "9605939170639897")

	inflation := Inflation{}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&inflation},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAaAAAAAAAAAAAAAAABAAAAAAAAAAkAAAAAAAAAAeoucsUAAABAWqznvTxLfn6Q+zIloGmLDXCJQWsFPlfIf/EVFF+FfpL/gNbsvTC/U2G/ZtxMTgvqTLsBJfZAailGvPS04rfYCw=="
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
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAaAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAhODe2rwbSS+e+giFk8xgmxj70pVXzEADo3GG0rEhdlQAAAAABfXhAAAAAAAAAAAB6i5yxQAAAEBa4swhXSxQ2SYXoT0FcwIrrslFrv/Q/pnXK2+f6XigqjxW0yjNQwIrpVZuNz4zNGXB3DULxyYkUi8wDwwbiKIB"
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
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAbAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAfhHLNNY19eGrAtSgLD3VpaRm2AjNjxIBWQg9zS4VWZgAAAAAAAAAAAX14QAAAAAAAAAAAeoucsUAAABA5rSL7gy8OGiMq2Rocvv6l6HwOdePwhIMw2aJ2j5mVumAmeADjMeeCcGQIj3A7bISo6eWoF49w3qcd7uBS4j6AQ=="
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
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp1, t)
	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAiILoAAAAIAAAAAAAAAAAAAAABAAAAAAAAAAsAIiC6AAAAbAAAAAAAAAAB0odkfgAAAEDLsgDc3tPETqlKxVMF16UePDbSXQ1X0i5b3U3DRHDEchU91YwsDb4oMZrCj0mwKhkiXzCUyg9pPmUG/vKtQVQD"
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
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAALAAAAAAAAAAAAAAABAAAAAAAAAAgAAAAAJcrx2g/Hbs/ohF5CVFG7B5JJSJR+OqDKzDGK7dKHZH4AAAAAAAAAAeoucsUAAABAz5wZN8BluFTXbzGyKYTrQJayT/8Ze5tForHjgkXwY9fIB/hINwHHQ+2wdBN5v6tvA1L6dfS76AytudjkX8CjDg=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestManageData(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := makeTestAccount(kp0, "40385577484298")

	manageData := ManageData{
		Name:  "Fruit preference",
		Value: []byte("Apple"),
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&manageData},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAALAAAAAAAAAAAAAAABAAAAAAAAAAoAAAAQRnJ1aXQgcHJlZmVyZW5jZQAAAAEAAAAFQXBwbGUAAAAAAAAAAAAAAeoucsUAAABAncYXM9JYk3FN1rcmjN58P1SoWHgCYSK1ckueZF4Ii7f42HZX5+z/h3CjxhCCwA7QK6s4uZ4n5ba3Ujh0x27YAQ=="
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
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAWAAAAAAAAAAAAAAABAAAAAAAAAAoAAAAQRnJ1aXQgcHJlZmVyZW5jZQAAAAAAAAAAAAAAAeoucsUAAABAvxTjMVAHpIn8EJOznQ5ffLccnaEP1HJcHP/FkVMGzRvtSUOj/F55ABajmUe/WteiU7eJgzbKkgHvIMv1JB5XBw=="
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
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAcAAAAAAAAAAAAAAABAAAAAAAAAAUAAAABAAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAeoucsUAAABAR/HVP3lr4CiR669LU1FZjO1uBQO36TduvYzOnSy786eNNNx+rSEhAt/w1iBdK9fKL8uw9FM+YH4eWOEixRu0Dw=="
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
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAfAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAAAAAAEAAAADAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHqLnLFAAAAQJ5MwX8wWHVyF/QhY9qkD9+NoSGf9TH1dyfHxc2l9jL3/1sw8cgNYx1XRAEpaMq9BZtpZ0+zLjc0TAq2B+jSKAM="
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
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAgAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAMAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHqLnLFAAAAQK2hb0/FTkNzS/C7CAWbrlgo6Wx5lJZdbt6cup723nGlGrkz92pvcrOQLZUBH3akI9Zdin51Wk4dvihghBFrcA8="
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
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAhAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAAAAAAAAAAABAAAACgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHqLnLFAAAAQOjpX3xs5uRACzzIJ9JZYyYjTd3kdEhhNNEwTJPS3jqd+gnwefJ/HKsHCL3S6WociUyn1B6nlhO63ZIu/+SPTwc="
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
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAjAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAEAAAABAAAAAgAAAAEAAAACAAAAAAAAAAAAAAAAAAAAAeoucsUAAABArWZCMkVyzoKl3ZAh4Pu+7/iy45ffPiC525qXWrFdWcC0NC18SMwg96gmamyIilDxCeN+8Xn+WzhziaSAbGbdBg=="
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
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAmAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAABxMb3ZlbHlMdW1lbnNMb29rTHVtaW5vdXMuY29tAAAAAAAAAAAAAAAB6i5yxQAAAEAXjzYPYoUdQ617Ltn4wwefJLuy0P3S3dOeFTOWlZxi9KeKsVgqOQ+B+hms2JdpSWRodr0N0Nj6LsZhTjLbv4wO"
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
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAmAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAJcrx2g/Hbs/ohF5CVFG7B5JJSJR+OqDKzDGK7dKHZH4AAAAEAAAAAAAAAAHqLnLFAAAAQB1P8K0BXzpWdiXwBoMkGLJ8V/HhFQkq+NXmf7DhFVOHQid8Rz2K9cGvlclXWfUqKB60niWlCPTFtmzrKpWVTQ0="
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
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp1, t)
	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAyAAiILoAAAAIAAAAAAAAAAAAAAACAAAAAAAAAAkAAAAAAAAACwAiILoAAABsAAAAAAAAAAHSh2R+AAAAQGx5xAPuF3rH3/KSHXduYYvE/Qw4CAseF2F0oSacIYi8e320OW07lr9VF8XEcDqMSVNhkFopoh5P0ZSixcTxyQI="
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
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAA9AAAAAAAAAAAAAAABAAAAAAAAAAYAAAABQUJDRAAAAAAlyvHaD8duz+iEXkJUUbsHkklIlH46oMrMMYrt0odkfgAAAAAF9eEAAAAAAAAAAAHqLnLFAAAAQCOIEK9f3CMCfb5CzB2G2q6PBNx1P0R71v1hf8JXEIICXjWwy6hT140PP8EV4/VcARlA9a09a4Rr8dRNnpeOwAI="
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
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAABDAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABQUJDRAAAAAAlyvHaD8duz+iEXkJUUbsHkklIlH46oMrMMYrt0odkfgAAAAAAAAAAAAAAAAAAAAHqLnLFAAAAQEop/qQ5+2GTSQxZWzL4BPKsAi47VVNxnbtWgSAZvJOqz0yG0GJaTpUUYskuEo1haBg0UDbQF4M0PIK4l0Pzegg="
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
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAABPAAAAAAAAAAAAAAABAAAAAAAAAAcAAAAAJcrx2g/Hbs/ohF5CVFG7B5JJSJR+OqDKzDGK7dKHZH4AAAABQUJDRAAAAAEAAAAAAAAAAeoucsUAAABAlP4A5hdKUQU18MY6wmf4GugGNnCUklsV9/aRoTv8Q2yw7skm5nkFExnjhgEya6AM7iCR6oaf2C0VhrU4oEEODQ=="
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
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp1, t)
	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAAJWoAAAAFAAAAAAAAAAAAAAABAAAAAAAAAAMAAAAAAAAAAUFCQ0QAAAAA4Nxt4XJcrGZRYrUvrOc1sooiQ+QdEk1suS1wo+oucsUAAAAAO5rKAAAAAAEAAABkAAAAAAAAAAAAAAAAAAAAAdKHZH4AAABAe/TZt+6EAWp8BxbOa+x8xZ+oKF83SKghhzfMaih0gn9Ark2kE+ZOdiftY+DDjLF8RVzbzWGFvHgGBCt5pY5lCg=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestManageOfferDeleteOffer(t *testing.T) {
	kp1 := newKeypair1()
	sourceAccount := makeTestAccount(kp1, "41137196761105")

	offerID := uint64(2921622)
	deleteOffer := DeleteOfferOp(offerID)

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&deleteOffer},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp1, t)
	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAAJWoAAAASAAAAAAAAAAAAAAABAAAAAAAAAAMAAAAAAAAAAUZBS0UAAAAAQQeAZMSVhmLzYCQaIl1KNrY4FpTZoRzDCncBje0UnbEAAAAAAAAAAAAAAAEAAAABAAAAAAAslJYAAAAAAAAAAdKHZH4AAABAkj1T85v1atBk0k0QenWxbcDxRAJs3PdkijBFFGVGhGJYcaMdQoEBpvb8hJEzpaJ/feK9pa00YCMGyizGfr4rDw=="
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
	offerID := uint64(2497628)
	updateOffer := UpdateOfferOp(selling, buying, sellAmount, price, offerID)

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&updateOffer},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp1, t)
	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAAJWoAAAAKAAAAAAAAAAAAAAABAAAAAAAAAAMAAAAAAAAAAUFCQ0QAAAAA4Nxt4XJcrGZRYrUvrOc1sooiQ+QdEk1suS1wo+oucsUAAAAAHc1lAAAAAAEAAAAyAAAAAAAmHFwAAAAAAAAAAdKHZH4AAABA7j/x1HuvyMiH9Q59sjLmFLak76hJGQvjx6ckTzuuI0tpBrB/7Wfra8JrWrzajTJGMoQGwdDND5rEi/jTxWMjCQ=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestCreatePassiveOffer(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := makeTestAccount(kp1, "41137196761100")

	createPassiveOffer := CreatePassiveOffer{
		Selling: NativeAsset{},
		Buying:  CreditAsset{"ABCD", kp0.Address()},
		Amount:  "10",
		Price:   "1.0"}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&createPassiveOffer},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp1, t)
	expected := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAAJWoAAAANAAAAAAAAAAAAAAABAAAAAAAAAAQAAAAAAAAAAUFCQ0QAAAAA4Nxt4XJcrGZRYrUvrOc1sooiQ+QdEk1suS1wo+oucsUAAAAABfXhAAAAAAEAAAABAAAAAAAAAAHSh2R+AAAAQIDB0yw4eH14RDnUI4Ef5eyTbkRYl2adTPAOgbZmodkhOsmXOZITw1B6RnwdDCIRSLk2ZPvq2FU8Mk50l0eK+Ag="
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
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp2, t)
	// https://www.stellar.org/laboratory/#xdr-viewer?input=AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAAql0AAAADAAAAAAAAAAAAAAABAAAAAAAAAAIAAAAAAAAAAAX14QAAAAAAfhHLNNY19eGrAtSgLD3VpaRm2AjNjxIBWQg9zS4VWZgAAAAAAAAAAACYloAAAAABAAAAAUFCQ0QAAAAA4Nxt4XJcrGZRYrUvrOc1sooiQ%2BQdEk1suS1wo%2BoucsUAAAAAAAAAAS4VWZgAAABAZBS66leC0Y7UMg6jPYWh04lLWW9cLOdjWKKIWCjBTwRPmRhb5KyVsRepZdAvl8jmaLnbTk20uJ1yWbenbbbqCw%3D%3D%0A&type=TransactionEnvelope&network=test
	expected := "AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAAql0AAAADAAAAAAAAAAAAAAABAAAAAAAAAAIAAAAAAAAAAAX14QAAAAAAfhHLNNY19eGrAtSgLD3VpaRm2AjNjxIBWQg9zS4VWZgAAAAAAAAAAACYloAAAAABAAAAAUFCQ0QAAAAA4Nxt4XJcrGZRYrUvrOc1sooiQ+QdEk1suS1wo+oucsUAAAAAAAAAAS4VWZgAAABAZBS66leC0Y7UMg6jPYWh04lLWW9cLOdjWKKIWCjBTwRPmRhb5KyVsRepZdAvl8jmaLnbTk20uJ1yWbenbbbqCw=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestMemoText(t *testing.T) {
	kp2 := newKeypair2()
	sourceAccount := makeTestAccount(kp2, "3428320205078528")

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Network:       network.TestNetworkPassphrase,
		Operations:    []Operation{&BumpSequence{BumpTo: 1}},
		Memo:          MemoText("Twas brillig"),
	}

	received := buildSignEncode(tx, kp2, t)
	// https://www.stellar.org/laboratory/#xdr-viewer?input=AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAMLgoAAAABAAAAAAAAAAEAAAAMVHdhcyBicmlsbGlnAAAAAQAAAAAAAAALAAAAAAAAAAEAAAAAAAAAAS4VWZgAAABAstxxDHhcXkfmDkHbe2ck2QFjh6w69VlBzqOeHbT0p0ZxS6cQrhlFZBdvBb4T5qlo0RF4D06z04ygqDqrXmiSDg%3D%3D&type=TransactionEnvelope&network=test
	expected := "AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAMLgoAAAABAAAAAAAAAAEAAAAMVHdhcyBicmlsbGlnAAAAAQAAAAAAAAALAAAAAAAAAAEAAAAAAAAAAS4VWZgAAABAstxxDHhcXkfmDkHbe2ck2QFjh6w69VlBzqOeHbT0p0ZxS6cQrhlFZBdvBb4T5qlo0RF4D06z04ygqDqrXmiSDg=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestMemoID(t *testing.T) {
	kp2 := newKeypair2()
	sourceAccount := makeTestAccount(kp2, "3428320205078528")

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Network:       network.TestNetworkPassphrase,
		Operations:    []Operation{&BumpSequence{BumpTo: 1}},
		Memo:          MemoID(314159),
	}

	received := buildSignEncode(tx, kp2, t)
	// https://www.stellar.org/laboratory/#xdr-viewer?input=AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAMLgoAAAABAAAAAAAAAAIAAAAAAATLLwAAAAEAAAAAAAAACwAAAAAAAAABAAAAAAAAAAEuFVmYAAAAQCKqa1rqle3g8Ksdvl9J67sKdHoXvVXgsmV2QVMZskO%2BDhGSnyxAZBjGf7MFWuz1JoXr5VMo0zphTBRjtMWQvAA%3D&type=TransactionEnvelope&network=test
	expected := "AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAMLgoAAAABAAAAAAAAAAIAAAAAAATLLwAAAAEAAAAAAAAACwAAAAAAAAABAAAAAAAAAAEuFVmYAAAAQCKqa1rqle3g8Ksdvl9J67sKdHoXvVXgsmV2QVMZskO+DhGSnyxAZBjGf7MFWuz1JoXr5VMo0zphTBRjtMWQvAA="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestMemoHash(t *testing.T) {
	kp2 := newKeypair2()
	sourceAccount := makeTestAccount(kp2, "3428320205078528")

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Network:       network.TestNetworkPassphrase,
		Operations:    []Operation{&BumpSequence{BumpTo: 1}},
		Memo:          MemoHash([32]byte{0x01}),
	}

	received := buildSignEncode(tx, kp2, t)
	// https://www.stellar.org/laboratory/#xdr-viewer?input=AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAMLgoAAAABAAAAAAAAAAMBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAACwAAAAAAAAABAAAAAAAAAAEuFVmYAAAAQJeRgV2MPt3E4IktlsDm6herfaR%2F5VTplcUUwFgBMbPyIxjZW8GEZAIUxjWBV7T9XWjzLrw7pEyldeOcC76PYwc%3D&type=TransactionEnvelope&network=test
	expected := "AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAMLgoAAAABAAAAAAAAAAMBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAACwAAAAAAAAABAAAAAAAAAAEuFVmYAAAAQJeRgV2MPt3E4IktlsDm6herfaR/5VTplcUUwFgBMbPyIxjZW8GEZAIUxjWBV7T9XWjzLrw7pEyldeOcC76PYwc="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestMemoReturn(t *testing.T) {
	kp2 := newKeypair2()
	sourceAccount := makeTestAccount(kp2, "3428320205078528")

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Network:       network.TestNetworkPassphrase,
		Operations:    []Operation{&BumpSequence{BumpTo: 1}},
		Memo:          MemoReturn([32]byte{0x01}),
	}

	received := buildSignEncode(tx, kp2, t)
	// https://www.stellar.org/laboratory/#xdr-viewer?input=AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAMLgoAAAABAAAAAAAAAAQBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAACwAAAAAAAAABAAAAAAAAAAEuFVmYAAAAQNhrY46fggs%2BTnOYvh3ILgWqmXjkW0968s00si5RLdxFh2%2FA7TTGgmBTarTEtF21hsAyNmW%2B0YkqVVzJ7eFAXAk%3D&type=TransactionEnvelope&network=test
	expected := "AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAMLgoAAAABAAAAAAAAAAQBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAACwAAAAAAAAABAAAAAAAAAAEuFVmYAAAAQNhrY46fggs+TnOYvh3ILgWqmXjkW0968s00si5RLdxFh2/A7TTGgmBTarTEtF21hsAyNmW+0YkqVVzJ7eFAXAk="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}
