package history

import (
	"bytes"
	"encoding/json"
	"text/template"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LedgerSequence return the ledger in which the effect occurred.
func (r *Operation) LedgerSequence() int32 {
	id := toid.Parse(r.ID)
	return id.LedgerSequence
}

// UnmarshalDetails unmarshals the details of this operation into `dest`
func (r *Operation) UnmarshalDetails(dest interface{}) error {
	if !r.DetailsString.Valid {
		return nil
	}

	err := json.Unmarshal([]byte(r.DetailsString.String), &dest)
	if err != nil {
		err = errors.Wrap(err, "error in unmarshal")
	}

	return err
}

var feeStatsQueryTemplate = template.Must(template.New("trade_aggregations_query").Parse(`
{{define "operation_count"}}(CASE WHEN new_max_fee IS NULL THEN operation_count ELSE operation_count + 1 END){{end}}
SELECT
	{{range .}}
	ceil(percentile_disc(0.{{ . }}) WITHIN GROUP (ORDER BY fee_charged/{{template "operation_count"}}))::bigint AS "fee_charged_p{{ . }}",
	{{end}}
	ceil(max(fee_charged/{{template "operation_count"}}))::bigint AS "fee_charged_max",
	ceil(min(fee_charged/{{template "operation_count"}}))::bigint AS "fee_charged_min",
	ceil(mode() within group (order by fee_charged/{{template "operation_count"}}))::bigint AS "fee_charged_mode",

	{{range .}}
	ceil(percentile_disc(0.{{ . }}) WITHIN GROUP (ORDER BY COALESCE(new_max_fee, max_fee)/{{template "operation_count"}}))::bigint AS "max_fee_p{{ . }}",
	{{end}}
	ceil(max(COALESCE(new_max_fee, max_fee)/{{template "operation_count"}}))::bigint AS "max_fee_max",
	ceil(min(COALESCE(new_max_fee, max_fee)/{{template "operation_count"}}))::bigint AS "max_fee_min",
	ceil(mode() within group (order by COALESCE(new_max_fee, max_fee)/{{template "operation_count"}}))::bigint AS "max_fee_mode"
FROM history_transactions
WHERE ledger_sequence > $1 AND ledger_sequence <= $2`))

// FeeStats returns operation fee stats for the last 5 ledgers.
// Currently, we hard code the query to return the last 5 ledgers worth of transactions.
// TODO: make the number of ledgers configurable.
func (q *Q) FeeStats(currentSeq int32, dest *FeeStats) error {
	percentiles := []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 99}

	var buf bytes.Buffer
	err := feeStatsQueryTemplate.Execute(&buf, percentiles)
	if err != nil {
		return errors.Wrap(err, "error executing the query template")
	}

	return q.GetRaw(dest, buf.String(), currentSeq-5, currentSeq)
}

// Operations provides a helper to filter the operations table with pre-defined
// filters.  See `OperationsQ` for the available filters.
func (q *Q) Operations() *OperationsQ {
	query := &OperationsQ{
		parent:              q,
		opIdCol:             "hop.id",
		includeFailed:       false,
		includeTransactions: false,
		sql:                 selectOperation,
	}

	return query
}

// OperationByID returns an Operation and optionally a Transaction given an operation id
func (q *Q) OperationByID(includeTransactions bool, id int64) (Operation, *Transaction, error) {
	sql := selectOperation.
		Limit(1).
		Where("hop.id = ?", id)

	var operation Operation
	err := q.Get(&operation, sql)
	if err != nil {
		return operation, nil, err
	}

	if includeTransactions {
		var transaction Transaction
		if err = q.TransactionByHash(&transaction, operation.TransactionHash); err != nil {
			return operation, nil, err
		}

		err = validateTransactionForOperation(transaction, operation)
		if err != nil {
			return operation, nil, err
		}

		return operation, &transaction, err
	}
	return operation, nil, err
}

// ForAccount filters the operations collection to a specific account
func (q *OperationsQ) ForAccount(aid string) *OperationsQ {
	var account Account
	q.Err = q.parent.AccountByAddress(&account, aid)
	if q.Err != nil {
		return q
	}

	q.sql = q.sql.Join(
		"history_operation_participants hopp ON "+
			"hopp.history_operation_id = hop.id",
	).Where("hopp.history_account_id = ?", account.ID)

	// in order to use history_operation_participants.hist_op_p_id index
	q.opIdCol = "hopp.history_operation_id"

	return q
}

// ForClaimableBalance filters the query to only operations pertaining to a
// claimable balance, specified by the claimable balance's hex-encoded id.
func (q *OperationsQ) ForClaimableBalance(cbID xdr.ClaimableBalanceId) *OperationsQ {
	var hCB HistoryClaimableBalance
	hCB, q.Err = q.parent.ClaimableBalanceByID(cbID)
	if q.Err != nil {
		return q
	}

	q.sql = q.sql.Join(
		"history_operation_claimable_balances hocb ON "+
			"hocb.history_operation_id = hop.id",
	).Where("hocb.history_claimable_balance_id = ?", hCB.InternalID)

	// in order to use hocb.history_claimable_balance_id index
	q.opIdCol = "hocb.history_claimable_balance_id"

	return q
}

// ForLedger filters the query to a only operations in a specific ledger,
// specified by its sequence.
func (q *OperationsQ) ForLedger(seq int32) *OperationsQ {
	var ledger Ledger
	q.Err = q.parent.LedgerBySequence(&ledger, seq)
	if q.Err != nil {
		return q
	}

	start := toid.ID{LedgerSequence: seq}
	end := toid.ID{LedgerSequence: seq + 1}
	q.sql = q.sql.Where(
		"hop.id >= ? AND hop.id < ?",
		start.ToInt64(),
		end.ToInt64(),
	)

	return q
}

// ForTransaction filters the query to only operations in a specific
// transaction, specified by the transactions's hex-encoded hash.
func (q *OperationsQ) ForTransaction(hash string) *OperationsQ {
	var tx Transaction
	q.Err = q.parent.TransactionByHash(&tx, hash)
	if q.Err != nil {
		return q
	}

	start := toid.Parse(tx.ID)
	end := start
	end.TransactionOrder++
	q.sql = q.sql.Where(
		"hop.id >= ? AND hop.id < ?",
		start.ToInt64(),
		end.ToInt64(),
	)

	return q
}

// OnlyPayments filters the query being built to only include operations that
// are in the "payment" class of operations:  CreateAccountOps, Payments, and
// PathPayments.
func (q *OperationsQ) OnlyPayments() *OperationsQ {
	q.sql = q.sql.Where(sq.Eq{"hop.type": []xdr.OperationType{
		xdr.OperationTypeCreateAccount,
		xdr.OperationTypePayment,
		xdr.OperationTypePathPaymentStrictReceive,
		xdr.OperationTypePathPaymentStrictSend,
		xdr.OperationTypeAccountMerge,
	}})
	return q
}

// IncludeFailed changes the query to include failed transactions.
func (q *OperationsQ) IncludeFailed() *OperationsQ {
	q.includeFailed = true
	return q
}

// IncludeTransactions changes the query to fetch transaction data in addition to operation records.
func (q *OperationsQ) IncludeTransactions() *OperationsQ {
	q.includeTransactions = true
	return q
}

// Page specifies the paging constraints for the query being built by `q`.
func (q *OperationsQ) Page(page db2.PageQuery) *OperationsQ {
	if q.Err != nil {
		return q
	}

	q.sql, q.Err = page.ApplyTo(q.sql, q.opIdCol)
	return q
}

// Fetch returns results specified by a filtered operations query
func (q *OperationsQ) Fetch() ([]Operation, []Transaction, error) {
	if q.Err != nil {
		return nil, nil, q.Err
	}

	if !q.includeFailed {
		q.sql = q.sql.
			Where("(ht.successful = true OR ht.successful IS NULL)")
	}

	var operations []Operation
	var transactions []Transaction
	q.Err = q.parent.Select(&operations, q.sql)
	if q.Err != nil {
		return nil, nil, q.Err
	}
	set := map[int64]bool{}
	transactionIDs := []int64{}

	for _, o := range operations {
		var resultXDR xdr.TransactionResult
		err := xdr.SafeUnmarshalBase64(o.TxResult, &resultXDR)
		if err != nil {
			return nil, nil, err
		}

		if !set[o.TransactionID] {
			set[o.TransactionID] = true
			transactionIDs = append(transactionIDs, o.TransactionID)
		}

		if !q.includeFailed {
			if !o.TransactionSuccessful {
				return nil, nil, errors.Errorf("Corrupted data! `include_failed=false` but returned transaction is failed: %s", o.TransactionHash)
			}

			if !resultXDR.Successful() {
				return nil, nil, errors.Errorf("Corrupted data! `include_failed=false` but returned transaction is failed: %s %s", o.TransactionHash, o.TxResult)
			}
		}

		// Check if `successful` equals resultXDR
		if o.TransactionSuccessful && !resultXDR.Successful() {
			return nil, nil, errors.Errorf("Corrupted data! `successful=true` but returned transaction is not success: %s %s", o.TransactionHash, o.TxResult)
		}

		if !o.TransactionSuccessful && resultXDR.Successful() {
			return nil, nil, errors.Errorf("Corrupted data! `successful=false` but returned transaction is success: %s %s", o.TransactionHash, o.TxResult)
		}
	}

	if q.includeTransactions && len(transactionIDs) > 0 {
		transactionsByID, err := q.parent.TransactionsByIDs(transactionIDs...)
		if err != nil {
			return nil, nil, err
		}
		for _, o := range operations {
			transaction, ok := transactionsByID[o.TransactionID]
			if !ok {
				return nil, nil, errors.Errorf("transaction with id %v could not be found", o.TransactionID)
			}
			err = validateTransactionForOperation(transaction, o)
			if err != nil {
				return nil, nil, err
			}

			transactions = append(transactions, transaction)
		}
	}

	return operations, transactions, nil
}

func validateTransactionForOperation(transaction Transaction, operation Operation) error {
	if transaction.ID != operation.TransactionID {
		return errors.Errorf(
			"transaction id %v does not match transaction id in operation %v",
			transaction.ID,
			operation.TransactionID,
		)
	}
	if transaction.TransactionHash != operation.TransactionHash {
		return errors.Errorf(
			"transaction hash %v does not match transaction hash in operation %v",
			transaction.TransactionHash,
			operation.TransactionHash,
		)
	}
	if transaction.TxResult != operation.TxResult {
		return errors.Errorf(
			"transaction result %v does not match transaction result in operation %v",
			transaction.TxResult,
			operation.TxResult,
		)
	}
	if transaction.Successful != operation.TransactionSuccessful {
		return errors.Errorf(
			"transaction successful flag %v does not match transaction successful flag in operation %v",
			transaction.Successful,
			operation.TransactionSuccessful,
		)
	}

	return nil
}

// QOperations defines history_operation related queries.
type QOperations interface {
	NewOperationBatchInsertBuilder(maxBatchSize int) OperationBatchInsertBuilder
}

var selectOperation = sq.Select(
	"hop.id, " +
		"hop.transaction_id, " +
		"hop.application_order, " +
		"hop.type, " +
		"hop.details, " +
		"hop.source_account, " +
		"ht.transaction_hash, " +
		"ht.tx_result, " +
		"COALESCE(ht.successful, true) as transaction_successful").
	From("history_operations hop").
	LeftJoin("history_transactions ht ON ht.id = hop.transaction_id")
