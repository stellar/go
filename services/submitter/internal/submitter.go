package internal

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/txnbuild"
)

// TransactionSubmitter is responsible for sending transactions to Stellar network
type TransactionSubmitter struct {
	Horizon         horizonclient.ClientInterface
	RootAccountSeed string
	NumChannels     uint
	Store           PostgresStore
	Network         string
	MaxBaseFee      uint

	log                 *log.Entry
	channels            []*Channel
	pendingTransactions chan *Transaction
}

// derive, create, and load channel accounts
// initialize private struct variables
func (ts *TransactionSubmitter) init() (err error) {
	// Logger
	ts.log = log.WithFields(log.F{
		"service": "TransactionSubmitter",
	})
	ts.log.SetLevel(log.InfoLevel)

	// pendingTransactions channel
	ts.pendingTransactions = make(chan *Transaction, len(ts.channels))

	ts.channels, err = DeriveChannelsFromSeed(
		ts.RootAccountSeed,
		uint32(ts.NumChannels),
		0,
	)
	if err != nil {
		return err
	}
	for _, channel := range ts.channels {
		channelKp := keypair.MustParseFull(channel.Seed)
		ts.log.WithFields(log.F{
			"account": channelKp.Address(),
		}).Info("derived channel account")
	}

	notFoundChannels, err := ts.loadChannels(ts.channels)
	if err != nil {
		return err
	}
	if len(notFoundChannels) == 0 {
		return
	}

	ts.log.WithFields(log.F{
		"num_accounts": len(notFoundChannels),
	}).Info("creating channel accounts")

	// we're limited to 19 account creation ops per tx due to the 20 signatures per tx limit, so
	numCreateAccountTxs := len(notFoundChannels) / 19
	if len(notFoundChannels)%19 != 0 {
		numCreateAccountTxs += 1
	}

	ts.log.WithFields(log.F{
		"num_transactions": numCreateAccountTxs,
	}).Info("submitting transactions to create accounts")

	rootKp, err := keypair.ParseFull(ts.RootAccountSeed)
	if err != nil {
		return err
	}

	rootAccount, err := ts.Horizon.AccountDetail(horizonclient.AccountRequest{
		AccountID: rootKp.Address(),
	})
	if err != nil {
		return err
	}

	var createAccountTxs []*txnbuild.Transaction
	for i := 0; i < numCreateAccountTxs; i++ {
		ts.log.WithFields(log.F{
			"i": i,
		}).Info("building transaction")
		var numOpsForTx int
		if numOpsForTx = len(notFoundChannels[i*19:]); numOpsForTx > 19 {
			numOpsForTx = 19
		}
		var txKps = []*keypair.Full{rootKp}
		var sponsoredCreateAccountOps []txnbuild.Operation
		for j := i * 19; j < i*19+numOpsForTx; j++ {
			ts.log.WithFields(log.F{
				"j": j,
			}).Info("building operations")
			channelKp := keypair.MustParseFull(notFoundChannels[j].Seed)
			beginSponsoringOp := txnbuild.BeginSponsoringFutureReserves{
				SponsoredID: channelKp.Address(),
			}
			createAccountOp := txnbuild.CreateAccount{
				Destination: channelKp.Address(),
				Amount:      "0",
			}
			endSponsoringOp := txnbuild.EndSponsoringFutureReserves{
				SourceAccount: channelKp.Address(),
			}
			txKps = append(txKps, channelKp)
			sponsoredCreateAccountOps = append(
				sponsoredCreateAccountOps,
				&beginSponsoringOp,
				&createAccountOp,
				&endSponsoringOp,
			)
		}
		tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
			SourceAccount:        &rootAccount,
			IncrementSequenceNum: true,
			Operations:           sponsoredCreateAccountOps,
			BaseFee:              int64(ts.MaxBaseFee),
			Preconditions: txnbuild.Preconditions{
				TimeBounds: txnbuild.NewInfiniteTimeout(),
			},
		})
		if err != nil {
			return err
		}
		txBase64, err := tx.Base64()
		if err != nil {
			return err
		}
		ts.log.WithFields(log.F{
			"transaction": txBase64,
		}).Info("signing transaction")
		tx, err = tx.Sign(ts.Network, txKps...)
		if err != nil {
			return err
		}
		txBase64, err = tx.Base64()
		if err != nil {
			return err
		}
		ts.log.WithFields(log.F{
			"transaction": txBase64,
		}).Info("adding transaction to set")
		createAccountTxs = append(createAccountTxs, tx)
	}

	for _, tx := range createAccountTxs {
		if err = ts.submitTx(tx); err != nil {
			// TODO: handle this error more thoughtfully
			// the only possible error returned is for insufficient funds
			hErr := horizonclient.GetError(err)
			ts.log.Infof("submission error: {}", hErr.Problem)
			return err
		}
	}

	notFoundChannels, err = ts.loadChannels(notFoundChannels)
	if err != nil {
		return err
	}
	if len(notFoundChannels) != 0 {
		ts.log.WithFields(log.F{
			"not_found_channels": notFoundChannels,
		}).Info("couldn't fetch channels after attempting to create them")
		return errors.New("unable to create the number of channels requested")
	}

	return
}

func (ts *TransactionSubmitter) loadChannels(channels []*Channel) (notFoundChannels []*Channel, err error) {
	for _, channel := range channels {
		if err := channel.LoadState(ts.Horizon); err != nil {
			if horizonclient.IsNotFoundError(err) {
				notFoundChannels = append(notFoundChannels, channel)
				continue
			}
			return notFoundChannels, err
		}
		ts.log.WithFields(log.F{
			"account_id":      channel.accountID,
			"sequence_number": channel.sequenceNumber,
		}).Info("Channel initialized")
	}
	return notFoundChannels, nil
}

// Start starts the service. This service works in the following way.
//
//  1. It initializes channels by loading their sequence numbers.
//
//  2. It starts a go routine to listen on the pendningTransactions buffered channel
//
//  3. Every second, it:
//
//     - checks if the buffered channel is full
//     - if the channel is not full, it loads transactions with state equal `pending`
//     - updates state of loaded transactions to `sending`
//     - queues them in the buffered channel
//
// See listenForPendingTransactions() for information on how transactions are processed.
func (ts *TransactionSubmitter) Start(ctx context.Context) {
	err := ts.init()
	if err != nil {
		ts.log.WithError(err).Fatal("Could not initialize TransactionSubmitter")
		return
	}

	for _, channel := range ts.channels {
		go ts.listenForPendingTransactions(ctx, channel)
	}

	for {
		time.Sleep(time.Second)
		if len(ts.channels) == len(ts.pendingTransactions) {
			continue
		}
		newPendingTransactions, err := ts.Store.LoadPendingTransactionsAndMarkSending(ctx, len(ts.channels)-len(ts.pendingTransactions))
		if err != nil {
			ts.log.WithError(err).Error("Error loading queued transactions")
			continue
		}
		for _, transaction := range newPendingTransactions {
			ts.pendingTransactions <- transaction
		}
	}
}

func (ts *TransactionSubmitter) listenForPendingTransactions(ctx context.Context, channel *Channel) {
	for transaction := range ts.pendingTransactions {
		ts.processTransaction(ctx, transaction, channel)
	}
}

// processTrannsaction manages the database state for a transaction being submitted by a channel
func (ts *TransactionSubmitter) processTransaction(ctx context.Context, transaction *Transaction, channel *Channel) {
	log := ts.log.WithFields(log.F{
		"transaction_id": transaction.ID,
		"destination":    transaction.Destination,
		"amount":         transaction.Amount,
		"channel":        channel.GetAccountID(),
	})

	log.Info("Processing transaction")

	if transaction.State != TransactionStateSending {
		log.WithField("state", transaction.State).Error("transaction in an invalid state")
		return
	}

	feeBumpTx, err := ts.buildTransaction(transaction, channel)
	if err != nil {
		log.WithError(err).Error("error building transaction")
		return
	}

	feeBumpTxHash, err := feeBumpTx.Hash(ts.Network)
	if err != nil {
		log.WithError(err).Error("error hashing transaction")
		return
	}

	// Important: We need to save tx hash before submitting a transaction.
	// If the script/server crashes after transaction is submitted but before the response
	// is processed, we can easily determine whether tx was sent or not later using tx hash.
	err = ts.Store.UpdateTransactionHash(ctx, transaction, hex.EncodeToString(feeBumpTxHash[:]))
	if err != nil {
		log.WithError(err).Error("error saving transaction hash")
		return
	}

	err = ts.submitFeeBumpTx(feeBumpTx)
	if err != nil {
		log.WithError(err).Error("Error submitting transaction")
		err = ts.Store.UpdateTransactionError(ctx, transaction)
	} else {
		log.Info("Success submitting transaction")
		err = ts.Store.UpdateTransactionSuccess(ctx, transaction)
	}

	if err != nil {
		log.WithError(err).Error("Error saving transaction sent state")
		return
	}
}

func (ts *TransactionSubmitter) buildTransaction(t *Transaction, channel *Channel) (feeBumpTx *txnbuild.FeeBumpTransaction, err error) {
	rootAccountKp, err := keypair.ParseFull(ts.RootAccountSeed)
	if err != nil {
		return feeBumpTx, errors.Wrap(err, "unable to parse RootAccountSeed")
	}

	channelAccountKp, err := keypair.ParseFull(channel.Seed)
	if err != nil {
		return feeBumpTx, errors.Wrap(err, "unable to parse Channel.Seed")
	}

	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &txnbuild.SimpleAccount{
				AccountID: channel.GetAccountID(),
				Sequence:  channel.GetSequenceNumber(),
			},
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					SourceAccount: rootAccountKp.Address(),
					Amount:        t.Amount,
					Destination:   t.Destination,
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
		return feeBumpTx, errors.Wrap(err, "error building transaction")
	}

	tx, err = tx.Sign(ts.Network, rootAccountKp, channelAccountKp)
	if err != nil {
		return feeBumpTx, errors.Wrap(err, "error signing transaction")
	}

	feeBumpTx, err = txnbuild.NewFeeBumpTransaction(
		txnbuild.FeeBumpTransactionParams{
			Inner:      tx,
			FeeAccount: rootAccountKp.Address(),
			BaseFee:    txnbuild.MinBaseFee,
		},
	)
	if err != nil {
		return feeBumpTx, errors.Wrap(err, "error building fee-bump transaction")
	}

	feeBumpTx, err = feeBumpTx.Sign(ts.Network, rootAccountKp)
	if err != nil {
		return feeBumpTx, errors.Wrap(err, "error signing fee-bump transaction")
	}

	return feeBumpTx, nil
}

// Submits the transaction and handles any recoverable errors until it gets included in a ledger.
// Returns any error that is not recoverable.
func (ts *TransactionSubmitter) submitFeeBumpTx(t *txnbuild.FeeBumpTransaction) (err error) {
	_, err = ts.Horizon.SubmitFeeBumpTransactionWithOptions(t, horizonclient.SubmitTxOpts{SkipMemoRequiredCheck: true})
	return err
}

func (ts *TransactionSubmitter) submitTx(t *txnbuild.Transaction) (err error) {
	_, err = ts.Horizon.SubmitTransactionWithOptions(t, horizonclient.SubmitTxOpts{SkipMemoRequiredCheck: true})
	return err
}
