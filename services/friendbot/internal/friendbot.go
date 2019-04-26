package internal

import (
	"strconv"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/clients/horizonclient"
	b "github.com/stellar/go/exp/txnbuild"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/errors"
)

// Minion contains a Stellar channel account and Go channels to communicate with friendbot.
type Minion struct {
	Secret string
	// XXX: Rename field
	txInputChan chan TxInput

	// Uninitialized.
	sequence             uint64
	forceRefreshSequence bool
}

// TxInput is the input to the Minion for constructing a payment transaction.
type TxInput struct {
	destAddress     string
	botSecret       string
	network         string
	startingBalance string
}

// TxResult is the result from the asynchronous payment tx construction method.
type TxResult struct {
	// XXX: Check type
	maybeTransaction *string
	maybeErr         error
}

// SubmitResult is the result from the asynchronous submit transaction method over a channel.
type SubmitResult struct {
	// XXX: Change this
	maybeTransactionSuccess *horizon.TransactionSuccess
	maybeErr                error
}

// Bot represents the friendbot subsystem.
type Bot struct {
	Horizon           *horizonclient.Client
	Secret            string
	Network           string
	StartingBalance   string
	SubmitTransaction func(minion *Minion, channel chan SubmitResult, hclient *horizonclient.Client, signed string)
	Minions           []Minion
	nextMinionIndex   int
}

// Pay funds the account at `destAddress`.
// XXX: Change return type
func (bot *Bot) Pay(destAddress string) (*horizon.TransactionSuccess, error) {
	minion := bot.Minions[bot.nextMinionIndex]
	// XXX: Launch separate thread
	input := TxInput{
		destAddress:     destAddress,
		botSecret:       bot.Secret,
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
	go bot.SubmitTransaction(&minion, submitResultChan, bot.Horizon, *maybeTxResult.maybeTransaction)
	bot.nextMinionIndex = (bot.nextMinionIndex + 1) % len(bot.Minions)

	maybeSubmitResult := <-submitResultChan
	return maybeSubmitResult.maybeTransactionSuccess, maybeSubmitResult.maybeErr
}

// AsyncSubmitTransaction should be passed to the bot.
func AsyncSubmitTransaction(minion *Minion, channel chan SubmitResult, hclient *horizonclient.Client, signed string) {
	// XXX: Change input to SubmitTransaction
	result, err := hclient.SubmitTransaction(signed)
	if err != nil {
		switch e := err.(type) {
		// XXX: Change horizon type
		case *horizon.Error:
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

func (minion *Minion) run(hclient *horizonclient.Client, input TxInput, channel chan TxResult) {
	err := minion.checkSequenceRefresh(hclient)
	if err != nil {
		channel <- TxResult{
			maybeTransaction: nil,
			maybeErr:         errors.Wrap(err, "checking minion seq"),
		}
	}
	signed, err := minion.makeTx(input)
	if err != nil {
		channel <- TxResult{
			maybeTransaction: nil,
			maybeErr:         errors.Wrap(err, "making payment tx"),
		}
	}
	channel <- TxResult{
		maybeTransaction: &signed,
		maybeErr:         nil,
	}
}

// XXX: Change param type
func (minion *Minion) checkHandleBadSequence(err *horizon.Error) {
	resCode, e := err.ResultCodes()
	isTxBadSeqCode := e == nil && resCode.TransactionCode == "tx_bad_seq"
	if !isTxBadSeqCode {
		return
	}
	minion.forceRefreshSequence = true
}

// Establishes the minion's initial sequence number, if needed.
func (minion *Minion) checkSequenceRefresh(hclient *horizonclient.Client) error {
	if minion.sequence != 0 && !minion.forceRefreshSequence {
		return nil
	}
	return minion.refreshSequence(hclient)
}

func (minion *Minion) makeTx(input TxInput) (string, error) {
	txnOld, err := b.Transaction(
		b.SourceAccount{AddressOrSeed: minion.Secret},
		b.Sequence{Sequence: minion.sequence + 1},
		b.Network{Passphrase: input.network},
		b.CreateAccount(
			b.Destination{AddressOrSeed: input.destAddress},
		),
		b.Payment(
			b.SourceAccount{AddressOrSeed: input.botSecret},
			b.Destination{AddressOrSeed: input.destAddress},
			b.NativeAmount{Amount: input.startingBalance},
		),
	)

	if err != nil {
		return "", errors.Wrap(err, "Error building a transaction")
	}

	txs, err := txn.Sign(input.botSecret, minion.Secret)
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
func (minion *Minion) refreshSequence(hclient *horizonclient.Client) error {
	// XXX: Change the LoadAccount call
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
