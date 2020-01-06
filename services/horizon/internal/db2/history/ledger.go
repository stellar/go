package history

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LedgerBySequence loads the single ledger at `seq` into `dest`
func (q *Q) LedgerBySequence(dest interface{}, seq int32) error {
	sql := selectLedger.
		Limit(1).
		Where("sequence = ?", seq)

	return q.Get(dest, sql)
}

// expLedgerBySequence returns a row from the exp_history_ledgers table
func (q *Q) expLedgerBySequence(seq int32) (Ledger, error) {
	sql := selectLedgerFields.
		From("exp_history_ledgers hl").
		Limit(1).
		Where("sequence = ?", seq)

	var dest Ledger
	err := q.Get(&dest, sql)
	return dest, err
}

// Ledgers provides a helper to filter rows from the `history_ledgers` table
// with pre-defined filters.  See `LedgersQ` methods for the available filters.
func (q *Q) Ledgers() *LedgersQ {
	return &LedgersQ{
		parent: q,
		sql:    selectLedger,
	}
}

// LedgersBySequence loads the a set of ledgers identified by the sequences
// `seqs` into `dest`.
func (q *Q) LedgersBySequence(dest interface{}, seqs ...int32) error {
	if len(seqs) == 0 {
		return errors.New("no sequence arguments provided")
	}
	in := fmt.Sprintf("sequence IN (%s)", sq.Placeholders(len(seqs)))

	whereArgs := make([]interface{}, len(seqs))
	for i, s := range seqs {
		whereArgs[i] = s
	}

	sql := selectLedger.Where(in, whereArgs...)

	return q.Select(dest, sql)
}

// LedgerCapacityUsageStats returns ledger capacity stats for the last 5 ledgers.
// Currently, we hard code the query to return the last 5 ledgers.
// TODO: make the number of ledgers configurable.
func (q *Q) LedgerCapacityUsageStats(currentSeq int32, dest *LedgerCapacityUsageStats) error {
	const ledgers int32 = 5
	return q.GetRaw(dest, `
		SELECT ROUND(SUM(CAST(operation_count as decimal))/SUM(max_tx_set_size), 2) as ledger_capacity_usage FROM
			(SELECT
			  hl.sequence, COALESCE(SUM(ht.operation_count), 0) as operation_count, hl.max_tx_set_size
			  FROM history_ledgers hl
			  LEFT JOIN history_transactions ht ON ht.ledger_sequence = hl.sequence
			  WHERE hl.sequence > $1 AND hl.sequence <= $2
			  GROUP BY hl.sequence, hl.max_tx_set_size) as a
	`, currentSeq-ledgers, currentSeq)
}

// Page specifies the paging constraints for the query being built by `q`.
func (q *LedgersQ) Page(page db2.PageQuery) *LedgersQ {
	if q.Err != nil {
		return q
	}

	q.sql, q.Err = page.ApplyTo(q.sql, "hl.id")
	return q
}

// Select loads the results of the query specified by `q` into `dest`.
func (q *LedgersQ) Select(dest interface{}) error {
	if q.Err != nil {
		return q.Err
	}

	q.Err = q.parent.Select(dest, q.sql)
	return q.Err
}

// QExpLedgers defines experimental ingestion ledger related queries.
type QExpLedgers interface {
	CheckExpLedger(seq int32) (bool, error)
	InsertExpLedger(
		ledger xdr.LedgerHeaderHistoryEntry,
		successTxsCount int,
		failedTxsCount int,
		opCount int,
		ingestVersion int,
	) (int64, error)
}

// CheckExpLedger checks that the ledger in exp_history_ledgers
// matches the one in history_ledgers  for a given sequence number
func (q *Q) CheckExpLedger(seq int32) (bool, error) {
	expLedger, err := q.expLedgerBySequence(seq)
	if err != nil {
		return false, err
	}

	var ledger Ledger
	err = q.LedgerBySequence(&ledger, seq)
	if err != nil {
		return false, err
	}

	// ignore importer version created time, and updated time
	expLedger.ImporterVersion = ledger.ImporterVersion
	expLedger.CreatedAt = ledger.CreatedAt
	expLedger.UpdatedAt = ledger.UpdatedAt

	// compare ClosedAt separately because reflect.DeepEqual does not handle time.Time
	expClosedAt := expLedger.ClosedAt
	expLedger.ClosedAt = ledger.ClosedAt

	return expClosedAt.Equal(ledger.ClosedAt) && reflect.DeepEqual(expLedger, ledger), nil
}

// InsertExpLedger creates a row in the exp_history_ledgers table.
// Returns number of rows affected and error.
func (q *Q) InsertExpLedger(
	ledger xdr.LedgerHeaderHistoryEntry,
	successTxsCount int,
	failedTxsCount int,
	opCount int,
	ingestVersion int,
) (int64, error) {
	m, err := ledgerHeaderToMap(
		ledger,
		successTxsCount,
		failedTxsCount,
		opCount,
		ingestVersion,
	)
	if err != nil {
		return 0, err
	}

	sql := sq.Insert("exp_history_ledgers").SetMap(m)
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// ExpIngestRemovalSummary describes how many rows in the experimental ingestion
// history tables have been deleted by RemoveExpIngestHistory()
type ExpIngestRemovalSummary struct {
	LedgersRemoved int64
}

// RemoveExpIngestHistory removes all rows in the experimental ingestion
// history tables which have a ledger sequence higher than `newerThanSequence`
func (q *Q) RemoveExpIngestHistory(newerThanSequence uint32) (ExpIngestRemovalSummary, error) {
	result, err := q.Exec(
		sq.Delete("exp_history_ledgers").
			Where("sequence > ?", newerThanSequence),
	)
	if err != nil {
		return ExpIngestRemovalSummary{}, err
	}

	summary := ExpIngestRemovalSummary{}
	summary.LedgersRemoved, err = result.RowsAffected()
	return summary, err
}

func ledgerHeaderToMap(
	ledger xdr.LedgerHeaderHistoryEntry,
	successTxsCount int,
	failedTxsCount int,
	opCount int,
	importerVersion int,
) (map[string]interface{}, error) {
	ledgerHeaderBase64, err := xdr.MarshalBase64(ledger.Header)
	if err != nil {
		return nil, err
	}
	closeTime := time.Unix(int64(ledger.Header.ScpValue.CloseTime), 0).UTC()
	return map[string]interface{}{
		"importer_version":             importerVersion,
		"id":                           toid.New(int32(ledger.Header.LedgerSeq), 0, 0).ToInt64(),
		"sequence":                     ledger.Header.LedgerSeq,
		"ledger_hash":                  hex.EncodeToString(ledger.Hash[:]),
		"previous_ledger_hash":         null.NewString(hex.EncodeToString(ledger.Header.PreviousLedgerHash[:]), ledger.Header.LedgerSeq > 1),
		"total_coins":                  ledger.Header.TotalCoins,
		"fee_pool":                     ledger.Header.FeePool,
		"base_fee":                     ledger.Header.BaseFee,
		"base_reserve":                 ledger.Header.BaseReserve,
		"max_tx_set_size":              ledger.Header.MaxTxSetSize,
		"closed_at":                    closeTime,
		"created_at":                   time.Now().UTC(),
		"updated_at":                   time.Now().UTC(),
		"transaction_count":            successTxsCount,
		"successful_transaction_count": successTxsCount,
		"failed_transaction_count":     failedTxsCount,
		"operation_count":              opCount,
		"protocol_version":             ledger.Header.LedgerVersion,
		"ledger_header":                ledgerHeaderBase64,
	}, nil
}

var selectLedgerFields = sq.Select(
	"hl.id",
	"hl.sequence",
	"hl.importer_version",
	"hl.ledger_hash",
	"hl.previous_ledger_hash",
	"hl.transaction_count",
	"hl.successful_transaction_count",
	"hl.failed_transaction_count",
	"hl.operation_count",
	"hl.closed_at",
	"hl.created_at",
	"hl.updated_at",
	"hl.total_coins",
	"hl.fee_pool",
	"hl.base_fee",
	"hl.base_reserve",
	"hl.max_tx_set_size",
	"hl.protocol_version",
	"hl.ledger_header",
)
var selectLedger = selectLedgerFields.From("history_ledgers hl")
