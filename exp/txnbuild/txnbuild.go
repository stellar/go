package txnbuild

import (
	"log"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/network"
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
}

// Build ...
func (tx *Transaction) Build() (Transaction, error) {
	return *tx, nil
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

	txe, err := tx.Build()
	checkError(err)

	txe, err = txe.Sign(secretSeed)
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
