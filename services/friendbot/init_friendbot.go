package main

import (
	"fmt"
	"net/http"
	"strconv"

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
	numMinions uint16,
) (*internal.Bot, error) {
	if friendbotSecret == "" || networkPassphrase == "" || horizonURL == "" || startingBalance == "" {
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

	minionBalance, err := getMinionBalance(startingBalance, numMinions)
	if err != nil {
		return nil, errors.Wrap(err, "getting minion balance")
	}
	minions, err := createMinionAccounts(botAccount, botKeypair, minionBalance, networkPassphrase, numMinions, hclient)
	if err != nil {
		return nil, errors.Wrap(err, "creating minion accounts")
	}

	return &internal.Bot{
		Horizon:           hclient,
		Account:           botAccount,
		Keypair:           botKeypair,
		Network:           networkPassphrase,
		StartingBalance:   startingBalance,
		SubmitTransaction: internal.AsyncSubmitTransaction,
		Minions:           minions,
	}, nil

}

func createMinionAccounts(botAccount txnbuild.Account, botKeypair *keypair.Full, minionBalance, networkPassphrase string, numMinions uint16, hclient *horizonclient.Client) ([]internal.Minion, error) {
	var (
		ops     []txnbuild.Operation
		minions []internal.Minion
	)
	signers := []*keypair.Full{botKeypair}

	for i := uint16(0); i < numMinions; i++ {
		minionKeypair, err := keypair.Random()
		if err != nil {
			return []internal.Minion{}, errors.Wrap(err, "making keypair")
		}
		signers = append(signers, minionKeypair)

		minions = append(minions, internal.Minion{
			SourceAccount: &internal.Account{AccountID: minionKeypair.Address()},
			Keypair:       minionKeypair,
		})

		ops = append(ops, &txnbuild.CreateAccount{
			Destination: minionKeypair.Address(),
			Amount:      minionBalance,
		})
	}

	txn := txnbuild.Transaction{
		SourceAccount: botAccount,
		Operations:    ops,
		Timebounds:    txnbuild.NewTimebounds(0, 300),
		Network:       networkPassphrase,
	}
	txe, err := txn.BuildSignEncode(signers...)
	if err != nil {
		return []internal.Minion{}, errors.Wrap(err, "making create accounts tx")
	}
	_, err = hclient.SubmitTransactionXDR(txe)
	if err != nil {
		return []internal.Minion{}, errors.Wrap(err, "submitting create accounts tx")
	}
	return minions, nil
}

func getMinionBalance(botBalanceStr string, numMinions uint16) (string, error) {
	botBalanceFloat, err := strconv.ParseFloat(botBalanceStr, 64)
	if err != nil {
		return "", errors.Wrap(err, "parsing bot balance")
	}
	minionBalanceStr := fmt.Sprintf("%f", botBalanceFloat/float64(numMinions))
	return minionBalanceStr, nil
}
