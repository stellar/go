package internal

import (
	"strconv"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

// Account implements the `txnbuild.Account` interface.
type Account struct {
	AccountID string
	Sequence  xdr.SequenceNumber
}

// GetAccountID returns the Account ID.
func (a Account) GetAccountID() string {
	return a.AccountID
}

// IncrementSequenceNumber increments the internal record of the
// account's sequence number by 1.
func (a Account) IncrementSequenceNumber() (xdr.SequenceNumber, error) {
	a.Sequence++
	return a.Sequence, nil
}

// Minion contains a Stellar channel account and Go channels to communicate with friendbot.
type Minion struct {
	Account Account
	Keypair *keypair.Full

	// Uninitialized.
	forceRefreshSequence bool
}

// MinionInput is the input to the Minion from the Bot to construct a payment.
type MinionInput struct {
	botAccount      txnbuild.Account
	botKeypair      *keypair.Full
	destAddress     string
	network         string
	startingBalance string
}

// TxResult is the result from the asynchronous payment tx construction method.
type TxResult struct {
	maybeTxXDR *string
	maybeErr   error
}

// SubmitResult is the result from the asynchronous submit transaction method over a channel.
type SubmitResult struct {
	maybeTransactionSuccess *hProtocol.TransactionSuccess
	maybeErr                error
}

// Bot represents the friendbot subsystem.
type Bot struct {
	Horizon           *horizonclient.Client
	Account           txnbuild.Account
	Keypair           *keypair.Full
	Network           string
	StartingBalance   string
	SubmitTransaction func(minion *Minion, channel chan SubmitResult, hclient *horizonclient.Client, signed string)
	Minions           []Minion
	nextMinionIndex   int
}

// Pay funds the account at `destAddress`.
func (bot *Bot) Pay(destAddress string) (*hProtocol.TransactionSuccess, error) {
	minion := bot.Minions[bot.nextMinionIndex]
	// XXX: Launch separate thread, potentially.
	input := MinionInput{
		destAddress:     destAddress,
		botAccount:      bot.Account,
		botKeypair:      bot.Keypair,
		network:         bot.Network,
		startingBalance: bot.StartingBalance,
	}

	txResultChan := make(chan TxResult)
	go minion.run(bot.Horizon, input, txResultChan)
	maybeTxResult := <-txResultChan
	if maybeTxResult.maybeErr != nil {
		return nil, errors.Wrap(maybeTxResult.maybeErr, "making tx")
	}

	submitResultChan := make(chan SubmitResult)
	go bot.SubmitTransaction(&minion, submitResultChan, bot.Horizon, *maybeTxResult.maybeTxXDR)
	bot.nextMinionIndex = (bot.nextMinionIndex + 1) % len(bot.Minions)

	maybeSubmitResult := <-submitResultChan
	return maybeSubmitResult.maybeTransactionSuccess, maybeSubmitResult.maybeErr
}

// AsyncSubmitTransaction should be passed to the bot.
func AsyncSubmitTransaction(minion *Minion, channel chan SubmitResult, hclient *horizonclient.Client, txXDR string) {
	result, err := hclient.SubmitTransactionXDR(txXDR)
	if err != nil {
		switch e := err.(type) {
		case *horizonclient.Error:
			minion.checkHandleBadSequence(e)
		}

		channel <- SubmitResult{
			maybeTransactionSuccess: nil,
			maybeErr:                err,
		}
	} else {
		channel <- SubmitResult{
			maybeTransactionSuccess: &result,
			maybeErr:                nil,
		}
	}
}

func (minion *Minion) run(hclient *horizonclient.Client, input MinionInput, channel chan TxResult) {
	err := minion.checkSequenceRefresh(hclient)
	if err != nil {
		channel <- TxResult{
			maybeTxXDR: nil,
			maybeErr:   errors.Wrap(err, "checking minion seq"),
		}
	}
	txStr, err := minion.makeTx(input)
	channel <- TxResult{
		maybeTxXDR: txStr,
		maybeErr:   errors.Wrap(err, "making payment tx"),
	}
}

func (minion *Minion) checkHandleBadSequence(err *horizonclient.Error) {
	resCode, e := err.ResultCodes()
	isTxBadSeqCode := e == nil && resCode.TransactionCode == "tx_bad_seq"
	if !isTxBadSeqCode {
		return
	}
	minion.forceRefreshSequence = true
}

// Establishes the minion's initial sequence number, if needed.
func (minion *Minion) checkSequenceRefresh(hclient *horizonclient.Client) error {
	if minion.Account.Sequence != 0 && !minion.forceRefreshSequence {
		return nil
	}
	return minion.refreshSequence(hclient)
}

func (minion *Minion) makeTx(input MinionInput) (*string, error) {
	createAccountOp := txnbuild.CreateAccount{
		Destination: input.destAddress,
		Amount:      input.startingBalance,
	}
	paymentOp := txnbuild.Payment{
		Destination:   input.destAddress,
		Amount:        input.startingBalance,
		Asset:         txnbuild.NativeAsset{},
		SourceAccount: input.botAccount,
	}
	txn := txnbuild.Transaction{
		SourceAccount: minion.Account,
		Operations:    []txnbuild.Operation{&createAccountOp, &paymentOp},
		Network:       input.network,
		Timebounds:    txnbuild.NewTimebounds(0, 300),
	}
	txe, err := txn.BuildSignEncode(minion.Keypair, input.botKeypair)
	if err != nil {
		return nil, errors.Wrap(err, "making account payment tx")
	}
	// Increment the in-memory sequence number, since the tx will be submitted.
	_, err = minion.Account.IncrementSequenceNumber()
	if err != nil {
		return nil, errors.Wrap(err, "incrementing minion seq")
	}
	return &txe, err
}

// Refreshes the in-memory sequence number from the minion Stellar account.
func (minion *Minion) refreshSequence(hclient *horizonclient.Client) error {
	minion.Account.Sequence = 0
	accountRequest := horizonclient.AccountRequest{AccountID: minion.Account.GetAccountID()}
	minionAccount, err := hclient.AccountDetail(accountRequest)
	if err != nil {
		return err
	}
	seq, err := strconv.ParseInt(minionAccount.Sequence, 10, 64)
	if err != nil {
		return err
	}
	minion.Account.Sequence = xdr.SequenceNumber(seq)
	minion.forceRefreshSequence = false
	return nil
}
