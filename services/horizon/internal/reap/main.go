// Package reap contains the history reaping subsystem for horizon.  This system
// is designed to remove data from the history database such that it does not
// grow indefinitely.  The system can be configured with a number of ledgers to
// maintain at a minimum.
package reap

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/db"
)

// Reaper represents the history reaping subsystem of horizon.
type Reaper struct {
	historyQ *history.Q
	config   Config

	deleteBatchDuration prometheus.Summary
	rowsDeleted         prometheus.Summary

	lock sync.Mutex
}

type Config struct {
	RetentionCount uint32
	ReapBatchSize  uint32
}

// New initializes the reaper, causing it to begin polling the stellar-core
// database for now ledgers and ingesting data into the horizon database.
func New(config Config, dbSession db.SessionInterface) *Reaper {
	r := &Reaper{
		historyQ: &history.Q{dbSession.Clone()},
		config:   config,
		deleteBatchDuration: prometheus.NewSummary(prometheus.SummaryOpts{
			Namespace: "horizon", Subsystem: "reap", Name: "delete_batch_duration",
			Help:       "reap batch duration in seconds, sliding window = 10m",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}),
		rowsDeleted: prometheus.NewSummary(prometheus.SummaryOpts{
			Namespace: "horizon", Subsystem: "reap", Name: "rows_deleted",
			Help:       "rows deleted during reap batch , sliding window = 10m",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}),
	}

	return r
}
