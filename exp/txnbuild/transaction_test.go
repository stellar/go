package txnbuild

import (
	"testing"

	"github.com/stellar/go/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		Asset:       NewNativeAsset(),
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

func TestPaymentFailsIfNoAssetSpecified(t *testing.T) {
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

	err := tx.Build()
	expectedErrMsg := "Failed to build operation *txnbuild.Payment: You must specify an asset for payment"
	require.EqualError(t, err, expectedErrMsg, "An asset is required")
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

func TestAccountMerge(t *testing.T) {
	kp0 := newKeypair0()

	sourceAccount := Account{
		ID:             kp0.Address(),
		SequenceNumber: 40385577484298,
	}

	accountMerge := AccountMerge{
		Destination: "GAS4V4O2B7DW5T7IQRPEEVCRXMDZESKISR7DVIGKZQYYV3OSQ5SH5LVP",
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&accountMerge},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAALAAAAAAAAAAAAAAABAAAAAAAAAAgAAAAAJcrx2g/Hbs/ohF5CVFG7B5JJSJR+OqDKzDGK7dKHZH4AAAAAAAAAAeoucsUAAABAz5wZN8BluFTXbzGyKYTrQJayT/8Ze5tForHjgkXwY9fIB/hINwHHQ+2wdBN5v6tvA1L6dfS76AytudjkX8CjDg=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestManageData(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := Account{
		ID:             kp0.Address(),
		SequenceNumber: 40385577484298,
	}

	manageData := ManageData{
		Name:  "Fruit preference",
		Value: []byte("Apple"),
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&manageData},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAALAAAAAAAAAAAAAAABAAAAAAAAAAoAAAAQRnJ1aXQgcHJlZmVyZW5jZQAAAAEAAAAFQXBwbGUAAAAAAAAAAAAAAeoucsUAAABAncYXM9JYk3FN1rcmjN58P1SoWHgCYSK1ckueZF4Ii7f42HZX5+z/h3CjxhCCwA7QK6s4uZ4n5ba3Ujh0x27YAQ=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}
func TestManageDataRemoveDataEntry(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := Account{
		ID:             kp0.Address(),
		SequenceNumber: 40385577484309,
	}

	manageData := ManageData{
		Name: "Fruit preference",
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
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
	sourceAccount := Account{
		ID:             kp0.Address(),
		SequenceNumber: 40385577484315,
	}

	setOptions := SetOptions{
		InflationDestination: NewInflationDestination(kp1.Address()),
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&setOptions},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAcAAAAAAAAAAAAAAABAAAAAAAAAAUAAAABAAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAeoucsUAAABAR/HVP3lr4CiR669LU1FZjO1uBQO36TduvYzOnSy786eNNNx+rSEhAt/w1iBdK9fKL8uw9FM+YH4eWOEixRu0Dw=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsSetFlags(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := Account{
		ID:             kp0.Address(),
		SequenceNumber: 40385577484318,
	}

	setOptions := SetOptions{
		SetFlags: []AccountFlag{AuthRequired, AuthRevocable},
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&setOptions},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAfAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAAAAAAEAAAADAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHqLnLFAAAAQJ5MwX8wWHVyF/QhY9qkD9+NoSGf9TH1dyfHxc2l9jL3/1sw8cgNYx1XRAEpaMq9BZtpZ0+zLjc0TAq2B+jSKAM="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsClearFlags(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := Account{
		ID:             kp0.Address(),
		SequenceNumber: 40385577484319,
	}

	setOptions := SetOptions{
		ClearFlags: []AccountFlag{AuthRequired, AuthRevocable},
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&setOptions},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAgAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAMAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHqLnLFAAAAQK2hb0/FTkNzS/C7CAWbrlgo6Wx5lJZdbt6cup723nGlGrkz92pvcrOQLZUBH3akI9Zdin51Wk4dvihghBFrcA8="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsMasterWeight(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := Account{
		ID:             kp0.Address(),
		SequenceNumber: 40385577484320,
	}

	setOptions := SetOptions{
		MasterWeight: NewThreshold(10),
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&setOptions},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAhAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAAAAAAAAAAABAAAACgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHqLnLFAAAAQOjpX3xs5uRACzzIJ9JZYyYjTd3kdEhhNNEwTJPS3jqd+gnwefJ/HKsHCL3S6WociUyn1B6nlhO63ZIu/+SPTwc="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsThresholds(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := Account{
		ID:             kp0.Address(),
		SequenceNumber: 40385577484322,
	}

	setOptions := SetOptions{
		LowThreshold:    NewThreshold(1),
		MediumThreshold: NewThreshold(2),
		HighThreshold:   NewThreshold(2),
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&setOptions},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAjAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAEAAAABAAAAAgAAAAEAAAACAAAAAAAAAAAAAAAAAAAAAeoucsUAAABArWZCMkVyzoKl3ZAh4Pu+7/iy45ffPiC525qXWrFdWcC0NC18SMwg96gmamyIilDxCeN+8Xn+WzhziaSAbGbdBg=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsHomeDomain(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := Account{
		ID:             kp0.Address(),
		SequenceNumber: 40385577484325,
	}

	setOptions := SetOptions{
		HomeDomain: NewHomeDomain("LovelyLumensLookLuminous.com"),
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&setOptions},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAmAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAABxMb3ZlbHlMdW1lbnNMb29rTHVtaW5vdXMuY29tAAAAAAAAAAAAAAAB6i5yxQAAAEAXjzYPYoUdQ617Ltn4wwefJLuy0P3S3dOeFTOWlZxi9KeKsVgqOQ+B+hms2JdpSWRodr0N0Nj6LsZhTjLbv4wO"
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestSetOptionsHomeDomainTooLong(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := Account{
		ID:             kp0.Address(),
		SequenceNumber: 40385577484323,
	}

	setOptions := SetOptions{
		HomeDomain: NewHomeDomain("LovelyLumensLookLuminousLately.com"),
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&setOptions},
		Network:       network.TestNetworkPassphrase,
	}

	err := tx.Build()
	assert.Error(t, err, "A validation error was expected (home domain > 32 chars)")
}

func TestSetOptionsSigner(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := Account{
		ID:             kp0.Address(),
		SequenceNumber: 40385577484325,
	}

	setOptions := SetOptions{
		Signer: &Signer{Address: kp1.Address(), Weight: Threshold(4)},
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&setOptions},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAAmAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAJcrx2g/Hbs/ohF5CVFG7B5JJSJR+OqDKzDGK7dKHZH4AAAAEAAAAAAAAAAHqLnLFAAAAQB1P8K0BXzpWdiXwBoMkGLJ8V/HhFQkq+NXmf7DhFVOHQid8Rz2K9cGvlclXWfUqKB60niWlCPTFtmzrKpWVTQ0="
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

func TestChangeTrust(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()

	sourceAccount := Account{
		ID:             kp0.Address(),
		SequenceNumber: 40385577484348,
	}

	changeTrust := ChangeTrust{
		Line:  NewAsset("ABCD", kp1.Address()),
		Limit: "10",
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&changeTrust},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAAA9AAAAAAAAAAAAAAABAAAAAAAAAAYAAAABQUJDRAAAAAAlyvHaD8duz+iEXkJUUbsHkklIlH46oMrMMYrt0odkfgAAAAAF9eEAAAAAAAAAAAHqLnLFAAAAQCOIEK9f3CMCfb5CzB2G2q6PBNx1P0R71v1hf8JXEIICXjWwy6hT140PP8EV4/VcARlA9a09a4Rr8dRNnpeOwAI="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}

func TestChangeTrustNativeAssetNotAllowed(t *testing.T) {
	kp0 := newKeypair0()

	sourceAccount := Account{
		ID:             kp0.Address(),
		SequenceNumber: 40385577484348,
	}

	changeTrust := ChangeTrust{
		Line:  NewNativeAsset(),
		Limit: "10",
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&changeTrust},
		Network:       network.TestNetworkPassphrase,
	}

	err := tx.Build()
	expectedErrMsg := "Failed to build operation *txnbuild.ChangeTrust: Trustline cannot be extended to a native (XLM) asset"
	require.EqualError(t, err, expectedErrMsg, "No trustlines for native assets")
}

func TestChangeTrustDeleteTrustline(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()

	sourceAccount := Account{
		ID:             kp0.Address(),
		SequenceNumber: 40385577484354,
	}

	issuedAsset := NewAsset("ABCD", kp1.Address())
	removeTrust := NewRemoveTrustlineOp(issuedAsset)

	tx := Transaction{
		SourceAccount: sourceAccount,
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

	sourceAccount := Account{
		ID:             kp0.Address(),
		SequenceNumber: 40385577484366,
	}

	issuedAsset := NewAsset("ABCD", kp1.Address())
	allowTrust := AllowTrust{
		Trustor:   kp1.Address(),
		Type:      issuedAsset,
		Authorize: true,
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{&allowTrust},
		Network:       network.TestNetworkPassphrase,
	}

	received := buildSignEncode(tx, kp0, t)
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAAJLsAAABPAAAAAAAAAAAAAAABAAAAAAAAAAcAAAAAJcrx2g/Hbs/ohF5CVFG7B5JJSJR+OqDKzDGK7dKHZH4AAAABQUJDRAAAAAEAAAAAAAAAAeoucsUAAABAlP4A5hdKUQU18MY6wmf4GugGNnCUklsV9/aRoTv8Q2yw7skm5nkFExnjhgEya6AM7iCR6oaf2C0VhrU4oEEODQ=="
	assert.Equal(t, expected, received, "Base 64 XDR should match")
}
