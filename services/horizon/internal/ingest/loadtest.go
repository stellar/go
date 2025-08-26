package ingest

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/stellar/go/services/horizon/internal/db2/history"
)

type LoadTestSnapshot struct {
	HistoryQ history.IngestionQ
	runId    string
}

func (l *LoadTestSnapshot) CheckPendingLoadTest(ctx context.Context) error {
	if runID, _, err := l.HistoryQ.GetLoadTestRestoreState(ctx); errors.Is(err, sql.ErrNoRows) {
		if l.runId != "" {
			return fmt.Errorf("expected load test to be active with run id: %s", l.runId)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("Error getting load test restore state: %w", err)
	} else if runID != l.runId {
		return fmt.Errorf("load test run id is %s, expected: %s", runID, l.runId)
	}
	return nil
}

func (l *LoadTestSnapshot) Save(ctx context.Context) error {
	if err := l.HistoryQ.Begin(ctx); err != nil {
		return fmt.Errorf("Error starting a transaction: %w", err)
	}
	defer l.HistoryQ.Rollback()
	if l.runId != "" {
		return fmt.Errorf("load test already active, run id: %s", l.runId)
	}

	// This will get the value `FOR UPDATE`, blocking it for other nodes.
	lastIngestedLedger, err := l.HistoryQ.GetLastLedgerIngest(ctx)
	if err != nil {
		return fmt.Errorf("Error getting last ledger ingested: %w", err)
	}

	runID, restoreLedger, err := l.HistoryQ.GetLoadTestRestoreState(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		// No active load test state; create one with a random runID
		buf := make([]byte, 16)
		if _, err = rand.Read(buf); err != nil {
			return fmt.Errorf("Error generating runID: %w", err)
		}
		runID = hex.EncodeToString(buf)
		if err = l.HistoryQ.SetLoadTestRestoreState(ctx, runID, lastIngestedLedger); err != nil {
			return fmt.Errorf("Error setting load test restore state: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("Error getting load test restore state: %w", err)
	} else {
		return fmt.Errorf("load test already active, restore ledger: %d, run id: %s", restoreLedger, runID)
	}

	if err = l.HistoryQ.Commit(); err != nil {
		return fmt.Errorf("Error committing a transaction: %w", err)
	}
	l.runId = runID
	return nil
}

func (l *LoadTestSnapshot) Restore(ctx context.Context) error {
	if err := l.HistoryQ.Begin(ctx); err != nil {
		return fmt.Errorf("Error starting a transaction: %w", err)
	}
	defer l.HistoryQ.Rollback()

	// This will get the value `FOR UPDATE`, blocking it for other nodes.
	lastIngestedLedger, err := l.HistoryQ.GetLastLedgerIngest(ctx)
	if err != nil {
		return fmt.Errorf("Error getting last ledger ingested: %w", err)
	}

	_, restoreLedger, err := l.HistoryQ.GetLoadTestRestoreState(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	} else if err != nil {
		return fmt.Errorf("Error getting load test restore ledger: %w", err)
	}

	if restoreLedger > lastIngestedLedger {
		return fmt.Errorf("load test restore ledger: %d is greater than last ingested ledger: %d", restoreLedger, lastIngestedLedger)
	}

	if _, err = l.HistoryQ.DeleteRangeAll(ctx, int64(restoreLedger+1), int64(lastIngestedLedger)); err != nil {
		return fmt.Errorf("Error deleting range all: %w", err)
	}

	if err = l.HistoryQ.UpdateIngestVersion(ctx, 0); err != nil {
		return fmt.Errorf("Error updating ingestion version: %w", err)
	}

	if err = l.HistoryQ.ClearLoadTestRestoreState(ctx); err != nil {
		return fmt.Errorf("Error clearing load test restore ledger: %w", err)
	}

	if err = l.HistoryQ.Commit(); err != nil {
		return fmt.Errorf("Error committing a transaction: %w", err)
	}
	return nil
}
