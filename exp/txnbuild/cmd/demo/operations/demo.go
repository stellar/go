// Package demo is an interactive demonstration of the Go SDK using the Stellar TestNet.
package demo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/stellar/go/clients/horizon"
	horizonclient "github.com/stellar/go/exp/clients/horizon"
	"github.com/stellar/go/exp/txnbuild"
	"github.com/stellar/go/network"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"

	"github.com/stellar/go/keypair"
)

// TODO:
// 2) Clean up printing output
// 3) Add missing operations

const friendbotAddress = "GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR"
const accountsFile = "demo.keys"

// Account represents a Stellar account for this demo.
type Account struct {
	Seed     string           `json:"name"`
	Address  string           `json:"address"`
	HAccount *horizon.Account `json:"account"`
	Keypair  *keypair.Full    `json:"keypair"`
	Exists   bool             `json:"exists"`
}

// InitKeys creates n random new keypairs, storing them in a local file. If the file exists,
// InitKeys reads the file instead to construct the keypairs (and ignores n).
func InitKeys(n int) []Account {
	var accounts []Account
	// accounts := make([]Account, n)
	fh, err := os.Open(accountsFile)

	if os.IsNotExist(err) {
		// Create the accounts and record them in a file
		log.Println("Accounts file not found...")
		for i := 0; i < n; i++ {
			accounts[i] = createKeypair()
		}

		jsonAccounts, err := json.MarshalIndent(accounts, "", "  ")
		dieIfError("problem marshalling json accounts", err)
		err = ioutil.WriteFile(accountsFile, jsonAccounts, 0644)
		dieIfError("problem writing json accounts file", err)

		return accounts
	}

	// Read the file and create keypairs
	log.Printf("Found accounts file %s...\n", accountsFile)
	defer fh.Close()
	bytes, err := ioutil.ReadAll(fh)
	dieIfError("problem converting json file to bytes", err)
	json.Unmarshal(bytes, &accounts)

	for i, k := range accounts {
		kp, err := keypair.Parse(k.Seed)
		dieIfError("keypair didn't parse!", err)
		accounts[i].Keypair = kp.(*keypair.Full)
	}

	return accounts
}

// Reset removes all test accounts created by this demo. All funds are transferred back to Friendbot.
func Reset(client *horizonclient.Client, keys []Account) {
	keys = loadAccounts(client, keys)
	for _, k := range keys {
		if !k.Exists {
			fmt.Printf("    Account %s not found - skipping further operations on it...\n", k.Address)
			continue
		}

		// It exists - so we will proceed to delete it
		fmt.Println("\n    Found testnet account with ID:", k.HAccount.ID)

		// Find any offers that need deleting...
		offerRequest := horizonclient.OfferRequest{
			ForAccount: k.Address,
			Cursor:     "now",
			Order:      horizonclient.OrderDesc,
		}
		offers, err := client.Offers(offerRequest)
		dieIfError("error while getting offers", err)
		fmt.Printf("    Account %s has %v offers:\n", k.Address, len(offers.Embedded.Records))

		// ...and delete them
		for _, o := range offers.Embedded.Records {
			fmt.Println("    ", o)
			txe, err := deleteOffer(k.HAccount, o.ID, k)
			dieIfError("problem building deleteOffer op", err)
			fmt.Printf("        Deleting offer %d...\n", o.ID)
			resp := submit(client, txe)
			fmt.Println(resp.TransactionSuccessToString())
		}

		// Find any authorised trustlines on this account...
		fmt.Printf("    Account %s has %d balances...\n", k.Address, len(k.HAccount.Balances))

		// ...and delete them
		for _, b := range k.HAccount.Balances {
			// Native balances don't have trustlines
			if b.Type == "native" {
				continue
			}
			asset := txnbuild.CreditAsset{
				Code:   b.Code,
				Issuer: b.Issuer,
			}

			// Send the asset back to the issuer...
			fmt.Printf("        Sending %v of surplus asset %s:%s back to issuer...\n", b.Balance, asset.Code, asset.Issuer)
			txe, err := payment(k.HAccount, asset.Issuer, b.Balance, asset, k)
			dieIfError("problem building payment op", err)
			resp := submit(client, txe)
			fmt.Println(resp.TransactionSuccessToString())

			// Delete the now-empty trustline...
			fmt.Printf("        Deleting trustline for asset %s:%s...\n", b.Code, b.Issuer)
			txe, err = deleteTrustline(k.HAccount, asset, k)
			dieIfError("problem building deleteTrustline op", err)
			resp = submit(client, txe)
			fmt.Println(resp.TransactionSuccessToString())
		}

		// Find any data entries on this account...
		fmt.Printf("    Account %s has %d data entries...\n", k.Address, len(k.HAccount.Data))
		for dataKey := range k.HAccount.Data {
			decodedV, _ := k.HAccount.GetData(dataKey)
			fmt.Printf("    Deleting data entry '%s' -> '%s'...\n", dataKey, decodedV)
			txe, err := deleteData(k.HAccount, dataKey, k)
			dieIfError("problem building manageData op", err)
			resp := submit(client, txe)
			fmt.Println(resp.TransactionSuccessToString())
		}
	}

	// Merge the accounts...
	for _, k := range keys {
		if !k.Exists {
			continue
		}
		fmt.Printf("    Merging account %s back to friendbot (%s)...\n", k.Address, friendbotAddress)
		txe, err := mergeAccount(k.HAccount, friendbotAddress, k)
		dieIfError("problem building mergeAccount op", err)
		resp := submit(client, txe)
		fmt.Println(resp.TransactionSuccessToString())
	}
}

// Initialise funds an initial set of accounts for use with other demo operations. The first account is
// funded from Friendbot; subseqeuent accounts are created and funded from this first account.
func Initialise(client *horizonclient.Client, keys []Account) {
	// Fund the first account from friendbot
	fmt.Printf("    Funding account %s from friendbot...\n", keys[0].Address)
	_, err := client.Fund(keys[0].Address)
	dieIfError(fmt.Sprintf("couldn't fund account %s from friendbot", keys[0].Address), err)

	keys[0].HAccount = loadAccount(client, keys[0].Address)
	keys[0].Exists = true

	// Fund the others using the create account operation
	for i := 1; i < len(keys); i++ {
		fmt.Printf("    Funding account %s from account %s...\n", keys[i].Address, keys[0].Address)
		txe, err := createAccount(keys[0].HAccount, keys[i].Address, keys[0])
		dieIfError("problem building createAccount op", err)
		resp := submit(client, txe)
		fmt.Println(resp.TransactionSuccessToString())
	}
}

// TXError deliberately creates a bad transaction to trigger an error response from Horizon. This code
// demonstrates how to retrieve and inspect the error.
func TXError(client *horizonclient.Client, keys []Account) {
	keys = loadAccounts(client, keys)
	// Create a bump seq operation
	// Set the seq number to 1 (invalid)
	// Create the transaction
	txe, err := bumpSequence(keys[0].HAccount, -1, keys[0])
	dieIfError("problem building createAccount op", err)
	resp := submit(client, txe)
	// Submit
	// Inspect and print error
	fmt.Println(resp.TransactionSuccessToString())
}

func bumpSequence(source *horizon.Account, seqNum int64, signer Account) (string, error) {
	bumpSequenceOp := txnbuild.BumpSequence{
		BumpTo: seqNum,
	}

	tx := txnbuild.Transaction{
		SourceAccount: source,
		Operations:    []txnbuild.Operation{&bumpSequenceOp},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64, err := tx.BuildSignEncode(signer.Keypair)
	if err != nil {
		return "", errors.Wrap(err, "couldn't serialise transaction")
	}

	return txeBase64, nil
}

func createAccount(source *horizon.Account, dest string, signer Account) (string, error) {
	createAccountOp := txnbuild.CreateAccount{
		Destination: dest,
		Amount:      "100",
	}

	tx := txnbuild.Transaction{
		SourceAccount: source,
		Operations:    []txnbuild.Operation{&createAccountOp},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64, err := tx.BuildSignEncode(signer.Keypair)
	if err != nil {
		return "", errors.Wrap(err, "couldn't serialise transaction")
	}

	return txeBase64, nil
}

func deleteData(source *horizon.Account, dataKey string, signer Account) (string, error) {
	manageDataOp := txnbuild.ManageData{
		Name: dataKey,
	}

	tx := txnbuild.Transaction{
		SourceAccount: source,
		Operations:    []txnbuild.Operation{&manageDataOp},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64, err := tx.BuildSignEncode(signer.Keypair)
	if err != nil {
		return "", errors.Wrap(err, "couldn't serialise transaction")
	}

	return txeBase64, nil
}

func manageData(source *horizon.Account, dataKey string, dataValue string, signer Account) (string, error) {
	manageDataOp := txnbuild.ManageData{
		Name:  dataKey,
		Value: []byte(dataValue),
	}

	tx := txnbuild.Transaction{
		SourceAccount: source,
		Operations:    []txnbuild.Operation{&manageDataOp},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64, err := tx.BuildSignEncode(signer.Keypair)
	if err != nil {
		return "", errors.Wrap(err, "couldn't serialise transaction")
	}

	return txeBase64, nil
}

func payment(source *horizon.Account, dest, amount string, asset txnbuild.Asset, signer Account) (string, error) {
	paymentOp := txnbuild.Payment{
		Destination: dest,
		Amount:      amount,
		Asset:       asset,
	}

	tx := txnbuild.Transaction{
		SourceAccount: source,
		Operations:    []txnbuild.Operation{&paymentOp},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64, err := tx.BuildSignEncode(signer.Keypair)
	if err != nil {
		return "", errors.Wrap(err, "couldn't serialise transaction")
	}

	return txeBase64, nil
}

func deleteTrustline(source *horizon.Account, asset txnbuild.Asset, signer Account) (string, error) {
	deleteTrustline := txnbuild.RemoveTrustlineOp(asset)

	tx := txnbuild.Transaction{
		SourceAccount: source,
		Operations:    []txnbuild.Operation{&deleteTrustline},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64, err := tx.BuildSignEncode(signer.Keypair)
	if err != nil {
		return "", errors.Wrap(err, "couldn't serialise transaction")
	}

	return txeBase64, nil
}

func deleteOffer(source *horizon.Account, offerID int64, signer Account) (string, error) {
	deleteOffer := txnbuild.DeleteOfferOp(offerID)

	tx := txnbuild.Transaction{
		SourceAccount: source,
		Operations:    []txnbuild.Operation{&deleteOffer},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64, err := tx.BuildSignEncode(signer.Keypair)
	if err != nil {
		return "", errors.Wrap(err, "couldn't serialise transaction")
	}

	return txeBase64, nil
}

func mergeAccount(source *horizon.Account, destAddress string, signer Account) (string, error) {
	accountMerge := txnbuild.AccountMerge{
		Destination: destAddress,
	}

	tx := txnbuild.Transaction{
		SourceAccount: source,
		Operations:    []txnbuild.Operation{&accountMerge},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64, err := tx.BuildSignEncode(signer.Keypair)
	if err != nil {
		return "", errors.Wrap(err, "couldn't serialise transaction")
	}

	return txeBase64, nil
}

// createKeypair constructs a new random keypair, and returns it in a DemoAccount.
func createKeypair() Account {
	pair, err := keypair.Random()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Seed:", pair.Seed())
	log.Println("Address:", pair.Address())

	return Account{
		Seed:    pair.Seed(),
		Address: pair.Address(),
		Keypair: pair,
	}
}

func loadAccounts(client *horizonclient.Client, keys []Account) []Account {
	for i, k := range keys {
		keys[i].HAccount = loadAccount(client, k.Address)
		keys[i].Exists = true
	}

	return keys
}

func loadAccount(client *horizonclient.Client, address string) *horizon.Account {
	accountRequest := horizonclient.AccountRequest{AccountID: address}
	horizonSourceAccount, err := client.AccountDetail(accountRequest)
	if err != nil {
		dieIfError(fmt.Sprintf("couldn't get account detail for %s", address), err)
	}

	return &horizonSourceAccount
}

func submit(client *horizonclient.Client, txeBase64 string) (resp horizon.TransactionSuccess) {
	resp, err := client.SubmitTransactionXDR(txeBase64)
	if err != nil {
		hError := err.(*horizonclient.Error)
		err = printHorizonError(hError)
		dieIfError("couldn't print Horizon eror", err)
		os.Exit(1)
	}

	return
}

func dieIfError(desc string, err error) {
	if err != nil {
		log.Fatalf("Fatal error (%s): %s", desc, err)
	}
}

func printHorizonError(hError *horizonclient.Error) error {
	problem := hError.Problem
	log.Println("Error type:", problem.Type)
	log.Println("Error title:", problem.Title)
	log.Println("Error status:", problem.Status)
	log.Println("Error detail:", problem.Detail)
	log.Println("Error instance:", problem.Instance)

	resultCodes, err := hError.ResultCodes()
	if err != nil {
		return errors.Wrap(err, "Couldn't read ResultCodes")
	}
	log.Println("TransactionCode:", resultCodes.TransactionCode)
	log.Println("OperationCodes:")
	for _, code := range resultCodes.OperationCodes {
		log.Println("    ", code)
	}

	resultString, err := hError.ResultString()
	if err != nil {
		return errors.Wrap(err, "Couldn't read ResultString")
	}
	log.Println("TransactionResult XDR (base 64):", resultString)

	envelope, err := hError.Envelope()
	if err != nil {
		return errors.Wrap(err, "Couldn't read Envelope")
	}

	// envelopeXDR, err := hError.EnvelopeXDR()
	// if err != nil {
	// 	return errors.Wrap(err, "Couldn't read Envelope XDR")
	// }

	log.Println("TransactionEnvelope XDR:", envelope)
	// log.Println("TransactionEnvelope XDR 2:", envelopeXDR)

	log.Println("***************")
	txe := envelope.Tx
	log.Println("Transaction:", txe)
	aid := txe.SourceAccount.MustEd25519()
	decodedAID, err := strkey.Encode(strkey.VersionByteAccountID, aid[:])
	if err != nil {
		log.Println("Couldn't decode account ID:", err)
	} else {
		log.Printf("SourceAccount (%s): %s\n", txe.SourceAccount.Type, decodedAID)
	}
	log.Println("Fee:", txe.Fee)
	log.Println("SequenceNumber:", txe.SeqNum)
	log.Println("TimeBounds:", txe.TimeBounds)
	log.Println("Memo:", txe.Memo)
	log.Println("Memo.Type:", txe.Memo.Type)
	if txe.Memo.Type != xdr.MemoTypeMemoNone {
		log.Println("Memo.Text:", txe.Memo.Text)
		log.Println("Memo.Id:", txe.Memo.Id)
		log.Println("Memo.Hash:", txe.Memo.Hash)
		log.Println("Memo.RetHash:", txe.Memo.RetHash)
	}
	log.Println("Operations:", txe.Operations)

	// Generalise printing for any operation type
	for _, op := range txe.Operations {
		log.Println("Operations.SourceAccount:", op.SourceAccount)
		log.Println("Operations.Body.Type:", op.Body.Type)
		// log.Println("Operations.Body.BumpSequenceOp.BumpTo:", op.Body.BumpSequenceOp.BumpTo)
	}
	log.Println("Ext:", txe.Ext)

	// SourceAccount AccountId
	// Fee           Uint32
	// SeqNum        SequenceNumber
	// TimeBounds    *TimeBounds
	// Memo          Memo
	// type Memo struct {
	// 	Type    MemoType
	// 	Text    *string `xdrmaxsize:"28"`
	// 	Id      *Uint64
	// 	Hash    *Hash
	// 	RetHash *Hash
	// Operations    []Operation `xdrmaxsize:"100"`
	// Ext           TransactionExt

	return nil
}
