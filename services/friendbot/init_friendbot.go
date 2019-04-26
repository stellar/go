package main

import (
	"fmt"
	"net/http"

	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/friendbot/internal"
	"github.com/stellar/go/strkey"
)

func initFriendbot(
	friendbotSecret string,
	networkPassphrase string,
	horizonURL string,
	startingBalance string,
	numMinions uint16,
) (*internal.Bot, error) {
	if friendbotSecret == "" || networkPassphrase == "" || horizonURL == "" || startingBalance == "" {
		// XXX: Check error formatting syntax
		return nil, fmt.Errorf("invalid input param")
	}

	// ensure its a seed if its not blank
	strkey.MustDecode(strkey.VersionByteSeed, friendbotSecret)

	// XXX: Change type
	hclient := &horizon.Client{
		URL:     horizonURL,
		HTTP:    http.DefaultClient,
		AppName: "friendbot",
	}

	var minions []internal.Minion
	// XXX: Looping logic discussed with Nikhil
	for i := 0; i < numMinions; i++ {
		minionSecret, err := createNewAccount(friendbotSecret, networkPassphrase, hclient)
		if err != nil {
			return nil, fmt.Errorf("creating minion account", err)
		}
		minions = append(minions, internal.Minion{
			Secret:       minionSecret,
			DestAddrChan: make(chan string),
			TxResultChan: make(chan internal.TxResult),
		})
	}

	return &internal.Bot{
		Secret:            friendbotSecret,
		Horizon:           hclient,
		Network:           networkPassphrase,
		StartingBalance:   startingBalance,
		SubmitTransaction: internal.AsyncSubmitTransaction,
		Minions:           minions,
	}, nil
}

// XXX: Change horizon Client type
// XXX: Rewrite to return a list and take in a number of secret keys
func createNewAccount(sourceSeed, networkPassphrase string, hclient *horizon.Client) (string, error) {
	kp, err := keypair.Random()
	if err != nil {
		return "", fmt.Errorf("creating keypair", err)
	}
	destAddr := kp.Address()
	// XXX: Create tx without CreateAccount
	tx, err := build.Transaction(
		build.SourceAccount{AddressOrSeed: sourceSeed},
		build.Network{Passphrase: networkPassphrase},
		build.AutoSequence{SequenceProvider: hclient},
		// XXX: See Nikhil PR comment
		build.CreateAccount(
			build.Destination{AddressOrSeed: destAddr},
			// XXX: Add StartingBalance here
		),
	)
	// XXX: Loop over tx and add CreateAccount ops

	txe, err := tx.Sign(sourceSeed)
	if err != nil {
		return "", fmt.Errorf("signing tx", err)
	}

	txeB64, err := txe.Base64()
	if err != nil {
		return "", fmt.Errorf("converting to base 64", err)
	}
	_, err = hclient.SubmitTransaction(txeB64)
	if err != nil {
		return "", fmt.Errorf("submitting tx", err)
	}
	return destKeypair.Seed(), nil
}
