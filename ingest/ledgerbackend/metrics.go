package ledgerbackend

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stellar/go/xdr"
)

// WithMetrics decorates the given LedgerBackend with metrics
func WithMetrics(base LedgerBackend, registry *prometheus.Registry, namespace string) LedgerBackend {
	if captiveCoreBackend, ok := base.(*CaptiveStellarCore); ok {
		captiveCoreBackend.registerCoreMetrics(registry, namespace)
	}
	summary := prometheus.NewSummary(
		prometheus.SummaryOpts{
			Namespace: namespace, Subsystem: "ingest", Name: "ledger_fetch_duration_seconds",
			Help: "duration of fetching ledgers from ledger backend, sliding window = 10m",
		},
	)
	registry.MustRegister(summary)
	return metricsLedgerBackend{
		LedgerBackend:              base,
		ledgerFetchDurationSummary: summary,
	}
}

type metricsLedgerBackend struct {
	LedgerBackend
	ledgerFetchDurationSummary prometheus.Summary
}

func (m metricsLedgerBackend) GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error) {
	startTime := time.Now()
	lcm, err := m.LedgerBackend.GetLedger(ctx, sequence)
	if err != nil {
		return xdr.LedgerCloseMeta{}, err
	}
	m.ledgerFetchDurationSummary.Observe(time.Since(startTime).Seconds())
	return lcm, nil
}
