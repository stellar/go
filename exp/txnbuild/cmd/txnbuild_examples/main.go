package main

// This is a scratch pad for testing new operations. Please DO NOT review!

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/exp/txnbuild"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/errors"
)

type key struct {
	Seed    string
	Address string
	Keypair *keypair.Full
}

func initKeys() []key {
	// Accounts created on testnet
	keys := []key{
		// test0
		key{Seed: "SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R",
			Address: "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3",
		},
		// test1
		key{Seed: "SBMSVD4KKELKGZXHBUQTIROWUAPQASDX7KEJITARP4VMZ6KLUHOGPTYW",
			Address: "GAS4V4O2B7DW5T7IQRPEEVCRXMDZESKISR7DVIGKZQYYV3OSQ5SH5LVP",
		},
		// test2
		key{Seed: "SBZVMB74Z76QZ3ZOY7UTDFYKMEGKW5XFJEB6PFKBF4UYSSWHG4EDH7PY",
			Address: "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H"},
		// dev-null
		key{Seed: "SD3ZKHOPXV6V2QPLCNNH7JWGKYWYKDFPFRNQSKSFF3Q5NJFPAB5VSO6D",
			Address: "GBAQPADEYSKYMYXTMASBUIS5JI3LMOAWSTM2CHGDBJ3QDDPNCSO3DVAA"},
	}

	for i, k := range keys {
		myKeypair, err := keypair.Parse(k.Seed)
		dieIfError("keypair didn't parse!", err)
		keys[i].Keypair = myKeypair.(*keypair.Full)
	}

	return keys
}

func main() {
	client := horizon.DefaultTestNetClient

	// resp := exampleCreateAccount(client, false)
	// resp := exampleSendLumens(client, false)
	// resp := exampleSendNonNative(client, false)
	// resp := exampleBumpSequence(client, false)
	// resp := exampleAccountMerge(client, true)
	// resp := exampleManageData(client, false)
	// resp := exampleManageDataRemoveDataEntry(client, false)
	// resp := exampleSetOptions(client, false)
	// resp := exampleChangeTrust(client, false)
	// resp := exampleChangeTrustDeleteTrustline(client, false)
	// resp := exampleAllowTrust(client, false)
	// resp := exampleManageOfferNewOffer(client, false)
	// resp := exampleManageOfferDeleteOffer(client, false)
	// resp := exampleManageOfferUpdateOffer(client, false)
	// resp := exampleCreatePassiveOffer(client, false)
	resp := examplePathPayment(client, false)
	fmt.Println(resp.TransactionSuccessToString())
}

func examplePathPayment(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	// test0 - distributor of ABCD token
	// test1 - has a trustline for ABCD
	// test2 - doesn't need a trustline for ABCD
	keys := initKeys()
	abcdAsset := txnbuild.CreditAsset{
		Code:   "ABCD",
		Issuer: keys[0].Address,
	}
	xlmAsset := txnbuild.NativeAsset{}

	// test0 creates an offer to sell ABCD
	horizonSourceAccount0, err := client.LoadAccount(keys[0].Address)
	dieIfError("loadaccount", err)
	sourceAccount0 := mapAccounts(horizonSourceAccount0)

	selling1 := &abcdAsset
	buying1 := xlmAsset
	sellAmount1 := "5"
	price1 := "4"
	offer1 := txnbuild.CreateOfferOp(selling1, buying1, sellAmount1, price1)

	// test1 creates an offer to buy ABCD
	horizonSourceAccount1, err := client.LoadAccount(keys[1].Address)
	dieIfError("loadaccount", err)
	sourceAccount1 := mapAccounts(horizonSourceAccount1)

	selling2 := xlmAsset
	buying2 := &abcdAsset
	sellAmount2 := "10"
	price2 := "1"
	offer2 := txnbuild.CreateOfferOp(selling2, buying2, sellAmount2, price2)

	// test2 performs the path payment
	horizonSourceAccount2, err := client.LoadAccount(keys[2].Address)
	dieIfError("loadaccount", err)
	sourceAccount2 := mapAccounts(horizonSourceAccount2)

	pathPayment := txnbuild.PathPayment{
		SendAsset:   xlmAsset,
		SendMax:     "10",
		Destination: keys[2].Address,
		DestAsset:   xlmAsset,
		DestAmount:  "1",
		Path:        []txnbuild.Asset{abcdAsset},
	}

	var resp horizon.TransactionSuccess
	// Submit the first offer
	tx1 := txnbuild.Transaction{
		SourceAccount: sourceAccount0,
		Operations:    []txnbuild.Operation{&offer1},
		Network:       network.TestNetworkPassphrase,
	}
	txeBase64_1 := buildSignEncode(tx1, keys[0].Keypair)
	log.Println("Base 64 TX: ", txeBase64_1)

	resp = submit(client, txeBase64_1, mock)
	fmt.Println(resp.TransactionSuccessToString())

	// Submit the second offer
	tx2 := txnbuild.Transaction{
		SourceAccount: sourceAccount1,
		Operations:    []txnbuild.Operation{&offer2},
		Network:       network.TestNetworkPassphrase,
	}
	txeBase64_2 := buildSignEncode(tx2, keys[1].Keypair)
	log.Println("Base 64 TX: ", txeBase64_2)

	resp = submit(client, txeBase64_2, mock)
	fmt.Println(resp.TransactionSuccessToString())

	// Submit the path payment
	tx3 := txnbuild.Transaction{
		SourceAccount: sourceAccount2,
		Operations:    []txnbuild.Operation{&pathPayment},
		Network:       network.TestNetworkPassphrase,
	}
	txeBase64_3 := buildSignEncode(tx3, keys[2].Keypair)
	log.Println("Base 64 TX: ", txeBase64_3)

	resp = submit(client, txeBase64_3, mock)

	return resp
}

func exampleCreatePassiveOffer(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	keys := initKeys()
	horizonSourceAccount, err := client.LoadAccount(keys[1].Address)
	dieIfError("loadaccount", err)
	sourceAccount := mapAccounts(horizonSourceAccount)

	selling := txnbuild.NativeAsset{}
	buying := txnbuild.CreditAsset{
		Code:   "ABCD",
		Issuer: keys[0].Address,
	}
	sellAmount := "10"
	price := "1.0"

	createPassiveOffer := txnbuild.CreatePassiveOffer{
		Selling: selling,
		Buying:  &buying,
		Amount:  sellAmount,
		Price:   price}

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		Operations:    []txnbuild.Operation{&createPassiveOffer},
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64 := buildSignEncode(tx, keys[1].Keypair)
	log.Println("Base 64 TX: ", txeBase64)

	return submit(client, txeBase64, mock)
}

func exampleManageOfferUpdateOffer(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	keys := initKeys()
	horizonSourceAccount, err := client.LoadAccount(keys[1].Address)
	dieIfError("loadaccount", err)
	sourceAccount := mapAccounts(horizonSourceAccount)

	selling := txnbuild.NativeAsset{}
	buying := txnbuild.CreditAsset{
		Code:   "ABCD",
		Issuer: keys[0].Address,
	}
	sellAmount := "50"
	price := "0.02"
	offerID := uint64(2497628)

	updateOffer := txnbuild.UpdateOfferOp(selling, &buying, sellAmount, price, offerID)

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		Operations:    []txnbuild.Operation{&updateOffer},
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64 := buildSignEncode(tx, keys[1].Keypair)
	log.Println("Base 64 TX: ", txeBase64)

	return submit(client, txeBase64, mock)
}

func exampleManageOfferDeleteOffer(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	keys := initKeys()
	horizonSourceAccount, err := client.LoadAccount(keys[1].Address)
	dieIfError("loadaccount", err)
	sourceAccount := mapAccounts(horizonSourceAccount)

	offerID := uint64(4326054)

	deleteOffer := txnbuild.DeleteOfferOp(offerID)

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		Operations:    []txnbuild.Operation{&deleteOffer},
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64 := buildSignEncode(tx, keys[1].Keypair)
	log.Println("Base 64 TX: ", txeBase64)

	return submit(client, txeBase64, mock)
}

func exampleManageOfferNewOffer(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	keys := initKeys()
	horizonSourceAccount, err := client.LoadAccount(keys[1].Address)
	dieIfError("loadaccount", err)
	sourceAccount := mapAccounts(horizonSourceAccount)

	selling := txnbuild.NativeAsset{}
	buying := txnbuild.CreditAsset{
		Code:   "ABCD",
		Issuer: keys[0].Address,
	}
	sellAmount := "100"
	price := "0.01"

	createOffer := txnbuild.CreateOfferOp(selling, &buying, sellAmount, price)

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		Operations:    []txnbuild.Operation{&createOffer},
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64 := buildSignEncode(tx, keys[1].Keypair)
	log.Println("Base 64 TX: ", txeBase64)

	return submit(client, txeBase64, mock)
}

func exampleAllowTrust(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	keys := initKeys()
	horizonSourceAccount, err := client.LoadAccount(keys[0].Address)
	dieIfError("loadaccount", err)
	sourceAccount := mapAccounts(horizonSourceAccount)

	issued := txnbuild.CreditAsset{
		Code:   "ABCD",
		Issuer: keys[0].Address,
	}
	allowTrust := txnbuild.AllowTrust{
		Trustor:   keys[1].Address,
		Type:      issued,
		Authorize: true,
	}

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		Operations:    []txnbuild.Operation{&allowTrust},
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64 := buildSignEncode(tx, keys[0].Keypair)
	log.Println("Base 64 TX: ", txeBase64)

	return submit(client, txeBase64, mock)
}

func exampleChangeTrust(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	keys := initKeys()
	horizonSourceAccount, err := client.LoadAccount(keys[1].Address)
	dieIfError("loadaccount", err)
	sourceAccount := mapAccounts(horizonSourceAccount)

	issuer := txnbuild.CreditAsset{
		Code:   "ABCD",
		Issuer: keys[0].Address,
	}
	changeTrust := txnbuild.ChangeTrust{
		Line:  issuer,
		Limit: "1000",
	}

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		Operations:    []txnbuild.Operation{&changeTrust},
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64 := buildSignEncode(tx, keys[1].Keypair)
	log.Println("Base 64 TX: ", txeBase64)

	return submit(client, txeBase64, mock)
}

func exampleChangeTrustDeleteTrustline(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	keys := initKeys()
	horizonSourceAccount, err := client.LoadAccount(keys[0].Address)
	dieIfError("loadaccount", err)
	sourceAccount := mapAccounts(horizonSourceAccount)

	issuedAsset := txnbuild.CreditAsset{
		Code:   "ABCD",
		Issuer: keys[1].Address,
	}
	removeTrust := txnbuild.RemoveTrustlineOp(issuedAsset)

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		Operations:    []txnbuild.Operation{&removeTrust},
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64 := buildSignEncode(tx, keys[0].Keypair)
	log.Println("Base 64 TX: ", txeBase64)

	return submit(client, txeBase64, mock)
}

func exampleSetOptions(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	keys := initKeys()

	horizonSourceAccount, err := client.LoadAccount(keys[0].Address)
	dieIfError("loadaccount", err)
	sourceAccount := mapAccounts(horizonSourceAccount)

	setOptions := txnbuild.SetOptions{
		// InflationDestination: keys[1].Address,
		// ClearFlags: []txnbuild.AccountFlag{txnbuild.AuthRequired, txnbuild.AuthRevocable},
		SetFlags: []txnbuild.AccountFlag{txnbuild.AuthRequired, txnbuild.AuthRevocable},
		// MasterWeight: txnbuild.NewThreshold(255),
		// LowThreshold:    txnbuild.NewThreshold(1),
		// MediumThreshold: txnbuild.NewThreshold(2),
		// HighThreshold:   txnbuild.NewThreshold(2),
		// HomeDomain: txnbuild.NewHomeDomain("LovelyLumensLookLuminousLately.com"),
		// Signer: &txnbuild.Signer{Address: keys[1].Address, Weight: *txnbuild.NewThreshold(0)},
	}

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		Operations:    []txnbuild.Operation{&setOptions},
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64 := buildSignEncode(tx, keys[0].Keypair)
	log.Println("Base 64 TX: ", txeBase64)

	return submit(client, txeBase64, mock)
}

func exampleManageDataRemoveDataEntry(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	keys := initKeys()

	horizonSourceAccount, err := client.LoadAccount(keys[0].Address)
	dieIfError("loadaccount", err)
	sourceAccount := mapAccounts(horizonSourceAccount)

	manageData := txnbuild.ManageData{
		Name: "Fruit preference",
		// Value: nil,
	}

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		Operations:    []txnbuild.Operation{&manageData},
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64 := buildSignEncode(tx, keys[0].Keypair)
	log.Println("Base 64 TX: ", txeBase64)

	return submit(client, txeBase64, mock)
}

func exampleManageData(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	keys := initKeys()

	horizonSourceAccount, err := client.LoadAccount(keys[0].Address)
	dieIfError("loadaccount", err)
	sourceAccount := mapAccounts(horizonSourceAccount)

	manageData := txnbuild.ManageData{
		Name:  "Fruit preference",
		Value: []byte("Apple"),
	}

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		Operations:    []txnbuild.Operation{&manageData},
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64 := buildSignEncode(tx, keys[0].Keypair)
	log.Println("Base 64 TX: ", txeBase64)

	return submit(client, txeBase64, mock)
}

func exampleAccountMerge(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	keys := initKeys()

	horizonSourceAccount, err := client.LoadAccount(keys[0].Address)
	dieIfError("loadaccount", err)
	sourceAccount := mapAccounts(horizonSourceAccount)

	accountMerge := txnbuild.AccountMerge{
		Destination: keys[1].Address,
	}

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		Operations:    []txnbuild.Operation{&accountMerge},
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64 := buildSignEncode(tx, keys[0].Keypair)
	log.Println("Base 64 TX: ", txeBase64)

	return submit(client, txeBase64, mock)
}

func exampleBumpSequence(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	keys := initKeys()

	horizonSourceAccount, err := client.LoadAccount(keys[1].Address)
	dieIfError("loadaccount", err)
	sourceAccount := mapAccounts(horizonSourceAccount)

	bumpSequence := txnbuild.BumpSequence{
		// BumpTo: 41137196761087,
		BumpTo: -1,
	}

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		Operations:    []txnbuild.Operation{&bumpSequence},
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64 := buildSignEncode(tx, keys[1].Keypair)
	log.Println("Base 64 TX: ", txeBase64)

	return submit(client, txeBase64, mock)
}

func exampleSendNonNative(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	keys := initKeys()
	horizonSourceAccount, err := client.LoadAccount(keys[0].Address)
	dieIfError("loadaccount", err)
	sourceAccount := mapAccounts(horizonSourceAccount)

	payment := txnbuild.Payment{
		Destination: keys[1].Address,
		Amount:      "100",
		Asset: txnbuild.CreditAsset{
			Code:   "ABCD",
			Issuer: keys[0].Address,
		},
	}

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		Operations:    []txnbuild.Operation{&payment},
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64 := buildSignEncode(tx, keys[0].Keypair)
	log.Println("Base 64 TX: ", txeBase64)

	return submit(client, txeBase64, mock)
}

func exampleSendLumens(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	keys := initKeys()
	horizonSourceAccount, err := client.LoadAccount(keys[0].Address)
	dieIfError("loadaccount", err)
	sourceAccount := mapAccounts(horizonSourceAccount)

	payment := txnbuild.Payment{
		Destination: keys[1].Address,
		Amount:      "100",
	}

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		Operations:    []txnbuild.Operation{&payment},
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64 := buildSignEncode(tx, keys[0].Keypair)
	log.Println("Base 64 TX: ", txeBase64)

	return submit(client, txeBase64, mock)
}

func exampleCreateAccount(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	keys := initKeys()
	horizonSourceAccount, err := client.LoadAccount(keys[0].Address)
	dieIfError("loadaccount", err)
	sourceAccount := mapAccounts(horizonSourceAccount)

	// newAccountKeypair := createKeypair()
	createAccount := txnbuild.CreateAccount{
		Destination: "GAS4V4O2B7DW5T7IQRPEEVCRXMDZESKISR7DVIGKZQYYV3OSQ5SH5LVP",
		Amount:      "10",
	}

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		Operations:    []txnbuild.Operation{&createAccount},
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64 := buildSignEncode(tx, keys[0].Keypair)
	log.Println("Base 64 TX: ", txeBase64)

	return submit(client, txeBase64, mock)
}

func submit(client *horizon.Client, txeBase64 string, mock bool) (resp horizon.TransactionSuccess) {
	if mock == true {
		resp = mockSuccess()
	} else {
		var err error
		resp, err = client.SubmitTransaction(txeBase64)
		if err != nil {
			bad := err.(*horizon.Error)
			PrintHorizonError(bad)
			os.Exit(1)
		}
	}

	return
}

func buildSignEncode(tx txnbuild.Transaction, keypair *keypair.Full) string {
	var err error
	err = tx.Build()
	dieIfError("build", err)

	err = tx.Sign(keypair)
	dieIfError("sign", err)

	txeBase64, err := tx.Base64()
	dieIfError("base64", err)

	return txeBase64
}

func dieIfError(desc string, err error) {
	if err != nil {
		log.Fatalf("Fatal error (%s): %s", desc, err)
	}
}

func mockSuccess() horizon.TransactionSuccess {
	return horizon.TransactionSuccess{}
}

func verify(received string) {
	expected := "AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAiII0AAAAWAAAAAAAAAAAAAAABAAAAAAAAAAkAAAAAAAAAAeoucsUAAABAOT7JB5aEckZsFYz4s0yh7IXoq09LqlAqw5HSgO83fk75NTYRiGt+gDebUiO1TUw/6HxZegJTZDu1Rw55m7uYCA=="

	if received != expected {
		log.Println("Assert failed!")
		log.Println("Expected: ", expected)
		log.Fatal("Received: ", received)
	}
}

// createKeypair constructs a new keypair
func createKeypair() *keypair.Full {
	pair, err := keypair.Random()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Seed:", pair.Seed())
	log.Println("Address:", pair.Address())

	return pair
}

func mapAccounts(horizonAccount horizon.Account) *horizon.Account {
	return &horizonAccount
}

// PrintHorizonError decodes and prints the contents of horizon.Error.Problem.
// Decoded XDR can be pasted into the Stellar Laboratory XDR viewer
// (https://www.stellar.org/laboratory) for further analysis.
// TODO: Move this to new client
func PrintHorizonError(hError *horizon.Error) error {
	problem := hError.Problem
	log.Println("Error type:", problem.Type)
	log.Println("Error title:", problem.Title)
	log.Println("Error status:", problem.Status)
	log.Println("Error detail:", problem.Detail)
	log.Println("Error instance:", problem.Instance)

	var decodedResultCodes map[string]interface{}
	var decodedResult, decodedEnvelope string
	var err error

	err = json.Unmarshal(problem.Extras["result_codes"], &decodedResultCodes)
	if err != nil {
		return errors.Wrap(err, "couldn't unmarshal result_codes")
	}
	log.Println("Error extras result codes:", decodedResultCodes)

	err = json.Unmarshal(problem.Extras["result_xdr"], &decodedResult)
	if err != nil {
		return errors.Wrap(err, "couldn't unmarshal result_xdr")
	}
	log.Println("Error extras result (TransactionResult) XDR:", decodedResult)

	err = json.Unmarshal(problem.Extras["envelope_xdr"], &decodedEnvelope)
	if err != nil {
		return errors.Wrap(err, "couldn't unmarshal envelope_xdr")
	}
	log.Println("Error extras envelope (TransactionEnvelope) XDR:", decodedEnvelope)

	return nil
}
