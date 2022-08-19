package services

import (
	"context"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

type TransactionRepository struct {
	TransactionService
	Config Config
}

type TransactionService interface {
	GetTransactionsByAccount(ctx context.Context,
		cursor int64, limit uint64,
		accountId string,
	) ([]common.Transaction, error)
}

func (tr *TransactionRepository) GetTransactionsByAccount(ctx context.Context,
	cursor int64, limit uint64,
	accountId string,
) ([]common.Transaction, error) {
	txs := []common.Transaction{}

	txsCallback := func(tx archive.LedgerTransaction, ledgerHeader *xdr.LedgerHeader) (bool, error) {
		txs = append(txs, common.Transaction{
			LedgerTransaction: &tx,
			LedgerHeader:      ledgerHeader,
			TxIndex:           int32(tx.Index),
			NetworkPassphrase: tr.Config.Passphrase,
		})

		return uint64(len(txs)) == limit, nil
	}

	err := searchAccountTransactions(ctx, cursor, accountId, tr.Config, txsCallback)
	if age := transactionsResponseAgeSeconds(txs); age >= 0 {
		tr.Config.Metrics.ResponseAgeHistogram.With(prometheus.Labels{
			"request":    "GetTransactionsByAccount",
			"successful": strconv.FormatBool(err == nil),
		}).Observe(age)
	}

	return txs, err
}

func transactionsResponseAgeSeconds(txs []common.Transaction) float64 {
	if len(txs) == 0 {
		return -1
	}

	oldest := txs[0].LedgerHeader.ScpValue.CloseTime
	for i := 1; i < len(txs); i++ {
		if closeTime := txs[i].LedgerHeader.ScpValue.CloseTime; closeTime < oldest {
			oldest = closeTime
		}
	}

	lastCloseTime := time.Unix(int64(oldest), 0).UTC()
	now := time.Now().UTC()
	if now.Before(lastCloseTime) {
		log.Errorf("current time %v is before oldest transaction close time %v", now, lastCloseTime)
		return -1
	}
	return now.Sub(lastCloseTime).Seconds()
}

var _ TransactionService = (*TransactionRepository)(nil) // ensure conformity to the interface
