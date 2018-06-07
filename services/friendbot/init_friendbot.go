package main

import (
	"net/http"

	"github.com/stellar/go/services/friendbot/internal"
	"github.com/stellar/go/strkey"
	hClient "github.com/stellar/go/clients/horizon"
)

func initFriendbot(
	friendbotSecret string,
	networkPassphrase string,
	horizonURL string,
	startingBalance string,
) *internal.Bot {

	if friendbotSecret == "" || networkPassphrase == "" || horizonURL == "" || startingBalance == "" {
		return nil
	}

	// ensure its a seed if its not blank
	strkey.MustDecode(strkey.VersionByteSeed, friendbotSecret)

	return &internal.Bot{
		Secret: friendbotSecret,
		Horizon: hClient.Client{
			URL:  horizonURL,
			HTTP: http.DefaultClient,
		},
		Network:           networkPassphrase,
		StartingBalance:   startingBalance,
		SubmitTransaction: internal.AsyncSubmitTransaction,
	}
}
