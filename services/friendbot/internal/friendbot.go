package internal

import (
	"strconv"

	b "github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/errors"
)

// Minion contains a Stellar channel account and Go channels to communicate with friendbot.
type Minion struct {
	Secret       string
	DestAddrChan chan string
	TxResultChan chan TxResult

	// Uninitialized.
	sequence             uint64
	forceRefreshSequence bool
}

// TxResult is the result from the asynchronous submit transaction method over a channel.
type TxResult struct {
	maybeTransactionSuccess *horizon.TransactionSuccess
	maybeErr                error
}

// Bot represents the friendbot subsystem.
type Bot struct {
	Horizon           *horizon.Client
	Secret            string
	Network           string
	StartingBalance   string
	SubmitTransaction func(minion *Minion, hclient *horizon.Client, signed string)
	Minions           []Minion
	nextMinionIndex   int
}

// Pay funds the account at `destAddress`.
func (bot *Bot) Pay(destAddress string) (*horizon.TransactionSuccess, error) {
	minion := bot.Minions[bot.nextMinionIndex]
	err := minion.checkSequenceRefresh(bot.Horizon)
	if err != nil {
		return nil, err
	}

	signed, err := minion.makeTx(destAddress, bot.Secret, bot.Network, bot.StartingBalance)
	if err != nil {
		return nil, err
	}
	go bot.SubmitTransaction(&minion, bot.Horizon, signed)
	bot.nextMinionIndex = (bot.nextMinionIndex + 1) % len(bot.Minions)

	v := <-minion.TxResultChan
	return v.maybeTransactionSuccess, v.maybeErr
}

// AsyncSubmitTransaction should be passed to the bot.
func AsyncSubmitTransaction(minion *Minion, hclient *horizon.Client, signed string) {
	result, err := hclient.SubmitTransaction(signed)
	if err != nil {
		switch e := err.(type) {
		case *horizon.Error:
			minion.checkHandleBadSequence(e)
		}

		minion.TxResultChan <- TxResult{
			maybeTransactionSuccess: nil,
			maybeErr:                err,
		}
	} else {
		minion.TxResultChan <- TxResult{
			maybeTransactionSuccess: &result,
			maybeErr:                nil,
		}
	}
}

func (minion *Minion) checkHandleBadSequence(err *horizon.Error) {
	resCode, e := err.ResultCodes()
	isTxBadSeqCode := e == nil && resCode.TransactionCode == "tx_bad_seq"
	if !isTxBadSeqCode {
		return
	}
	minion.forceRefreshSequence = true
}

// Establishes the minion's initial sequence number, if needed.
func (minion *Minion) checkSequenceRefresh(hclient *horizon.Client) error {
	if minion.sequence != 0 && !minion.forceRefreshSequence {
		return nil
	}
	return minion.refreshSequence(hclient)
}

func (minion *Minion) makeTx(destAddress, botSecret, networkPassphrase, initBalance string) (string, error) {
	txn, err := b.Transaction(
		b.SourceAccount{AddressOrSeed: minion.Secret},
		b.Sequence{Sequence: minion.sequence + 1},
		b.Network{Passphrase: networkPassphrase},
		b.CreateAccount(
			b.Destination{AddressOrSeed: destAddress},
		),
		b.Payment(
			b.SourceAccount{AddressOrSeed: botSecret},
			b.Destination{AddressOrSeed: destAddress},
			b.NativeAmount{Amount: initBalance},
		),
	)

	if err != nil {
		return "", errors.Wrap(err, "Error building a transaction")
	}

	txs, err := txn.Sign(botSecret, minion.Secret)
	if err != nil {
		return "", errors.Wrap(err, "Error signing a transaction")
	}

	base64, err := txs.Base64()

	// We only increment the in-memory sequence number if the tx will be submitted.
	if err == nil {
		minion.sequence++
	}
	return base64, err
}

// Refreshes the in-memory sequence number from the minion Stellar account.
func (minion *Minion) refreshSequence(hclient *horizon.Client) error {
	minionAccount, err := hclient.LoadAccount(minion.address())
	if err != nil {
		minion.sequence = 0
		return err
	}
	seq, err := strconv.ParseInt(minionAccount.Sequence, 10, 64)
	if err != nil {
		minion.sequence = 0
		return err
	}
	minion.sequence = uint64(seq)
	minion.forceRefreshSequence = false
	return nil
}

func (minion *Minion) address() string {
	kp := keypair.MustParse(minion.Secret)
	return kp.Address()
}
