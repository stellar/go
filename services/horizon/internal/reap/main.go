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

// System represents the history reaping subsystem of horizon.
type System struct {
	historyQ       *history.Q
	RetentionCount uint32
	RetentionBatch uint32

	deleteBatchDuration prometheus.Summary
	rowsDeleted         prometheus.Summary

	lock sync.Mutex
}

// New initializes the reaper, causing it to begin polling the stellar-core
// database for now ledgers and ingesting data into the horizon database.
func New(retention, retentionBatchSize uint32, dbSession db.SessionInterface) *System {
	r := &System{
		historyQ:       &history.Q{dbSession.Clone()},
		RetentionCount: retention,
		RetentionBatch: retentionBatchSize,
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
