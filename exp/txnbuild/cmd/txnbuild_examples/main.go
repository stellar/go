package main

// This is a scratch pad for testing new operations. Please DO NOT review!

import (
	"log"
	"os"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/exp/txnbuild"
	"github.com/stellar/go/keypair"
)

type key struct {
	Seed    string
	Address string
	Keypair keypair.KP
}

func initKeys() []key {
	// Accounts created on testnet
	keys := []key{
		// test1
		key{Seed: "SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R",
			Address: "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3",
		},
		// test2
		key{Seed: "SBMSVD4KKELKGZXHBUQTIROWUAPQASDX7KEJITARP4VMZ6KLUHOGPTYW",
			Address: "GAS4V4O2B7DW5T7IQRPEEVCRXMDZESKISR7DVIGKZQYYV3OSQ5SH5LVP",
		},
		// test3
		key{Seed: "SBZVMB74Z76QZ3ZOY7UTDFYKMEGKW5XFJEB6PFKBF4UYSSWHG4EDH7PY",
			Address: "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H"},
	}

	for i, k := range keys {
		myKeypair, err := keypair.Parse(k.Seed)
		dieIfError("keypair didn't parse!", err)
		keys[i].Keypair = myKeypair
	}

	return keys
}

func main() {
	client := horizon.DefaultTestNetClient
	txnbuild.UseTestNetwork()

	// resp := exampleCreateAccount(client)
	// resp := exampleSendLumens(client, true)
	resp := exampleBumpSequence(client, true)
	txnbuild.PrintTransactionSuccess(resp)
}

func exampleBumpSequence(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	keys := initKeys()
	sourceAccount, err := client.LoadAccount(keys[1].Address)
	dieIfError("loadaccount", err)

	bumpSequence := txnbuild.BumpSequence{
		BumpTo: 9606132444168300,
	}

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		Operations:    []txnbuild.Operation{&bumpSequence},
	}

	txeBase64 := buildSignEncode(tx, keys[1].Seed)
	log.Println("Base 64 TX: ", txeBase64)

	var resp horizon.TransactionSuccess
	if mock == true {
		resp = mockSuccess()
	} else {
		resp, err = client.SubmitTransaction(txeBase64)
		if err != nil {
			bad := err.(*horizon.Error)
			txnbuild.PrintHorizonError(bad)
			os.Exit(1)
		}
	}

	return resp
}

func exampleSendLumens(client *horizon.Client, mock bool) horizon.TransactionSuccess {
	keys := initKeys()
	sourceAccount, err := client.LoadAccount(keys[0].Address)
	dieIfError("loadaccount", err)

	payment := txnbuild.Payment{
		Destination: keys[2].Address,
		Amount:      "10",
	}

	tx := txnbuild.Transaction{
		SourceAccount: sourceAccount,
		Operations:    []txnbuild.Operation{&payment},
	}

	txeBase64 := buildSignEncode(tx, keys[0].Seed)
	log.Println("Base 64 TX: ", txeBase64)

	var resp horizon.TransactionSuccess
	if mock == true {
		resp = mockSuccess()
	} else {
		resp, err = client.SubmitTransaction(txeBase64)
		if err != nil {
			bad := err.(*horizon.Error)
			txnbuild.PrintHorizonError(bad)
			os.Exit(1)
		}
	}

	return resp
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

	txeBase64 := buildSignEncode(tx, secretSeed)
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

func buildSignEncode(tx txnbuild.Transaction, secretSeed string) string {
	var err error
	err = tx.Build()
	dieIfError("build", err)

	err = tx.Sign(secretSeed)
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
