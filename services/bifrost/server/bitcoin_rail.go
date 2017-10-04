package server

import (
	"strconv"

	"github.com/stellar/go/services/bifrost/bitcoin"
	"github.com/stellar/go/services/bifrost/database"
	"github.com/stellar/go/services/bifrost/queue"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

// onNewBitcoinTransaction checks if transaction is valid and adds it to
// the transactions queue.
func (s *Server) onNewBitcoinTransaction(transaction bitcoin.Transaction) error {
	localLog := s.log.WithFields(log.F{"transaction": transaction, "rail": "bitcoin"})
	localLog.Debug("Processing transaction")

	// Check if transaction has not been processed
	processed, err := s.Database.IsTransactionProcessed(database.ChainBitcoin, transaction.Hash)
	if err != nil {
		return err
	}

	if processed {
		localLog.Debug("Transaction already processed, skipping")
		return nil
	}

	// Check if value is above minimum required
	// TODO, check actual minimum (so user doesn't get more in XLM than in ETH)
	if transaction.Value <= 0 {
		localLog.Debug("Value is below minimum required amount, skipping")
		return nil
	}

	addressAssociation, err := s.Database.GetAssociationByChainAddress(database.ChainBitcoin, transaction.To)
	if err != nil {
		return errors.Wrap(err, "Error getting association")
	}

	if addressAssociation == nil {
		localLog.Debug("Associated address not found, skipping")
		return nil
	}

	value := strconv.FormatInt(transaction.Value, 10)

	// Add tx to the processing queue
	queueTx := queue.Transaction{
		TransactionID: transaction.Hash,
		AssetCode:     queue.AssetCodeBTC,
		// Amount in the smallest unit of currency.
		// For 1 satoshi = 0.00000001 BTC this should be equal `1`
		Amount:           value,
		StellarPublicKey: addressAssociation.StellarPublicKey,
	}

	err = s.TransactionsQueue.Add(queueTx)
	if err != nil {
		return errors.Wrap(err, "Error adding transaction to the processing queue")
	}

	localLog.Info("Transaction added to transaction queue")

	// Save transaction as processed
	err = s.Database.AddProcessedTransaction(database.ChainBitcoin, transaction.Hash)
	if err != nil {
		return errors.Wrap(err, "Error saving transaction as processed")
	}

	localLog.Info("Transaction processed successfully")

	// Publish event to address stream
	s.publishEvent(transaction.To, TransactionReceivedAddressEvent, nil)

	return nil
}
