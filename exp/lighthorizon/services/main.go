package services

import (
	"context"
	"io"
	"strconv"
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

// NewMetrics returns a Metrics instance containing all the prometheus
// metrics necessary for running light horizon services.
func NewMetrics(registry *prometheus.Registry) Metrics {
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
	return Metrics{
		ResponseAgeHistogram: responseAgeHistogram,
	}
}

type LightHorizon struct {
	Operations   OperationsService
	Transactions TransactionsService
}

type Metrics struct {
	ResponseAgeHistogram *prometheus.HistogramVec
}

type Config struct {
	Archive    archive.Archive
	IndexStore index.Store
	Passphrase string
	Metrics    Metrics
}

type TransactionsService struct {
	TransactionRepository
	Config Config
}

type OperationsService struct {
	OperationsRepository
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

	oldest := ops[0].LedgerHeader.ScpValue.CloseTime
	for i := 1; i < len(ops); i++ {
		if closeTime := ops[i].LedgerHeader.ScpValue.CloseTime; closeTime < oldest {
			oldest = closeTime
		}
	}

	lastCloseTime := time.Unix(int64(oldest), 0).UTC()
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

	err := searchAccountTransactions(ctx, cursor, accountId, os.Config, opsCallback)
	if age := operationsResponseAgeSeconds(ops); age >= 0 {
		os.Config.Metrics.ResponseAgeHistogram.With(prometheus.Labels{
			"request":    "GetOperationsByAccount",
			"successful": strconv.FormatBool(err == nil),
		}).Observe(age)
	}

	return ops, err
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

	err := searchAccountTransactions(ctx, cursor, accountId, ts.Config, txsCallback)
	if age := transactionsResponseAgeSeconds(txs); age >= 0 {
		ts.Config.Metrics.ResponseAgeHistogram.With(prometheus.Labels{
			"request":    "GetTransactionsByAccount",
			"successful": strconv.FormatBool(err == nil),
		}).Observe(age)
	}

	return txs, err
}

func searchAccountTransactions(ctx context.Context,
	cursor int64,
	accountId string,
	config Config,
	callback searchCallback,
) error {
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

	fullStart := time.Now()
	avgFetchDuration := time.Duration(0)
	avgProcessDuration := time.Duration(0)
	avgIndexFetchDuration := time.Duration(0)
	count := int64(0)

	defer func() {
		log.WithField("ledgers", count).
			WithField("ledger-fetch", avgFetchDuration.String()).
			WithField("ledger-process", avgProcessDuration.String()).
			WithField("index-fetch", avgIndexFetchDuration.String()).
			WithField("total", time.Since(fullStart)).
			Infof("Fulfilled request for account %s at cursor %d", accountId, cursor)
	}()

	for {
		count++
		start := time.Now()
		ledger, ledgerErr := config.Archive.GetLedger(ctx, nextLedger)
		if ledgerErr != nil {
			return errors.Wrapf(ledgerErr,
				"ledger export state is out of sync at ledger %d", nextLedger)
		}
		fetchDuration := time.Since(start)
		if fetchDuration > time.Second {
			log.WithField("duration", fetchDuration).
				Warnf("Fetching ledger %d was really slow", nextLedger)
		}
		incrementAverage(&avgFetchDuration, fetchDuration, count)

		start = time.Now()
		reader, readerErr := config.Archive.NewLedgerTransactionReaderFromLedgerCloseMeta(config.Passphrase, ledger)
		if readerErr != nil {
			return readerErr
		}

		for {
			tx, readErr := reader.Read()
			if readErr == io.EOF {
				break
			} else if readErr != nil {
				return readErr
			}

			// Note: If we move to ledger-based indices, we don't need this,
			// since we have a guarantee that the transaction will contain the
			// account as a participant.
			participants, participantErr := config.Archive.GetTransactionParticipants(tx)
			if participantErr != nil {
				return participantErr
			}

			if _, found := participants[accountId]; found {
				finished, callBackErr := callback(tx, &ledger.V0.LedgerHeader.Header)
				if callBackErr != nil {
					return callBackErr
				} else if finished {
					incrementAverage(&avgProcessDuration, time.Since(start), count)
					return nil
				}
			}

			if ctx.Err() != nil {
				return ctx.Err()
			}
		}

		incrementAverage(&avgProcessDuration, time.Since(start), count)

		start = time.Now()
		cursor, err = cursorMgr.Advance()
		if err != nil && err != io.EOF {
			return err
		}

		nextLedger = getLedgerFromCursor(cursor)
		incrementAverage(&avgIndexFetchDuration, time.Since(start), count)
		if err == io.EOF {
			return nil
		}
	}
}

// This calculates the average by incorporating a new value into an existing
// average in place. Note that `newCount` should represent the *new* total
// number of values incorporated into the average.
//
// Reference: https://math.stackexchange.com/a/106720
func incrementAverage(prevAverage *time.Duration, latest time.Duration, newCount int64) {
	increment := int64(latest-*prevAverage) / newCount
	*prevAverage = *prevAverage + time.Duration(increment)
}
