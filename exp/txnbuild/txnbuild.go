// package txnbuild
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"strconv"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// StellarNetwork ...
var StellarNetwork string

// UseTestNetwork ...
func UseTestNetwork() {
	StellarNetwork = network.TestNetworkPassphrase
}

// UsePublicNetwork ...
func UsePublicNetwork() {
	StellarNetwork = network.PublicNetworkPassphrase
}

// type Account struct {
// 	AccountID string
// 	Sequence  string
// }

// Operation ...
type Operation interface{}

// CreateAccount ...
type CreateAccount struct {
	Destination string
	Amount      string
	Asset       string
}

// Transaction ...
type Transaction struct {
	SourceAccount horizon.Account
	Operations    []Operation
	TX            *xdr.Transaction
	BaseFee       uint64
	Envelope      *xdr.TransactionEnvelope
}

// Hash ...
func (tx *Transaction) Hash() ([32]byte, error) {
	return network.HashTransaction(tx.TX, StellarNetwork)
}

// Bytes ...
func (tx *Transaction) Bytes() ([]byte, error) {
	var txBytes bytes.Buffer
	_, err := xdr.Marshal(&txBytes, tx.Envelope)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to marshal XDR")
	}

	return txBytes.Bytes(), nil
}

// Base64 ...
func (tx *Transaction) Base64() (string, error) {
	bs, err := tx.Bytes()
	if err != nil {
		return "", errors.Wrap(err, "Failed to get XDR bytestring")
	}

	return base64.StdEncoding.EncodeToString(bs), nil
}

// Build ...
func (tx *Transaction) Build() error {
	// Initialise TX (XDR) struct if needed
	if tx.TX == nil {
		tx.TX = &xdr.Transaction{}
	}

	// Skipped: Set network passphrase (in XDR?)

	// Set account ID in TX
	tx.TX.SourceAccount.SetAddress(tx.SourceAccount.ID)

	// Set sequence number in TX
	seqNum, err := seqNumFromAccount(tx.SourceAccount)
	if err != nil {
		return err
	}
	tx.TX.SeqNum = seqNum + 1

	// TODO: Loop through operations sequentially
	// Create operation body

	// TODO: Generalise, remove hard-coded inflation type
	body, err := xdr.NewOperationBody(xdr.OperationTypeInflation, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to create XDR")
	}
	// Append relevant operation to TX.operations
	operation := xdr.Operation{Body: body}
	tx.TX.Operations = append(tx.TX.Operations, operation)

	// Set defaults
	var DefaultBaseFee uint64 = 100
	if tx.BaseFee == 0 {
		tx.BaseFee = DefaultBaseFee
	}
	if tx.TX.Fee == 0 {
		tx.TX.Fee = xdr.Uint32(int(tx.BaseFee) * len(tx.TX.Operations))
	}

	return nil
}

// Sign ...
func (tx *Transaction) Sign(seed string) error {
	// Initialise transaction envelope
	if tx.Envelope == nil {
		tx.Envelope = &xdr.TransactionEnvelope{}
		tx.Envelope.Tx = *tx.TX
	}

	// TODO: Next steps:
	// 1) Untangle the connections between EnvelopeBuilder and TransactionBuilder
	// 2) Determine what needs to be new, what can be kept

	// Hash the transaction
	hash, err := tx.Hash()
	if err != nil {
		return errors.Wrap(err, "Failed to hash transaction")
	}

	// Sign the hash
	// TODO: Allow multiple signers
	kp, err := keypair.Parse(seed)
	if err != nil {
		return errors.Wrap(err, "Failed to parse seed")
	}

	sig, err := kp.SignDecorated(hash[:])
	if err != nil {
		return errors.Wrap(err, "Failed to sign transaction")
	}

	// Append the signature to the envelope
	tx.Envelope.Signatures = append(tx.Envelope.Signatures, sig)

	return nil
}

// ExampleCreateAccount ...
func ExampleCreateAccount(client *horizon.Client) (horizon.TransactionSuccess, error) {
	secretSeed := "SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R"
	sourceAddress := "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3"
	sourceAccount, err := client.LoadAccount(sourceAddress)
	checkError("loadaccount", err)

	createAccount := CreateAccount{
		Destination: "G...",
		Amount:      "10",
		Asset:       "native",
	}

	tx := Transaction{
		SourceAccount: sourceAccount,
		Operations:    []Operation{createAccount},
	}

	err = tx.Build()
	checkError("build", err)

	err = tx.Sign(secretSeed)
	checkError("sign", err)

	txeBase64, err := tx.Base64()
	checkError("base64", err)

	log.Println("Base 64 TX: ", txeBase64)

	resp, err := client.SubmitTransaction(txeBase64)
	if err != nil {
		bad := err.(*horizon.Error)
		printHorizonError(bad)
		os.Exit(1)
	}

	return resp, err
}

func main() {
	client := horizon.DefaultTestNetClient
	UseTestNetwork()

	resp, err := ExampleCreateAccount(client)
	printTransactionSuccess(resp)
	checkError("examplecreateaccount", err)

}

func checkError(desc string, err error) {
	if err != nil {
		log.Fatalf("Fatal error (%s): %s", desc, err)
	}
}

func printTransactionSuccess(resp horizon.TransactionSuccess) {
	log.Println("TransactionSuccess:")
	log.Println("")
	log.Println("Links:", resp.Links)
	log.Println("Hash:", resp.Hash)
	log.Println("Ledger:", resp.Ledger)
	log.Println("Env:", resp.Env)
	log.Println("Result:", resp.Result)
	log.Println("Meta:", resp.Meta)
	log.Println("")
}

func printHorizonError(hError *horizon.Error) {
	problem := hError.Problem
	log.Println("Error type:", problem.Type)
	log.Println("Error title:", problem.Title)
	log.Println("Error status:", problem.Status)
	log.Println("Error detail:", problem.Detail)
	log.Println("Error instance:", problem.Instance)

	var decodedResultCodes map[string]interface{}
	var decodedResult string
	var decodedEnvelope string
	var err error

	// err = json.Unmarshal(problem.Extras["envelope_xdr"], &decoded)
	// checkError("json unmarshal horizon.Error.Problem.Extras[\"envelope_xdr\"]", err)
	// log.Println("Error extras envelope XDR:", decoded)

	err = json.Unmarshal(problem.Extras["result_codes"], &decodedResultCodes)
	checkError("json unmarshal horizon.Error.Problem.Extras[\"result_codes\"]", err)
	log.Println("Error extras result codes:", decodedResultCodes)

	err = json.Unmarshal(problem.Extras["result_xdr"], &decodedResult)
	checkError("json unmarshal horizon.Error.Problem.Extras[\"result_xdr\"]", err)
	log.Println("Error extras result (TransactionResult) XDR:", decodedResult)

	err = json.Unmarshal(problem.Extras["envelope_xdr"], &decodedEnvelope)
	checkError("json unmarshal horizon.Error.Problem.Extras[\"envelope_xdr\"]", err)
	log.Println("Error extras envelope (TransactionEnvelope) XDR:", decodedEnvelope)
}

func seqNumFromAccount(account horizon.Account) (xdr.SequenceNumber, error) {
	seqNum, err := strconv.ParseUint(account.Sequence, 10, 64)

	if err != nil {
		return 0, errors.Wrap(err, "Failed to parse account sequence number")
	}

	return xdr.SequenceNumber(seqNum), nil
}
