package services

import (
	"context"
	"io"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/constraints"

	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/exp/lighthorizon/ingester"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

const (
	allTransactionsIndex       = "all/all"
	allPaymentsIndex           = "all/payments"
	slowFetchDurationThreshold = time.Second
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
	Operations   OperationService
	Transactions TransactionService
}

type Metrics struct {
	ResponseAgeHistogram *prometheus.HistogramVec
}

type Config struct {
	Ingester   ingester.Ingester
	IndexStore index.Store
	Passphrase string
	Metrics    Metrics
}

// searchCallback is a generic way for any endpoint to process a transaction and
// its corresponding ledger. It should return whether or not we should stop
// processing (e.g. when a limit is reached) and any error that occurred.
type searchCallback func(ingester.LedgerTransaction, *xdr.LedgerHeader) (finished bool, err error)

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

	log.WithField("cursor", cursor).
		Debugf("Searching %s for account %s starting at ledger %d",
			allTransactionsIndex, accountId, nextLedger)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	fullStart := time.Now()
	fetchDuration := time.Duration(0)
	processDuration := time.Duration(0)
	indexFetchDuration := time.Duration(0)
	count := int64(0)

	defer func() {
		log.WithField("ledgers", count).
			WithField("ledger-fetch", fetchDuration).
			WithField("ledger-process", processDuration).
			WithField("index-fetch", indexFetchDuration).
			WithField("avg-ledger-fetch", getAverageDuration(fetchDuration, count)).
			WithField("avg-ledger-process", getAverageDuration(processDuration, count)).
			WithField("avg-index-fetch", getAverageDuration(indexFetchDuration, count)).
			WithField("total", time.Since(fullStart)).
			Infof("Fulfilled request for account %s at cursor %d", accountId, cursor)
	}()

	checkpointMgr := historyarchive.NewCheckpointManager(0)

	for {
		if checkpointMgr.IsCheckpoint(nextLedger) {
			r := historyarchive.Range{
				Low:  nextLedger,
				High: checkpointMgr.NextCheckpoint(nextLedger + 1),
			}
			log.Infof("Preparing ledger range [%d, %d]", r.Low, r.High)
			if innerErr := config.Ingester.PrepareRange(ctx, r); innerErr != nil {
				log.Errorf("failed to prepare ledger range [%d, %d]: %v",
					r.Low, r.High, innerErr)
			}
		}

		start := time.Now()
		ledger, innerErr := config.Ingester.GetLedger(ctx, nextLedger)

		// TODO: We should have helpful error messages when innerErr points to a
		// 404 for that particular ledger, since that situation shouldn't happen
		// under normal operations, but rather indicates a problem with the
		// backing archive.
		if innerErr != nil {
			return errors.Wrapf(innerErr,
				"failed to retrieve ledger %d from archive", nextLedger)
		}
		count++
		thisFetchDuration := time.Since(start)
		if thisFetchDuration > slowFetchDurationThreshold {
			log.WithField("duration", thisFetchDuration).
				Warnf("Fetching ledger %d was really slow", nextLedger)
		}
		fetchDuration += thisFetchDuration

		start = time.Now()
		reader, innerErr := config.Ingester.NewLedgerTransactionReader(ledger)
		if innerErr != nil {
			return errors.Wrapf(innerErr,
				"failed to read ledger %d", nextLedger)
		}

		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			tx, readErr := reader.Read()
			if readErr == io.EOF {
				break
			} else if readErr != nil {
				return readErr
			}

			// Note: If we move to ledger-based indices, we don't need this,
			// since we have a guarantee that the transaction will contain
			// the account as a participant.
			participants, participantErr := ingester.GetTransactionParticipants(tx)
			if participantErr != nil {
				return participantErr
			}

			if _, found := participants[accountId]; found {
				finished, callBackErr := callback(tx, &ledger.V0.V0.LedgerHeader.Header)
				if callBackErr != nil {
					return callBackErr
				} else if finished {
					processDuration += time.Since(start)
					return nil
				}
			}
		}

		processDuration += time.Since(start)
		start = time.Now()

		cursor, err = cursorMgr.Advance(1)
		if err != nil && err != io.EOF {
			return err
		}

		nextLedger = getLedgerFromCursor(cursor)
		indexFetchDuration += time.Since(start)
		if err == io.EOF {
			break
		}
	}

	return nil
}

func getAverageDuration[
	T constraints.Signed | constraints.Float,
](d time.Duration, count T) time.Duration {
	if count == 0 {
		return 0 // don't bomb on div-by-zero
	}
	return time.Duration(int64(float64(d.Nanoseconds()) / float64(count)))
}
