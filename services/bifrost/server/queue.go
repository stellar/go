package server

import (
	"time"

	"github.com/stellar/go/services/bifrost/queue"
)

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
		var amount string
		switch transaction.AssetCode {
		case queue.AssetCodeBTC:
			amount, err = transaction.AmountToBtc(7)
		case queue.AssetCodeETH:
			amount, err = transaction.AmountToEth(7)
		default:
			s.log.Error("Invalid asset code pooled from the queue")
			continue
		}

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
