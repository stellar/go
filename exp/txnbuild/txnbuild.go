package txnbuild

import (
	"log"
	"strconv"

	"github.com/stellar/go/clients/horizon"
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
}

// Build ...
func (tx *Transaction) Build() error {
	// Skipped: Create TX struct

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

	// Run through operations sequentially
	// Create operation body

	// TODO: Generalise, remove hard-coded inflation type
	body, err := xdr.NewOperationBody(xdr.OperationTypeInflation, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to create XDR")
	}
	// Append relevant operation to TX.operations
	operation := xdr.Operation{Body: body}
	tx.TX.Operations = append(tx.TX.Operations, operation)

	return nil
}

// Sign ...
func (tx *Transaction) Sign(seed string) (Transaction, error) {
	return *tx, nil
}

// Base64 ...
func (tx *Transaction) Base64() (string, error) {
	return "", nil
}

// ExampleCreateAccount ...
func ExampleCreateAccount(client *horizon.Client) (horizon.TransactionSuccess, error) {
	secretSeed := "S..."
	sourceAddress := "G..."
	sourceAccount, err := client.LoadAccount(sourceAddress)
	checkError(err)

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
	checkError(err)

	txe, err := tx.Sign(secretSeed)
	checkError(err)

	txeBase64, err := txe.Base64()
	checkError(err)

	resp, err := client.SubmitTransaction(txeBase64)

	return resp, err
}

func main() {
	client := horizon.DefaultTestNetClient

	resp, err := ExampleCreateAccount(client)
	checkError(err)

	printTransactionSuccess(resp)
}

func checkError(err error) {
	if err != nil {
		log.Fatal("Fatal error:", err)
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

func seqNumFromAccount(account horizon.Account) (xdr.SequenceNumber, error) {
	seqNum, err := strconv.ParseUint(account.Sequence, 10, 64)

	if err != nil {
		return 0, errors.Wrap(err, "Failed to parse account sequence number")
	}

	return xdr.SequenceNumber(seqNum), nil
}
