package main

// This is a file of useful experiments with the original Go SDK (build lib), for reference
// as the new SDK is implemented. Please DO NOT review!

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
)

type key struct {
	Seed    string
	Address string
	Keypair keypair.KP
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

// createFriendbotAcct creates an account on the testnet using friendbot
func createFriendbotAcct() {
	pair := createKeypair()
	address := pair.Address()
	resp, err := http.Get("https://friendbot.stellar.org/?addr=" + address)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	printHorizonResponse(resp)
	printAcctBalance(address)
}

// getAcct uses a keypair to lookup an account
func getAcct(address string) (account horizon.Account) {
	account, err := horizon.DefaultTestNetClient.LoadAccount(address)
	if err != nil {
		log.Fatal(err)
	}

	return account
}

func printHorizonResponse(resp *http.Response) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Response from Horizon:")
	fmt.Println(string(body))
}

func printAcctBalance(address string) {
	account := getAcct(address)
	log.Println("Balances for account:", address)

	for _, balance := range account.Balances {
		log.Println(balance)
	}
}

func checkError(err error) {
	if err != nil {
		log.Fatal("Fatal error:", err)
	}
}

func createNewAccount(sourceSeed string) (horizon.TransactionSuccess, *keypair.Full, error) {
	// Nominate a new random account keypair
	destinationKeypair := createKeypair()
	destination := destinationKeypair.Address()

	// Build the transaction
	tx, err := build.Transaction(
		build.SourceAccount{AddressOrSeed: sourceSeed},
		// build.Sequence{1},
		build.TestNetwork,
		build.AutoSequence{SequenceProvider: horizon.DefaultTestNetClient},
		build.CreateAccount(
			build.Destination{AddressOrSeed: destination},
			build.NativeAmount{Amount: "10"},
		),
	)

	// Sign
	txe, err := tx.Sign(sourceSeed)
	checkError(err)

	// Encode
	txeB64, err := txe.Base64()
	checkError(err)
	log.Printf("TXE base64: %s\n", txeB64)

	// Submit
	resp, err := horizon.DefaultTestNetClient.SubmitTransaction(txeB64)

	return resp, destinationKeypair, err
}

func sendLumens(sourceSeed, sourceAddress, destination, amount string) (horizon.TransactionSuccess, error) {
	// Make sure destination account exists
	if _, err := horizon.DefaultTestNetClient.LoadAccount(destination); err != nil {
		log.Fatal("Fatal error:", err)
	}

	// passphrase := network.TestNetworkPassphrase

	tx, err := build.Transaction(
		build.TestNetwork,
		build.SourceAccount{AddressOrSeed: sourceAddress}, // No need to pass secret seed here!
		build.AutoSequence{SequenceProvider: horizon.DefaultTestNetClient},
		build.Payment(
			build.Destination{AddressOrSeed: destination},
			build.NativeAmount{Amount: amount},
		),
	)
	checkError(err)

	txe, err := tx.Sign(sourceSeed)
	checkError(err)

	txeB64, err := txe.Base64()
	checkError(err)

	resp, err := horizon.DefaultTestNetClient.SubmitTransaction(txeB64)
	log.Println("Horizon.Problem: ", horizon.Problem{})

	return resp, err
}

func main() {
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
		checkError(err)
		keys[i].Keypair = myKeypair
	}

	// log.Fatal("Keypair seed:", keys[0].Keypair.(*keypair.Full).Seed())

	sourceSeed := keys[0].Seed
	sourceAddress := keys[0].Address
	// destinationSeed := keys[1].Seed
	destinationAddress := keys[1].Address

	// resp, destinationKeypair, err := createNewAccount(sourceSeed)
	// resp, err := sendLumens(sourceSeed, destinationAddress, "10")
	resp, err := sendLumens(sourceSeed, sourceAddress, destinationAddress, "10")

	// Check how we did
	checkError(err)
	fmt.Println(resp.TransactionSuccessToString())

	// Print final balances
	printAcctBalance(sourceAddress)
	// printAcctBalance(destinationKeypair.Address())
	printAcctBalance(destinationAddress)

}
