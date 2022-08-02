package services

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"

	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

const (
	allTransactionsIndex = "all/all"
	allPaymentsIndex     = "all/payments"
	maxParallelDownloads = 8
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

	log.WithField("cursor", cursor).
		Debugf("Searching %s for account %s starting at ledger %d",
			allTransactionsIndex, accountId, nextLedger)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

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
		start := time.Now()
		r := historyarchive.Range{
			Low:  nextLedger,
			High: checkpointManager.NextCheckpoint(nextLedger),
		}
		count += int64(1 + (r.High - r.Low))
		ledgerReader := make(chan xdr.LedgerCloseMeta, 1)
		wg := downloadLedgers(ctx, config.Archive,
			ledgerReader,
			r,
			maxParallelDownloads,
		)
		defer wg.Wait()

		if ctx.Err() != nil {
			return ctx.Err()
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
		}

		incrementAverage(&avgProcessDuration, time.Since(start), count)
		start = time.Now()

		// We just processed an entire checkpoint range, so we can fast forward
		// the cursor ahead to the next one.
		cursor, err = cursorMgr.Skip(64)
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

// downloadLedgers allows parallel downloads of ledgers via an archive. Give it
// a ledger range and a channel to which to output ledgers, and it will feed
// ledgers to it *in sequential order* while downloading up to
// `downloadWorkerCount` of them in parallel.
//
// It's the caller's responsibility to ensure that all of the goroutines in the
// returned group have completed. In contrast, this function closes the output
// channel when all work has been submitted (or the context errors).
//
// FIXME: Should this be a part of archive.Archive?
func downloadLedgers(
	ctx context.Context,
	ledgerArchive archive.Archive,
	outputChan chan<- xdr.LedgerCloseMeta,
	ledgerRange historyarchive.Range,
	downloadWorkerCount int,
) *errgroup.Group {
	start := time.Now()
	workQueue := make(chan uint32, downloadWorkerCount)

	// Alternatively, we can keep a `downloadWorkerCount`-sized buffer around
	// rather than the full count and then either do clever index manipulation
	// or a simple append+search to maintain sequential order.
	//
	// As it stands, this is only called w/ a checkpoint range, so keeping 64
	// txmetas in memory isn't a big deal.
	count := (ledgerRange.High - ledgerRange.Low) + 1
	ledgerFeed := make([]*xdr.LedgerCloseMeta, count)

	wg, ctx := errgroup.WithContext(ctx)

	// This work publisher adds ledger sequence numbers to the work queue.
	wg.Go(func() error {
		defer func() {
			log.WithField("duration", time.Since(start)).
				WithField("workers", downloadWorkerCount).
				WithError(ctx.Err()).
				Infof("Download of ledger range: [%d, %d] (%d ledgers) complete",
					ledgerRange.Low, ledgerRange.High, count)

			close(workQueue)
		}()

		for seq := ledgerRange.Low; seq <= ledgerRange.High; seq++ {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			workQueue <- seq
		}

		return nil
	})

	// This result publisher pushes txmetas to the output queue in order.
	lock := sync.Mutex{}
	cond := sync.NewCond(&lock)

	go func() {
		lastPublishedIdx := int64(-1)

		// Until the last ledger in the range has been published to the queue:
		//  - wait for the signal of a new ledger being available
		//  - ensure the ledger is sequential after the last one
		//  - increment and go again
		for lastPublishedIdx < int64(count)-1 {
			lock.Lock()
			before := lastPublishedIdx
			for ledgerFeed[before+1] == nil {
				cond.Wait()
			}

			// The signal might have triggered because there was a context
			// error, so check that first.
			if ctx.Err() != nil {
				break
			}

			outputChan <- *ledgerFeed[before+1]
			ledgerFeed[before+1] = nil // save memory
			lastPublishedIdx++
			lock.Unlock()
		}

		close(outputChan)
	}()

	// These are the workers that download & store ledgers in memory.
	for i := 0; i < downloadWorkerCount; i++ {
		wg.Go(func() error {
			for ledgerSeq := range workQueue {
				if ctx.Err() != nil { // timeout, cancel, etc.?
					cond.Signal() // signals publisher should stop
					return ctx.Err()
				}

				start := time.Now()
				txmeta, err := ledgerArchive.GetLedger(ctx, ledgerSeq)
				log.WithField("duration", time.Since(start)).
					Debugf("Downloaded ledger %d", ledgerSeq)
				if err != nil {
					return err
				}

				ledgerFeed[ledgerSeq-ledgerRange.Low] = &txmeta
				cond.Signal()
			}

			return nil
		})
	}

	return wg
}
