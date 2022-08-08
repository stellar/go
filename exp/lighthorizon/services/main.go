package services

import (
	"context"
	"io"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stellar/go/exp/lighthorizon/archive"
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

// searchCallback is a generic way for any endpoint to process a transaction and
// its corresponding ledger. It should return whether or not we should stop
// processing (e.g. when a limit is reached) and any error that occurred.
type searchCallback func(archive.LedgerTransaction, *xdr.LedgerHeader) (finished bool, err error)

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
