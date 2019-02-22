package main

import (
	"log"
	"os"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/exp/txnbuild"
)

func main() {
	client := horizon.DefaultTestNetClient
	txnbuild.UseTestNetwork()

	resp, err := ExampleCreateAccount(client)
	txnbuild.PrintTransactionSuccess(resp)
	txnbuild.CheckError("examplecreateaccount", err)

}

// ExampleCreateAccount ...
func ExampleCreateAccount(client *horizon.Client) (horizon.TransactionSuccess, error) {
	secretSeed := "SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R"
	sourceAddress := "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3"
	sourceAccount, err := client.LoadAccount(sourceAddress)
	txnbuild.CheckError("loadaccount", err)

	createAccount := txnbuild.CreateAccount{
		Destination: "G...",
		Amount:      "10",
		Asset:       "native",
	}

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		Operations:    []txnbuild.Operation{createAccount},
	}

	err = tx.Build()
	txnbuild.CheckError("build", err)

	err = tx.Sign(secretSeed)
	txnbuild.CheckError("sign", err)

	txeBase64, err := tx.Base64()
	txnbuild.CheckError("base64", err)

	log.Println("Base 64 TX: ", txeBase64)

	resp, err := client.SubmitTransaction(txeBase64)
	if err != nil {
		bad := err.(*horizon.Error)
		txnbuild.PrintHorizonError(bad)
		os.Exit(1)
	}

	return resp, err
}
