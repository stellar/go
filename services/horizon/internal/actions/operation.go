package actions

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ledger"
	iproblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
)

// OperationsQuery query struct for operations end-points
type OperationsQuery struct {
	AccountID                 string `schema:"account_id" valid:"accountID,optional"`
	TransactionHash           string `schema:"tx_id" valid:"transactionHash,optional"`
	IncludeFailedTransactions bool   `schema:"include_failed" valid:"-"`
	LedgerID                  uint32 `schema:"ledger_id" valid:"-"`
	Join                      string `schema:"join" valid:"in(transactions)~Accepted values: transactions,optional"`
}

// IncludeTransactions returns extra fields to include in the response
func (qp OperationsQuery) IncludeTransactions() bool {
	return qp.Join == "transactions"
}

// Validate runs extra validations on query parameters
func (qp OperationsQuery) Validate() error {
	filters, err := countNonEmpty(
		qp.AccountID,
		qp.LedgerID,
		qp.TransactionHash,
	)

	if err != nil {
		return problem.BadRequest
	}

	if filters > 1 {
		return problem.MakeInvalidFieldProblem(
			"filters",
			errors.New("Use a single filter for operations, you can only use one of tx_id, account_id or ledger_id"),
		)
	}

	return nil
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
	if err = GetParams(&qp, r); err != nil {
		return nil, err
	}

	historyQ, err := HistoryQFromRequest(r)
	if err != nil {
		return nil, err
	}

	query := historyQ.Operations()

	switch {
	case qp.AccountID != "":
		query.ForAccount(qp.AccountID)
	case qp.LedgerID > 0:
		query.ForLedger(int32(qp.LedgerID))
	case qp.TransactionHash != "":
		query.ForTransaction(qp.TransactionHash)
	}
	// When querying operations for transaction return both successful
	// and failed operations. We assume that because the user is querying
	// this specific transactions, they knows its status.
	if qp.TransactionHash != "" || qp.IncludeFailedTransactions {
		query.IncludeFailed()
	}

	if qp.IncludeTransactions() {
		query.IncludeTransactions()
	}

	if handler.OnlyPayments {
		query.OnlyPayments()
	}

	ops, txs, err := query.Page(pq).Fetch()
	if err != nil {
		return nil, err
	}

	return buildOperationsPage(ctx, historyQ, ops, txs, qp.IncludeTransactions())
}

func buildOperationsPage(ctx context.Context, historyQ *history.Q, operations []history.Operation, transactions []history.Transaction, includeTransactions bool) ([]hal.Pageable, error) {
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

		if includeTransactions {
			transactionRecord = &transactions[i]
		}

		var res hal.Pageable
		res, err := resourceadapter.NewOperation(
			ctx,
			operationRecord,
			operationRecord.TransactionHash,
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

// OperationsByIDQuery query struct for operations end-points
type OperationsByIDQuery struct {
	OperationID uint64 `schema:"op_id" valid:"-"`
	Join        string `schema:"join" valid:"in(transactions)~Accepted values: transactions,optional"`
}

// IncludeTransactions returns extra fields to include in the response
func (qp OperationsByIDQuery) IncludeTransactions() bool {
	return qp.Join == "transactions"
}

type GetOperationsByIDHandler struct{}

func (handler GetOperationsByIDHandler) GetResource(w HeaderWriter, r *http.Request) (hal.Pageable, error) {
	qp := OperationsByIDQuery{}
	if err := GetParams(&qp, r); err != nil {
		return nil, err
	}

	parsed := toid.Parse(int64(qp.OperationID))
	if parsed.LedgerSequence < ledger.CurrentState().HistoryElder {
		return nil, iproblem.BeforeHistory
	}

	historyQ, err := HistoryQFromRequest(r)
	if err != nil {
		return nil, err
	}

	operationRecord, transactionRecord, err := historyQ.OperationByID(qp.IncludeTransactions(), int64(qp.OperationID))
	if err != nil {
		return nil, err
	}

	var ledger history.Ledger
	if err = historyQ.LedgerBySequence(&ledger, operationRecord.LedgerSequence()); err != nil {
		return nil, err
	}

	operation, err := resourceadapter.NewOperation(
		r.Context(),
		operationRecord,
		operationRecord.TransactionHash,
		transactionRecord,
		ledger,
	)
	if err != nil {
		return nil, err
	}

	return operation, nil

}
