package txsub

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

type HorizonDB interface {
	GetLatestHistoryLedger(ctx context.Context) (uint32, error)
	PreFilteredTransactionByHash(ctx context.Context, dest interface{}, hash string) error
	TransactionByHash(ctx context.Context, dest interface{}, hash string) error
	AllTransactionsByHashesSinceLedger(ctx context.Context, hashes []string, sinceLedgerSeq uint32) ([]history.Transaction, error)
	GetSequenceNumbers(ctx context.Context, addresses []string) (map[string]uint64, error)
	BeginTx(context.Context, *sql.TxOptions) error
	Rollback() error
	NoRows(error) bool
}

// System represents a completely configured transaction submission system.
// Its methods tie together the various pieces used to reliably submit transactions
// to a stellar-core instance.
type System struct {
	initializer sync.Once

	tickMutex      sync.Mutex
	tickInProgress bool

	accountSeqPollInterval time.Duration

	DB                func(context.Context) HorizonDB
	Submitter         Submitter
	SubmissionTimeout time.Duration
	Log               *log.Entry

	Metrics struct {
		// SubmissionDuration exposes timing metrics about the rate and latency of
		// submissions to stellar-core
		SubmissionDuration prometheus.Summary

		// FailedSubmissionsCounter tracks the rate of failed transactions that have
		// been submitted to this process
		FailedSubmissionsCounter prometheus.Counter

		// SuccessfulSubmissionsCounter tracks the rate of successful transactions that
		// have been submitted to this process
		SuccessfulSubmissionsCounter prometheus.Counter

		// V0TransactionsCounter tracks the rate of v0 transaction envelopes that
		// have been submitted to this process
		V0TransactionsCounter prometheus.Counter

		// V1TransactionsCounter tracks the rate of v1 transaction envelopes that
		// have been submitted to this process
		V1TransactionsCounter prometheus.Counter

		// FeeBumpTransactionsCounter tracks the rate of fee bump transaction envelopes that
		// have been submitted to this process
		FeeBumpTransactionsCounter prometheus.Counter
	}
}

// RegisterMetrics registers the prometheus metrics
func (sys *System) RegisterMetrics(registry *prometheus.Registry) {
	registry.MustRegister(sys.Metrics.SubmissionDuration)
	registry.MustRegister(sys.Metrics.FailedSubmissionsCounter)
	registry.MustRegister(sys.Metrics.SuccessfulSubmissionsCounter)
	registry.MustRegister(sys.Metrics.V0TransactionsCounter)
	registry.MustRegister(sys.Metrics.V1TransactionsCounter)
	registry.MustRegister(sys.Metrics.FeeBumpTransactionsCounter)
}

// Submit submits the provided base64 encoded transaction envelope to the
// network using this submission system.
func (sys *System) Submit(
	ctx context.Context,
	rawTx string,
	envelope xdr.TransactionEnvelope,
	hash string,
) (resultReadCh <-chan Result) {
	sys.Init()
	resultCh := make(chan Result, 1)
	resultReadCh = resultCh

	db := sys.DB(ctx)
	// The database doesn't (yet) store muxed accounts, so we query
	// the corresponding AccountId
	sourceAccount := envelope.SourceAccount().ToAccountId()
	sourceAddress := sourceAccount.Address()

	sys.Log.Ctx(ctx).WithFields(log.F{
		"hash":    hash,
		"tx_type": envelope.Type.String(),
		"tx":      rawTx,
	}).Info("Processing transaction")

	seqNum := envelope.SeqNum()
	minSeqNum := envelope.MinSeqNum()
	// Ensure sequence numbers make sense
	if seqNum < 0 || (minSeqNum != nil && (*minSeqNum < 0 || *minSeqNum >= seqNum)) {
		sys.finish(ctx, hash, resultCh, Result{Err: ErrBadSequence})
		return
	}

	tx, _, err := checkTxAlreadyExists(ctx, db, hash, sourceAddress)
	if err == nil {
		sys.Log.Ctx(ctx).WithField("hash", hash).Info("Found submission result in a DB")
		sys.finish(ctx, hash, resultCh, Result{Transaction: tx})
		return
	}
	if err != ErrNoResults {
		sys.Log.Ctx(ctx).WithField("hash", hash).Info("Error getting submission result from a DB")
		sys.finish(ctx, hash, resultCh, Result{Transaction: tx, Err: err})
		return
	}

	sr := sys.submitOnce(ctx, rawTx)
	sys.updateTransactionTypeMetrics(envelope)

	if sr.Err != nil {
		// any error other than "txBAD_SEQ" is a failure
		isBad, err := sr.IsBadSeq()
		if err != nil {
			sys.finish(ctx, hash, resultCh, Result{Err: err})
			return
		}
		if !isBad {
			sys.finish(ctx, hash, resultCh, Result{Err: sr.Err})
			return
		}

		if err = sys.waitUntilAccountSequence(ctx, db, sourceAddress, uint64(envelope.SeqNum())); err != nil {
			sys.finish(ctx, hash, resultCh, Result{Err: err})
			return
		}

		// If error is txBAD_SEQ, check for the result again
		tx, err = txResultByHash(ctx, db, hash)
		if err != nil {
			// finally, return the bad_seq error if no result was found on 2nd attempt
			sys.finish(ctx, hash, resultCh, Result{Err: sr.Err})
			return
		}
		// If we found the result, use it as the result
		sys.finish(ctx, hash, resultCh, Result{Transaction: tx})
		return
	}

	return
}

// waitUntilAccountSequence blocks until either the context times out or the sequence number of the
// given source account is greater than or equal to `seq`
func (sys *System) waitUntilAccountSequence(ctx context.Context, db HorizonDB, sourceAddress string, seq uint64) error {
	timer := time.NewTimer(sys.accountSeqPollInterval)
	defer timer.Stop()

	for {
		sequenceNumbers, err := db.GetSequenceNumbers(ctx, []string{sourceAddress})
		if err != nil {
			sys.Log.Ctx(ctx).
				WithError(err).
				WithField("sourceAddress", sourceAddress).
				Warn("cannot fetch sequence number")
		} else {
			num, ok := sequenceNumbers[sourceAddress]
			if !ok {
				sys.Log.Ctx(ctx).
					WithField("sequenceNumbers", sequenceNumbers).
					WithField("sourceAddress", sourceAddress).
					Warn("missing sequence number for account")
			}
			if num >= seq {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return sys.deriveTxSubError(ctx)
		case <-timer.C:
			timer.Reset(sys.accountSeqPollInterval)
		}
	}
}

func (sys *System) deriveTxSubError(ctx context.Context) error {
	if ctx.Err() == context.Canceled {
		return ErrCanceled
	}
	return ErrTimeout
}

// Submit submits the provided base64 encoded transaction envelope to the
// network using this submission system.
func (sys *System) submitOnce(ctx context.Context, env string) SubmissionResult {
	// submit to stellar-core
	sr := sys.Submitter.Submit(ctx, env)
	sys.Metrics.SubmissionDuration.Observe(float64(sr.Duration.Seconds()))

	// Update submission metrics
	if sr.Err == nil {
		sys.Metrics.SuccessfulSubmissionsCounter.Inc()
	} else {
		sys.Metrics.FailedSubmissionsCounter.Inc()
	}

	return sr
}

func (sys *System) updateTransactionTypeMetrics(envelope xdr.TransactionEnvelope) {
	switch envelope.Type {
	case xdr.EnvelopeTypeEnvelopeTypeTxV0:
		sys.Metrics.V0TransactionsCounter.Inc()
	case xdr.EnvelopeTypeEnvelopeTypeTx:
		sys.Metrics.V1TransactionsCounter.Inc()
	case xdr.EnvelopeTypeEnvelopeTypeTxFeeBump:
		sys.Metrics.FeeBumpTransactionsCounter.Inc()
	}
}

// setTickInProgress sets `tickInProgress` to `true` if it's
// `false`. Returns `true` if `tickInProgress` has been switched
// to `true` inside this method and `Tick()` should continue.
func (sys *System) setTickInProgress(ctx context.Context) bool {
	sys.tickMutex.Lock()
	defer sys.tickMutex.Unlock()

	if sys.tickInProgress {
		logger := log.Ctx(ctx)
		logger.Info("ticking in progress")
		return false
	}

	sys.tickInProgress = true
	return true
}

func (sys *System) unsetTickInProgress() {
	sys.tickMutex.Lock()
	defer sys.tickMutex.Unlock()
	sys.tickInProgress = false
}

// Init initializes `sys`
func (sys *System) Init() {
	sys.initializer.Do(func() {
		sys.Log = log.DefaultLogger.WithField("service", "txsub.System")

		sys.Metrics.SubmissionDuration = prometheus.NewSummary(prometheus.SummaryOpts{
			Namespace: "horizon", Subsystem: "txsub", Name: "submission_duration_seconds",
			Help: "submission durations to Stellar-Core, sliding window = 10m",
		})
		sys.Metrics.FailedSubmissionsCounter = prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "horizon", Subsystem: "txsub", Name: "failed",
		})
		sys.Metrics.SuccessfulSubmissionsCounter = prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "horizon", Subsystem: "txsub", Name: "succeeded",
		})
		sys.Metrics.V0TransactionsCounter = prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "horizon", Subsystem: "txsub", Name: "v0",
		})
		sys.Metrics.V1TransactionsCounter = prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "horizon", Subsystem: "txsub", Name: "v1",
		})
		sys.Metrics.FeeBumpTransactionsCounter = prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "horizon", Subsystem: "txsub", Name: "feebump",
		})

		sys.accountSeqPollInterval = time.Second

		if sys.SubmissionTimeout == 0 {
			// HTTP clients in SDKs usually timeout in 60 seconds. We want SubmissionTimeout
			// to be lower than that to make sure that they read the response before the client
			// timeout.
			// 30 seconds is 6 ledgers (with avg. close time = 5 sec), enough for stellar-core
			// to drop the transaction if not added to the ledger and ask client to try again
			// by sending a Timeout response.
			sys.SubmissionTimeout = 30 * time.Second
		}
	})
}

func (sys *System) finish(ctx context.Context, hash string, response chan<- Result, r Result) {
	sys.Log.Ctx(ctx).
		WithField("result", fmt.Sprintf("%+v", r)).
		WithField("hash", hash).
		Info("Submission system result")
	response <- r
	close(response)
}
