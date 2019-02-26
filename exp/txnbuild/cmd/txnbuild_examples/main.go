package main

// This is a scratch pad for testing new operations. Please DO NOT review!

import (
	"log"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/exp/txnbuild"
	"github.com/stellar/go/keypair"
)

func main() {
	client := horizon.DefaultTestNetClient
	txnbuild.UseTestNetwork()

	resp := exampleCreateAccount(client)
	txnbuild.PrintTransactionSuccess(resp)
}

func exampleCreateAccount(client *horizon.Client) horizon.TransactionSuccess {
	secretSeed := "SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R"
	sourceAddress := "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3"
	sourceAccount, err := client.LoadAccount(sourceAddress)
	dieIfError("loadaccount", err)

	// newAccountKeypair := createKeypair()
	createAccount := txnbuild.CreateAccount{
		// Destination: newAccountKeypair.Address(),
		Destination: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:      "10",
		Asset:       "native",
	}
	// inflation := txnbuild.Inflation{}

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		// Operations:    []txnbuild.Operation{&inflation},
		// Operations: []txnbuild.Operation{&inflation, &createAccount},
		Operations: []txnbuild.Operation{&createAccount},
	}

	err = tx.Build()
	dieIfError("build", err)

	err = tx.Sign(secretSeed)
	dieIfError("sign", err)

	txeBase64, err := tx.Base64()
	dieIfError("base64", err)

	log.Println("Base 64 TX: ", txeBase64)

	// TODO: Add client method to convert to base 64 internally.
	// resp, err := client.SubmitTransaction(txeBase64)
	// if err != nil {
	// 	bad := err.(*horizon.Error)
	// 	txnbuild.PrintHorizonError(bad)
	// 	os.Exit(1)
	// }

	// verify(txeBase64)
	resp := mockSuccess()

	return resp
}

func dieIfError(desc string, err error) {
	if err != nil {
		log.Fatalf("Fatal error (%s): %s", desc, err)
	}
}

func mockSuccess() horizon.TransactionSuccess {
	resp := horizon.TransactionSuccess{}

	return resp
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
