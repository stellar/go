package actions

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
)

// OperationsQuery query struct for offers end-point
type OperationsQuery struct {
	AccountID       string `schema:"account_id" valid:"accountID,optional"`
	TransactionHash string `schema:"tx_id" valid:"transactionHash,optional"`
	IncludeFailed   bool   `schema:"include_failed"`
	LedgerID        string `schema:"ledger_id" valid:"ledgerID,optional"`
}

// Ledger returns the ledger id from the query parameter as an integer
func (qp OperationsQuery) Ledger() (int32, error) {
	if qp.LedgerID == "" {
		return 0, nil
	}

	ledger, err := strconv.ParseInt(qp.LedgerID, 10, 32)
	if err != nil {
		return 0, problem.MakeInvalidFieldProblem(
			"ledger_id",
			errors.Wrapf(err, "invalid ledger_id"),
		)
	}

	return int32(ledger), nil
}

// GetOperationsHandler is the action handler for all end-points returning a list of operations.
type GetOperationsHandler struct {
	OnlyPayments bool
}

// GetResourcePage returns a page of operations.
func (handler GetOperationsHandler) GetResourcePage(w HeaderWriter, r *http.Request) ([]hal.Pageable, error) {
	ctx := r.Context()

	pq, err := GetPageQuery(r)
	if err != nil {
		return nil, err
	}

	err = ValidateCursorWithinHistory(pq)
	if err != nil {
		return nil, err
	}

	qp := OperationsQuery{}
	err = GetParams(&qp, r)
	if err != nil {
		return nil, err
	}

	historyQ, err := HistoryQFromRequest(r)
	if err != nil {
		return nil, err
	}

	query := historyQ.Operations()

	if qp.AccountID != "" {
		query.ForAccount(qp.AccountID)
		if query.Err != nil {
			return nil, query.Err
		}
	}
	if qp.TransactionHash != "" {
		query.ForTransaction(qp.TransactionHash)
		if query.Err != nil {
			return nil, query.Err
		}
	}

	// When querying operations for transaction return both successful
	// and failed operations. We assume that because the user is querying
	// this specific transactions, they knows its status.
	if qp.TransactionHash != "" || qp.IncludeFailed {
		query.IncludeFailed()
		if query.Err != nil {
			return nil, query.Err
		}
	}

	ledgerID, err := qp.Ledger()
	if err != nil {
		return nil, err
	}
	if ledgerID > 0 {
		query.ForLedger(ledgerID)
		if query.Err != nil {
			return nil, query.Err
		}
	}

	if handler.OnlyPayments {
		query.OnlyPayments()
	}

	ops, txs, err := query.Page(pq).Fetch()
	if err != nil {
		return nil, err
	}

	// TODO: add test and run this check
	// for i, o := range action.OperationRecords {
	// 	if !action.IncludeFailed && action.TransactionFilter == "" {
	// 		if !o.TransactionSuccessful {
	// 			action.Err = errors.Errorf("Corrupted data! `include_failed=false` but returned transaction in /operations is failed: %s", o.TransactionHash)
	// 			return
	// 		}

	// 		var resultXDR xdr.TransactionResult
	// 		action.Err = xdr.SafeUnmarshalBase64(o.TxResult, &resultXDR)
	// 		if action.Err != nil {
	// 			return
	// 		}

	// 		if !resultXDR.Successful() {
	// 			action.Err = errors.Errorf("Corrupted data! `include_failed=false` but returned transaction /operations is failed: %s %s", o.TransactionHash, o.TxResult)
	// 			return
	// 		}
	// 	}
	// 	if action.IncludeTransactions {
	// 		transaction := action.TransactionRecords[i]
	// 		action.Err = validateTransactionForOperation(transaction, o)
	// 		if action.Err != nil {
	// 			return
	// 		}
	// 	}
	// }
	return buildOperationsPage(ctx, historyQ, ops, txs)
}

func buildOperationsPage(ctx context.Context, historyQ *history.Q, operations []history.Operation, transactions []history.Transaction) ([]hal.Pageable, error) {
	ledgerCache := history.LedgerCache{}
	for _, record := range operations {
		ledgerCache.Queue(record.LedgerSequence())
	}

	if err := ledgerCache.Load(historyQ); err != nil {
		return nil, errors.Wrap(err, "failed to load ledger batch")
	}

	var response []hal.Pageable
	for i, operationRecord := range operations {
		ledger, found := ledgerCache.Records[operationRecord.LedgerSequence()]
		if !found {
			msg := fmt.Sprintf("could not find ledger data for sequence %d", operationRecord.LedgerSequence())
			return nil, errors.New(msg)
		}

		var transactionRecord *history.Transaction

		// TODO: fix -- maybe we should pass down the query?
		// if action.IncludeTransactions {
		if false {
			transactionRecord = &transactions[i]
		}

		var res hal.Pageable
		transactionHash := "" // this doesn't make sense -- if we  are using the query then all tx will belong to the tx hash action.TransactionFilter
		if len(transactionHash) == 0 {
			transactionHash = operationRecord.TransactionHash
		}
		res, err := resourceadapter.NewOperation(
			ctx,
			operationRecord,
			transactionHash,
			transactionRecord,
			ledger,
		)
		if err != nil {
			return nil, err
		}
		response = append(response, res)
	}

	return response, nil
}
