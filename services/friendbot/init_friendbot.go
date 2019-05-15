package main

import (
	"log"
	"net/http"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/friendbot/internal"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/txnbuild"
)

func initFriendbot(
	friendbotSecret string,
	networkPassphrase string,
	horizonURL string,
	startingBalance string,
	numMinions int,
) (*internal.Bot, error) {
	if friendbotSecret == "" || networkPassphrase == "" || horizonURL == "" || startingBalance == "" || numMinions < 0 {
		return nil, errors.New("invalid input param(s)")
	}

	// Guarantee that friendbotSecret is a seed, if not blank.
	strkey.MustDecode(strkey.VersionByteSeed, friendbotSecret)

	hclient := &horizonclient.Client{
		HorizonURL: horizonURL,
		HTTP:       http.DefaultClient,
		AppName:    "friendbot",
	}

	botKP, err := keypair.Parse(friendbotSecret)
	if err != nil {
		return nil, errors.Wrap(err, "parsing bot keypair")
	}

	// Casting from the interface type will work, since we
	// already confirmed that friendbotSecret is a seed.
	botKeypair := botKP.(*keypair.Full)
	botAccount := internal.Account{AccountID: botKeypair.Address()}
	minionBalance := "101.00"
	minions, err := createMinionAccounts(botAccount, botKeypair, networkPassphrase, startingBalance, minionBalance, numMinions, hclient)
	if err != nil {
		return nil, errors.Wrap(err, "creating minion accounts")
	}

	return &internal.Bot{Minions: minions}, nil
}

func createMinionAccounts(botAccount internal.Account, botKeypair *keypair.Full, networkPassphrase, newAccountBalance, minionBalance string, numMinions int, hclient *horizonclient.Client) ([]internal.Minion, error) {
	var minions []internal.Minion
	numRemainingMinions := numMinions
	minionBatchSize := 100
	for numRemainingMinions > 0 {
		var ops []txnbuild.Operation
		// Refresh the sequence number before submitting a new transaction.
		err := botAccount.RefreshSequenceNumber(hclient)
		if err != nil {
			return nil, errors.Wrap(err, "refreshing bot seqnum")
		}
		// The tx will create min(numRemainingMinions, 100) Minion accounts.
		numCreateMinions := minionBatchSize
		if numRemainingMinions < minionBatchSize {
			numCreateMinions = numRemainingMinions
		}
		for i := 0; i < numCreateMinions; i++ {
			minionKeypair, err := keypair.Random()
			if err != nil {
				return []internal.Minion{}, errors.Wrap(err, "making keypair")
			}
			minions = append(minions, internal.Minion{
				Account:           internal.Account{AccountID: minionKeypair.Address()},
				Keypair:           minionKeypair,
				BotAccount:        botAccount,
				BotKeypair:        botKeypair,
				Horizon:           hclient,
				Network:           networkPassphrase,
				StartingBalance:   newAccountBalance,
				SubmitTransaction: internal.SubmitTransaction,
			})

			ops = append(ops, &txnbuild.CreateAccount{
				Destination: minionKeypair.Address(),
				Amount:      minionBalance,
			})
		}
		numRemainingMinions -= numCreateMinions

		// Build and submit batched account creation tx.
		txn := txnbuild.Transaction{
			SourceAccount: botAccount,
			Operations:    ops,
			Timebounds:    txnbuild.NewTimeout(300),
			Network:       networkPassphrase,
		}
		txe, err := txn.BuildSignEncode(botKeypair)
		if err != nil {
			return []internal.Minion{}, errors.Wrap(err, "making create accounts tx")
		}
		resp, err := hclient.SubmitTransactionXDR(txe)
		if err != nil {
			log.Print(resp)
			return []internal.Minion{}, errors.Wrap(err, "submitting create accounts tx")
		}
	}
	return minions, nil
}
