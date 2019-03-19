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
	resp := exampleAllowTrust(client, false)
	fmt.Println(resp.TransactionSuccessToString())
}

func exampleAllowTrust(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	keys := initKeys()
	horizonSourceAccount, err := client.LoadAccount(keys[0].Address)
	dieIfError("loadaccount", err)
	sourceAccount := mapAccounts(horizonSourceAccount)

	issued := txnbuild.NewAsset("ABCD", keys[0].Address)
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

	issuer := txnbuild.NewAsset("ABCD", keys[0].Address)
	changeTrust := txnbuild.ChangeTrust{
		Line:  issuer,
		Limit: "10",
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

	issuedAsset := txnbuild.NewAsset("ABCD", keys[1].Address)
	removeTrust := txnbuild.NewRemoveTrustlineOp(issuedAsset)

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
		Asset:       txnbuild.NewAsset("ABCD", keys[0].Address),
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
		Asset:       "native",
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

func mapAccounts(horizonAccount horizon.Account) txnbuild.Account {
	myAccount := txnbuild.Account{
		ID: horizonAccount.ID,
	}
	var err error
	myAccount.SequenceNumber, err = horizonAccount.GetSequenceNumber()
	dieIfError("GetSequenceNumber", err)

	return myAccount
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
		return errors.Wrap(err, "Couldn't unmarshal result_codes")
	}
	log.Println("Error extras result codes:", decodedResultCodes)

	err = json.Unmarshal(problem.Extras["result_xdr"], &decodedResult)
	if err != nil {
		return errors.Wrap(err, "Couldn't unmarshal result_xdr")
	}
	log.Println("Error extras result (TransactionResult) XDR:", decodedResult)

	err = json.Unmarshal(problem.Extras["envelope_xdr"], &decodedEnvelope)
	if err != nil {
		return errors.Wrap(err, "Couldn't unmarshal envelope_xdr")
	}
	log.Println("Error extras envelope (TransactionEnvelope) XDR:", decodedEnvelope)

	return nil
}
