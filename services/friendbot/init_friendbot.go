package main

import (
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
	numMinions int,
) *internal.Bot {
	if friendbotSecret == "" || networkPassphrase == "" || horizonURL == "" || startingBalance == "" || numMinions < 0 {
		return nil
	}

	// ensure its a seed if its not blank
	strkey.MustDecode(strkey.VersionByteSeed, friendbotSecret)

	hclient := &horizon.Client{
		URL:     horizonURL,
		HTTP:    http.DefaultClient,
		AppName: "friendbot",
	}

	minions := make([]internal.Minion, numMinions)
	for i := 0; i < numMinions; i++ {
		minionSecret, err := createNewAccount(friendbotSecret, networkPassphrase, hclient)
		if err != nil {
			return nil
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
	}
}

func createNewAccount(sourceSeed, networkPassphrase string, hclient *horizon.Client) (string, error) {
	destKeypair, err := keypair.Random()
	if err != nil {
		return "", nil
	}
	destAddr := destKeypair.Address()
	tx, err := build.Transaction(
		build.SourceAccount{AddressOrSeed: sourceSeed},
		build.Network{Passphrase: networkPassphrase},
		build.AutoSequence{SequenceProvider: hclient},
		build.CreateAccount(
			build.Destination{AddressOrSeed: destAddr},
		),
	)

	txe, err := tx.Sign(sourceSeed)
	if err != nil {
		return "", nil
	}

	txeB64, err := txe.Base64()
	if err != nil {
		return "", nil
	}
	_, err = hclient.SubmitTransaction(txeB64)
	if err != nil {
		return "", nil
	}
	return destKeypair.Seed(), nil
}
