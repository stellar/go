package server

import (
	"github.com/stellar/go/services/bifrost/database"
	"github.com/stellar/go/services/bifrost/ethereum"
	"github.com/stellar/go/services/bifrost/queue"
	"github.com/stellar/go/services/bifrost/sse"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

// onNewEthereumTransaction checks if transaction is valid and adds it to
// the transactions queue for StellarAccountConfigurator to consume.
//
// Transaction added to transactions queue should be in a format described in
// queue.Transaction (especialy amounts). Pooling service should not have to deal with any
// conversions.
func (s *Server) onNewEthereumTransaction(transaction ethereum.Transaction) error {
	localLog := s.log.WithFields(log.F{"transaction": transaction, "rail": "ethereum"})
	localLog.Debug("Processing transaction")

	// Let's check if tx is valid first.

	// Check if value is above minimum required
	if transaction.ValueWei.Cmp(s.minimumValueWei) < 0 {
		localLog.Debug("Value is below minimum required amount, skipping")
		return nil
	}

	addressAssociation, err := s.Database.GetAssociationByChainAddress(database.ChainEthereum, transaction.To)
	if err != nil {
		return errors.Wrap(err, "Error getting association")
	}

	if addressAssociation == nil {
		localLog.Debug("Associated address not found, skipping")
		return nil
	}

	// Add transaction as processing.
	processed, err := s.Database.AddProcessedTransaction(database.ChainEthereum, transaction.Hash, transaction.To)
	if err != nil {
		return err
	}

	if processed {
		localLog.Debug("Transaction already processed, skipping")
		return nil
	}

	// Add tx to the processing queue
	queueTx := queue.Transaction{
		TransactionID: transaction.Hash,
		AssetCode:     queue.AssetCodeETH,
		// Amount in the base unit of currency.
		Amount:           transaction.ValueToStellar(),
		StellarPublicKey: addressAssociation.StellarPublicKey,
	}

	err = s.TransactionsQueue.QueueAdd(queueTx)
	if err != nil {
		return errors.Wrap(err, "Error adding transaction to the processing queue")
	}
	localLog.Info("Transaction added to transaction queue")

	// Broadcast event to address stream
	s.SSEServer.BroadcastEvent(transaction.To, sse.TransactionReceivedAddressEvent, nil)
	localLog.Info("Transaction processed successfully")
	return nil
}
