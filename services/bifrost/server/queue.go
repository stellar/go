package server

import (
	"context"
	"time"

	"github.com/stellar/go/services/bifrost/database"
	"github.com/stellar/go/services/bifrost/queue"
)

var (
	processingTimeout      = database.DefaultTransactionLockTTL - 5*time.Second
	defaultQueueRetryDelay = time.Second
)

// poolTransactionsQueue pools transactions queue which contains only processed and
// validated transactions and sends it to StellarAccountConfigurator for account configuration.
func (s *Server) poolTransactionsQueue(ctx context.Context) {
	s.log.Info("Started pooling transactions queue")
	defer s.log.Info("Stopped pooling transactions queue")

	exit := ctx.Done()
	retryDelayer := time.NewTimer(defaultQueueRetryDelay)
	defer retryDelayer.Stop()

	for ctx.Err() == nil {
		switch empty, err := s.TransactionsQueue.IsEmpty(); {
		case err != nil:
			s.log.WithField("err", err).Error("failed to pull transaction queue")
		case !empty:
			go s.processNextQueuedTransaction(ctx)
			continue
		}

		retryDelayer.Reset(defaultQueueRetryDelay)
		select {
		case <-exit:
			return
		case <-retryDelayer.C:
			continue
		}
	}
}

func (s *Server) processNextQueuedTransaction(parentCtx context.Context) {
	err := s.TransactionsQueue.WithQueuedTransaction(func(transaction queue.Transaction) error {
		ctx, done := context.WithTimeout(parentCtx, processingTimeout)
		defer done()
		s.log.WithField("transaction", transaction).Info("Received transaction from transactions queue")
		return s.StellarAccountConfigurator.ConfigureAccount(
			ctx,
			transaction.TransactionID,
			transaction.StellarPublicKey,
			string(transaction.AssetCode),
			transaction.Amount,
		)
	})
	if err != nil {
		s.log.WithField("err", err).Error("Error processing transactions queue")
	}
}
