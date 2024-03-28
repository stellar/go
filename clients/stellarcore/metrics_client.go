package stellarcore

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	proto "github.com/stellar/go/protocols/stellarcore"
	"github.com/stellar/go/xdr"
	"time"
)

type ClientWithMetrics interface {
	SubmitTransaction(ctx context.Context, rawTx string, envelope xdr.TransactionEnvelope) (resp *proto.TXResponse, err error)
}

type clientWithMetrics struct {
	CoreClient Client

	TxSubMetrics struct {
		// SubmissionDuration exposes timing metrics about the rate and latency of
		// submissions to stellar-core
		SubmissionDuration *prometheus.SummaryVec

		// SubmissionsCounter tracks the rate of transactions that have
		// been submitted to this process
		SubmissionsCounter *prometheus.CounterVec

		// V0TransactionsCounter tracks the rate of v0 transaction envelopes that
		// have been submitted to this process
		V0TransactionsCounter *prometheus.CounterVec

		// V1TransactionsCounter tracks the rate of v1 transaction envelopes that
		// have been submitted to this process
		V1TransactionsCounter *prometheus.CounterVec

		// FeeBumpTransactionsCounter tracks the rate of fee bump transaction envelopes that
		// have been submitted to this process
		FeeBumpTransactionsCounter *prometheus.CounterVec
	}
}

func (c *clientWithMetrics) SubmitTransaction(ctx context.Context, rawTx string, envelope xdr.TransactionEnvelope) (*proto.TXResponse, error) {
	startTime := time.Now()
	response, err := c.CoreClient.SubmitTransaction(ctx, rawTx)
	c.updateTxSubMetrics(time.Since(startTime).Seconds(), envelope, response, err)

	return response, err
}

func (c *clientWithMetrics) updateTxSubMetrics(duration float64, envelope xdr.TransactionEnvelope, response *proto.TXResponse, err error) {
	var label prometheus.Labels
	if err != nil {
		label = prometheus.Labels{"status": "request_error"}
	} else if response.IsException() {
		label = prometheus.Labels{"status": "exception"}
	} else {
		label = prometheus.Labels{"status": response.Status}
	}

	c.TxSubMetrics.SubmissionDuration.With(label).Observe(duration)
	c.TxSubMetrics.SubmissionsCounter.With(label).Inc()

	switch envelope.Type {
	case xdr.EnvelopeTypeEnvelopeTypeTxV0:
		c.TxSubMetrics.V0TransactionsCounter.With(label).Inc()
	case xdr.EnvelopeTypeEnvelopeTypeTx:
		c.TxSubMetrics.V1TransactionsCounter.With(label).Inc()
	case xdr.EnvelopeTypeEnvelopeTypeTxFeeBump:
		c.TxSubMetrics.FeeBumpTransactionsCounter.With(label).Inc()
	}
}

func NewClientWithMetrics(client Client, registry *prometheus.Registry, prometheusSubsystem string) ClientWithMetrics {
	submissionDuration := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "horizon",
		Subsystem:  prometheusSubsystem,
		Name:       "submission_duration_seconds",
		Help:       "submission durations to Stellar-Core, sliding window = 10m",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"status"})
	submissionsCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "horizon",
		Subsystem: prometheusSubsystem,
		Name:      "submissions_count",
		Help:      "number of submissions, sliding window = 10m",
	}, []string{"status"})
	v0TransactionsCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "horizon",
		Subsystem: prometheusSubsystem,
		Name:      "v0_count",
		Help:      "number of v0 transaction envelopes submitted, sliding window = 10m",
	}, []string{"status"})
	v1TransactionsCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "horizon",
		Subsystem: prometheusSubsystem,
		Name:      "v1_count",
		Help:      "number of v1 transaction envelopes submitted, sliding window = 10m",
	}, []string{"status"})
	feeBumpTransactionsCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "horizon",
		Subsystem: prometheusSubsystem,
		Name:      "feebump_count",
		Help:      "number of fee bump transaction envelopes submitted, sliding window = 10m",
	}, []string{"status"})

	registry.MustRegister(
		submissionDuration,
		submissionsCounter,
		v0TransactionsCounter,
		v1TransactionsCounter,
		feeBumpTransactionsCounter,
	)

	return &clientWithMetrics{
		CoreClient: client,
		TxSubMetrics: struct {
			SubmissionDuration         *prometheus.SummaryVec
			SubmissionsCounter         *prometheus.CounterVec
			V0TransactionsCounter      *prometheus.CounterVec
			V1TransactionsCounter      *prometheus.CounterVec
			FeeBumpTransactionsCounter *prometheus.CounterVec
		}{
			SubmissionDuration:         submissionDuration,
			SubmissionsCounter:         submissionsCounter,
			V0TransactionsCounter:      v0TransactionsCounter,
			V1TransactionsCounter:      v1TransactionsCounter,
			FeeBumpTransactionsCounter: feeBumpTransactionsCounter,
		},
	}
}
