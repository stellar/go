package main

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/txnbuild"
)

// TransactionSubmitter is responsible for sending transactions to Stellar network
type TransactionSubmitter struct {
	Horizon       horizonclient.ClientInterface
	MasterAccount string
	Channels      []*Channel
	Store         PostgresStore
	log           *log.Entry

	// This counter is used to get the next channel (transactionCounter % len(channels))
	transactionCounterMutex sync.Mutex
	transactionCounter      uint64
}

// TransactionPerSecond indicates how many transaction should be sent every second.
const TransactionPerSecond int = 20

// init initialized struct fields that couldn't be injected
func (ts *TransactionSubmitter) init() (err error) {
	// Logger
	ts.log = log.WithFields(log.F{
		"service": "TransactionSubmitter",
	})

	ts.transactionCounter = 0

	// Load channels
	for i, channel := range ts.Channels {
		ts.log.WithField("i", i).Info("Initializing channel")
		accountID, sequenceNumber, err := channel.ReloadState(ts.Horizon)
		if err != nil {
			return err
		}
		ts.log.WithFields(log.F{
			"i":               i,
			"account_id":      accountID,
			"sequence_number": sequenceNumber,
		}).Info("Channel initialized")
	}

	return
}

// Start starts the service. This service works in the following way.
//
//  1. First, it initializes channels by loading their sequence numbers.
//
//  2. Then, it starts a new database transaction in which:
//
//     - it loads `TransactionPerSecond` transactions every second with state equal `pending`,
//     - updates state of loaded transactions to `sending`.
//
//  3. For each transaction, it calculates the hash and saves it. This allows
//     checking if transaction was successfully submitter or not if the app or server
//     crashes at this point.
//
//  4. Transaction is submitted and the state is changed to `sent` or `error`.
//
// TODO Processing transactions in `sending` state that were not sent?
func (ts *TransactionSubmitter) Start(ctx context.Context) {
	err := ts.init()
	if err != nil {
		ts.log.WithError(err).Fatal("Could not initialize TransactionSubmitter")
	}

	for {
		time.Sleep(time.Second)

		transactions, err := ts.Store.LoadPendingTransactionsAndMarkSending(ctx, TransactionPerSecond)
		if err != nil {
			ts.log.WithError(err).Error("Error loading queued transactions")
		} else {
			for _, transaction := range transactions {
				go ts.processTransaction(ctx, transaction)
			}
		}
	}
}

// processTransaction builds transaction using free channel and submits it to horizon.
// If bad_seq error is returned it will call Channel.ReloadState()
func (ts *TransactionSubmitter) processTransaction(ctx context.Context, transaction *Transaction) {
	channel, channelID := ts.getChannel()

	log := ts.log.WithFields(log.F{
		"transaction_id": transaction.ID,
		"destination":    transaction.Destination,
		"amount":         transaction.Amount,
		"channel_id":     channelID,
		"channel":        channel.GetAccountID(),
	})

	log.Info("Processing transaction")

	if transaction.State != TransactionStateSending {
		log.WithField("state", transaction.State).Error("transaction in an invalid state")
		return
	}

	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &txnbuild.SimpleAccount{
				AccountID: channel.GetAccountID(),
				Sequence:  channel.GetSequenceNumber(),
			},
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					SourceAccount: ts.MasterAccount,
					Amount:        transaction.Amount,
					Destination:   transaction.Destination,
					Asset:         txnbuild.NativeAsset{},
				},
			},
			BaseFee: txnbuild.MinBaseFee,
			Preconditions: txnbuild.Preconditions{
				TimeBounds: txnbuild.NewInfiniteTimeout(),
			},
		},
	)
	if err != nil {
		log.WithError(err).Error("error building transaction")
		return
	}

	txHash, err := tx.Hash(network.PublicNetworkPassphrase)
	if err != nil {
		log.WithError(err).Error("error building transaction")
		return
	}

	// Important: We need to save tx hash before submitting a transaction.
	// If the script/server crashes after transaction is submitted but before the response
	// is processed, we can easily determine whether tx was sent or not later using tx hash.
	err = ts.Store.UpdateTransactionHash(ctx, transaction, hex.EncodeToString(txHash[:]))
	if err != nil {
		log.WithError(err).Error("error saving transaction hash")
		return
	}

	_, err = ts.Horizon.SubmitTransaction(tx)
	if err != nil {
		horizonError, ok := err.(*horizonclient.Error)
		if ok {
			log.WithError(err).Error("error submitting transaction")

			// Check for bad_seq errors
			if horizonError.Problem.Extras["result_codes"] != nil {
				resultCodes, ok := horizonError.Problem.Extras["result_codes"].(map[string]interface{})
				if ok {
					if resultCodes["transaction"] == "tx_bad_seq" {
						log.Warn("bad sequence error - reloading channel sequence number")
						_, _, err := channel.ReloadState(ts.Horizon)
						if err != nil {
							log.WithError(err).Error("Error reloading channel sequence number")
						}
					}
				}
			}
		} else {
			log.WithError(err).Error("error submitting transaction")
		}

		err = ts.Store.UpdateTransactionError(ctx, transaction)
		if err != nil {
			log.WithError(err).Error("Error saving transaction error state")
		}

		return
	}

	err = ts.Store.UpdateTransactionSuccess(ctx, transaction)
	if err != nil {
		log.WithError(err).Error("Error saving transaction sent state")
		return
	}

	log.Info("Success submitting transaction")
}

// getChannel returns the current channel
func (ts *TransactionSubmitter) getChannel() (*Channel, uint64) {
	ts.transactionCounterMutex.Lock()
	channelID := ts.transactionCounter
	ts.transactionCounter++
	ts.transactionCounterMutex.Unlock()
	return ts.Channels[channelID], channelID
}
