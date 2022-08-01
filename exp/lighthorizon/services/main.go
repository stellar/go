package services

import (
	"context"
	"io"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/xdr"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

const (
	allTransactionsIndex = "all/all"
	allPaymentsIndex     = "all/payments"
)

var (
	checkpointManager = historyarchive.NewCheckpointManager(0)
)

// NewResponseAgeHistogramMetric creates a new prometheus histogram metric to measure
// the age of response for a service.
func NewResponseAgeHistogramMetric(registry *prometheus.Registry) *prometheus.HistogramVec {
	const minute = 60
	const day = 24 * 60 * minute
	responseAgeHistogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "horizon_lite",
		Subsystem: "services",
		Name:      "response_age",
		Buckets: []float64{
			5 * minute,
			60 * minute,
			day,
			7 * day,
			30 * day,
			90 * day,
			180 * day,
			365 * day,
		},
		Help: "Age of the response for each service, sliding window = 10m",
	},
		[]string{"request", "successful"},
	)
	registry.MustRegister(responseAgeHistogram)
	return responseAgeHistogram
}

type LightHorizon struct {
	Operations   OperationsService
	Transactions TransactionsService
}

type Config struct {
	Archive              archive.Archive
	IndexStore           index.Store
	Passphrase           string
	ResponseAgeHistogram *prometheus.HistogramVec
}

type TransactionsService struct {
	TransactionRepository
	Config Config
}

type OperationsService struct {
	OperationsRepository,
	Config Config
}

type OperationsRepository interface {
	GetOperationsByAccount(ctx context.Context, cursor int64, limit uint64, accountId string) ([]common.Operation, error)
}

type TransactionRepository interface {
	GetTransactionsByAccount(ctx context.Context, cursor int64, limit uint64, accountId string) ([]common.Transaction, error)
}

// searchCallback is a generic way for any endpoint to process a transaction and
// its corresponding ledger. It should return whether or not we should stop
// processing (e.g. when a limit is reached) and any error that occurred.
type searchCallback func(archive.LedgerTransaction, *xdr.LedgerHeader) (finished bool, err error)

func operationsResponseAgeSeconds(ops []common.Operation) float64 {
	if len(ops) == 0 {
		return -1
	}
	lastCloseTime := time.Unix(int64(ops[len(ops)-1].LedgerHeader.ScpValue.CloseTime), 0).UTC()
	now := time.Now().UTC()
	if now.Before(lastCloseTime) {
		log.Errorf("current time %v is before oldest operation close time %v", now, lastCloseTime)
		return -1
	}
	return now.Sub(lastCloseTime).Seconds()
}

func (os *OperationsService) GetOperationsByAccount(ctx context.Context,
	cursor int64, limit uint64,
	accountId string,
) ([]common.Operation, error) {
	ops := []common.Operation{}

	opsCallback := func(tx archive.LedgerTransaction, ledgerHeader *xdr.LedgerHeader) (bool, error) {
		for operationOrder, op := range tx.Envelope.Operations() {
			opParticipants, err := os.Config.Archive.GetOperationParticipants(tx, op, operationOrder)
			if err != nil {
				return false, err
			}

			if _, foundInOp := opParticipants[accountId]; foundInOp {
				ops = append(ops, common.Operation{
					TransactionEnvelope: &tx.Envelope,
					TransactionResult:   &tx.Result.Result,
					LedgerHeader:        ledgerHeader,
					TxIndex:             int32(tx.Index),
					OpIndex:             int32(operationOrder),
				})

				if uint64(len(ops)) == limit {
					return true, nil
				}
			}
		}

		return false, nil
	}

	if err := searchTxByAccount(ctx, cursor, accountId, os.Config, opsCallback); err != nil {
		if age := operationsResponseAgeSeconds(ops); age >= 0 {
			os.Config.ResponseAgeHistogram.With(prometheus.Labels{
				"request":    "GetOperationsByAccount",
				"successful": "false",
			}).Observe(age)
		}
		return nil, err
	}

	if age := operationsResponseAgeSeconds(ops); age >= 0 {
		os.Config.ResponseAgeHistogram.With(prometheus.Labels{
			"request":    "GetOperationsByAccount",
			"successful": "true",
		}).Observe(age)
	}
	return ops, nil
}

func transactionsResponseAgeSeconds(txs []common.Transaction) float64 {
	if len(txs) == 0 {
		return -1
	}
	lastCloseTime := time.Unix(int64(txs[len(txs)-1].LedgerHeader.ScpValue.CloseTime), 0).UTC()
	now := time.Now().UTC()
	if now.Before(lastCloseTime) {
		log.Errorf("current time %v is before oldest transaction close time %v", now, lastCloseTime)
		return -1
	}
	return now.Sub(lastCloseTime).Seconds()
}

func (ts *TransactionsService) GetTransactionsByAccount(ctx context.Context,
	cursor int64, limit uint64,
	accountId string,
) ([]common.Transaction, error) {
	txs := []common.Transaction{}

	txsCallback := func(tx archive.LedgerTransaction, ledgerHeader *xdr.LedgerHeader) (bool, error) {
		txs = append(txs, common.Transaction{
			TransactionEnvelope: &tx.Envelope,
			TransactionResult:   &tx.Result.Result,
			LedgerHeader:        ledgerHeader,
			TxIndex:             int32(tx.Index),
			NetworkPassphrase:   ts.Config.Passphrase,
		})

		return uint64(len(txs)) == limit, nil
	}

	if err := searchTxByAccount(ctx, cursor, accountId, ts.Config, txsCallback); err != nil {
		if age := transactionsResponseAgeSeconds(txs); age >= 0 {
			ts.Config.ResponseAgeHistogram.With(prometheus.Labels{
				"request":    "GetTransactionsByAccount",
				"successful": "false",
			}).Observe(age)
		}
		return nil, err
	}

	if age := transactionsResponseAgeSeconds(txs); age >= 0 {
		ts.Config.ResponseAgeHistogram.With(prometheus.Labels{
			"request":    "GetTransactionsByAccount",
			"successful": "true",
		}).Observe(age)
	}
	return txs, nil
}

func searchTxByAccount(ctx context.Context, cursor int64, accountId string, config Config, callback searchCallback) error {
	cursorMgr := NewCursorManagerForAccountActivity(config.IndexStore, accountId)
	cursor, err := cursorMgr.Begin(cursor)
	if err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}

	nextLedger := getLedgerFromCursor(cursor)
	log.Debugf("Searching %s for account %s starting at ledger %d",
		allTransactionsIndex, accountId, nextLedger)

	for {
		ledger, ledgerErr := config.Archive.GetLedger(ctx, nextLedger)
		if ledgerErr != nil {
			return errors.Wrapf(ledgerErr,
				"ledger export state is out of sync at ledger %d", nextLedger)
		}

		reader, readerErr := config.Archive.NewLedgerTransactionReaderFromLedgerCloseMeta(config.Passphrase, ledger)
		if readerErr != nil {
			return readerErr
		}

		for {
			tx, readErr := reader.Read()
			if readErr != nil {
				if readErr == io.EOF {
					break
				}
				return readErr
			}

			participants, participantErr := config.Archive.GetTransactionParticipants(tx)
			if participantErr != nil {
				return participantErr
			}

			if _, found := participants[accountId]; found {
				finished, callBackErr := callback(tx, &ledger.V0.LedgerHeader.Header)
				if finished || callBackErr != nil {
					return callBackErr
				}
			}

			if ctx.Err() != nil {
				return ctx.Err()
			}
		}

		cursor, err = cursorMgr.Advance()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		nextLedger = getLedgerFromCursor(cursor)
	}
}
