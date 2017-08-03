package server

import (
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stellar/go/services/bifrost/database"
	"github.com/stellar/go/services/bifrost/queue"
	"github.com/stellar/go/support/errors"
)

// onNewEthereumTransaction checks if transaction is valid and adds it to
// the transactions queue.
func (s *Server) onNewEthereumTransaction(transaction *types.Transaction) error {
	transactionHash := transaction.Hash().Hex()
	localLog := s.log.WithField("transaction", transactionHash)
	localLog.Debug("Processing transaction")

	// Check if transaction has not been processed
	processed, err := s.Database.IsTransactionProcessed(database.ChainEthereum, transactionHash)
	if err != nil {
		return err
	}

	if processed {
		localLog.Debug("Transaction already processed, skipping")
		return nil
	}

	// Check if transaction is sent to one of our addresses
	to := transaction.To()
	if to == nil {
		// Contract creation
		localLog.Debug("Transaction is a contract creation, skipping")
		return nil
	}

	// Check if value is above minimum required
	// TODO, check actual minimum (so user doesn't get more in XLM than in ETH)
	if transaction.Value().Sign() <= 0 {
		localLog.Debug("Value is below minimum required amount, skipping")
		return nil
	}

	address := to.Hex()

	addressAssociation, err := s.Database.GetAssociationByEthereumAddress(address)
	if err != nil {
		return errors.Wrap(err, "Error getting association")
	}

	if addressAssociation == nil {
		localLog.Debug("Associated address not found, skipping")
		return nil
	}

	// Add tx to the processing queue
	queueTx := queue.Transaction{
		TransactionID: transactionHash,
		AssetCode:     queue.AssetCodeETH,
		// Amount in the smallest unit of currency.
		// For 1 Wei = 0.000000000000000001 ETH this should be equal `1`
		Amount:           transaction.Value().String(),
		StellarPublicKey: addressAssociation.StellarPublicKey,
	}

	err = s.TransactionsQueue.Add(queueTx)
	if err != nil {
		return errors.Wrap(err, "Error adding transaction to the processing queue")
	}

	localLog.Info("Transaction added to transaction queue")

	// Save transaction as processed
	err = s.Database.AddProcessedTransaction(database.ChainEthereum, transactionHash)
	if err != nil {
		return errors.Wrap(err, "Error saving transaction as processed")
	}

	localLog.Info("Transaction processed successfully")

	// Publish event to address stream
	s.publishEvent(address, TransactionReceivedAddressEvent, nil)

	return nil
}

func (s *Server) poolTransactionsQueue() {
	s.log.Info("Started pooling transactions queue")

	for {
		transaction, err := s.TransactionsQueue.Pool()
		if err != nil {
			s.log.WithField("err", err).Error("Error pooling transactions queue")
			time.Sleep(time.Second)
			continue
		}

		if transaction == nil {
			time.Sleep(time.Second)
			continue
		}

		s.log.WithField("transaction", transaction).Info("Received transaction from transactions queue")

		// Use Stellar Precision
		amount, err := transaction.AmountToEth(7)
		if err != nil {
			s.log.WithField("transaction", transaction).Error("Amount is invalid")
			continue
		}

		go s.StellarAccountConfigurator.ConfigureAccount(
			transaction.StellarPublicKey,
			string(transaction.AssetCode),
			amount,
		)
	}
}
