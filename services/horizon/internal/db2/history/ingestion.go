package history

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/services/horizon/internal/toid"
)

// TruncateExpingestStateTables clears out ingestion state tables.
// Ingestion state tables are horizon database tables populated by
// the experimental ingestion system using history archive snapshots.
// Any horizon database tables which cannot be populated using
// history archive snapshots will not be truncated.
func (q *Q) TruncateExpingestStateTables() error {
	return q.TruncateTables([]string{
		"accounts",
		"accounts_data",
		"accounts_signers",
		"exp_asset_stats",
		"offers",
		"trust_lines",
	})
}

// ExpIngestRemovalSummary describes how many rows in the experimental ingestion
// history tables have been deleted by RemoveExpIngestHistory()
type ExpIngestRemovalSummary struct {
	LedgersRemoved                 int64
	TransactionsRemoved            int64
	TransactionParticipantsRemoved int64
	OperationsRemoved              int64
	OperationParticipantsRemoved   int64
	TradesRemoved                  int64
	EffectsRemoved                 int64
}

// RemoveExpIngestHistory removes all rows in the experimental ingestion
// history tables which have a ledger sequence higher than `newerThanSequence`
func (q *Q) RemoveExpIngestHistory(newerThanSequence uint32) (ExpIngestRemovalSummary, error) {
	summary := ExpIngestRemovalSummary{}

	result, err := q.Exec(
		sq.Delete("exp_history_ledgers").
			Where("sequence > ?", newerThanSequence),
	)
	if err != nil {
		return summary, err
	}

	summary.LedgersRemoved, err = result.RowsAffected()
	if err != nil {
		return summary, err
	}

	result, err = q.Exec(
		sq.Delete("exp_history_transactions").
			Where("ledger_sequence > ?", newerThanSequence),
	)
	if err != nil {
		return summary, err
	}

	summary.TransactionsRemoved, err = result.RowsAffected()
	if err != nil {
		return summary, err
	}

	result, err = q.Exec(
		sq.Delete("exp_history_transaction_participants").
			Where("history_transaction_id >= ?", toid.ID{LedgerSequence: int32(newerThanSequence + 1)}.ToInt64()),
	)
	if err != nil {
		return summary, err
	}

	summary.TransactionParticipantsRemoved, err = result.RowsAffected()
	if err != nil {
		return summary, err
	}

	result, err = q.Exec(
		sq.Delete("exp_history_operations").
			Where("id >= ?", toid.ID{LedgerSequence: int32(newerThanSequence + 1)}.ToInt64()),
	)
	if err != nil {
		return summary, err
	}

	summary.OperationsRemoved, err = result.RowsAffected()
	if err != nil {
		return summary, err
	}

	result, err = q.Exec(
		sq.Delete("exp_history_operation_participants").
			Where("history_operation_id >= ?", toid.ID{LedgerSequence: int32(newerThanSequence + 1)}.ToInt64()),
	)
	if err != nil {
		return summary, err
	}

	summary.OperationParticipantsRemoved, err = result.RowsAffected()
	if err != nil {
		return summary, err
	}

	result, err = q.Exec(
		sq.Delete("exp_history_trades").
			Where("history_operation_id >= ?", toid.ID{LedgerSequence: int32(newerThanSequence + 1)}.ToInt64()),
	)
	if err != nil {
		return summary, err
	}

	summary.TradesRemoved, err = result.RowsAffected()
	if err != nil {
		return summary, err
	}

	result, err = q.Exec(
		sq.Delete("exp_history_effects").
			Where("history_operation_id >= ?", toid.ID{LedgerSequence: int32(newerThanSequence + 1)}.ToInt64()),
	)
	if err != nil {
		return summary, err
	}

	summary.EffectsRemoved, err = result.RowsAffected()

	return summary, err
}
