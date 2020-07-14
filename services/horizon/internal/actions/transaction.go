package actions

import (
	"context"
	"net/http"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	supportProblem "github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

// loadTransactionRecords returns a slice of transaction records of an
// account/ledger identified by accountID/ledgerID based on pq and
// includeFailedTx.
func loadTransactionRecords(hq *history.Q, accountID string, ledgerID int32, includeFailedTx bool, pq db2.PageQuery) ([]history.Transaction, error) {
	if accountID != "" && ledgerID != 0 {
		return nil, errors.New("conflicting exclusive fields are present: account_id and ledger_id")
	}

	var records []history.Transaction

	txs := hq.Transactions()
	switch {
	case accountID != "":
		txs.ForAccount(accountID)
	case ledgerID > 0:
		txs.ForLedger(ledgerID)
	}

	if includeFailedTx {
		txs.IncludeFailed()
	}

	err := txs.Page(pq).Select(&records)
	if err != nil {
		return nil, errors.Wrap(err, "executing transaction records query")
	}

	for _, t := range records {
		if !includeFailedTx {
			if !t.Successful {
				return nil, errors.Errorf("Corrupted data! `include_failed=false` but returned transaction is failed: %s", t.TransactionHash)
			}

			var resultXDR xdr.TransactionResult
			err = xdr.SafeUnmarshalBase64(t.TxResult, &resultXDR)
			if err != nil {
				return nil, errors.Wrap(err, "unmarshalling tx result")
			}

			if !resultXDR.Successful() {
				return nil, errors.Errorf("Corrupted data! `include_failed=false` but returned transaction is failed: %s %s", t.TransactionHash, t.TxResult)
			}
		}
	}

	return records, nil
}

// TransactionResource returns a single transaction resource identified by txHash.
func TransactionResource(ctx context.Context, hq *history.Q, txHash string) (horizon.Transaction, error) {
	var (
		record   history.Transaction
		resource horizon.Transaction
	)
	err := hq.TransactionByHash(&record, txHash)
	if err != nil {
		return resource, errors.Wrap(err, "loading transaction record")
	}

	if err = resourceadapter.PopulateTransaction(ctx, txHash, &resource, record); err != nil {
		return resource, errors.Wrap(err, "could not populate transaction")
	}
	return resource, nil
}

// TransactionsQuery query struct for transactions end-points
type TransactionsQuery struct {
	AccountID                 string `schema:"account_id" valid:"accountID,optional"`
	IncludeFailedTransactions bool   `schema:"include_failed" valid:"-"`
	LedgerID                  uint32 `schema:"ledger_id" valid:"-"`
}

// Validate runs extra validations on query parameters
func (qp TransactionsQuery) Validate() error {
	filters, err := countNonEmpty(
		qp.AccountID,
		qp.LedgerID,
	)

	if err != nil {
		return supportProblem.BadRequest
	}

	if filters > 1 {
		return supportProblem.MakeInvalidFieldProblem(
			"filters",
			errors.New("Use a single filter for transaction, you can only use one of account_id or ledger_id"),
		)
	}

	return nil
}

// GetTransactionsHandler is the action handler for all end-points returning a list of transactions.
type GetTransactionsHandler struct {
}

// GetResourcePage returns a page of operations.
func (handler GetTransactionsHandler) GetResourcePage(w HeaderWriter, r *http.Request) ([]hal.Pageable, error) {
	ctx := r.Context()

	pq, err := GetPageQuery(r)
	if err != nil {
		return nil, err
	}

	err = ValidateCursorWithinHistory(pq)
	if err != nil {
		return nil, err
	}

	qp := TransactionsQuery{}
	err = GetParams(&qp, r)
	if err != nil {
		return nil, err
	}

	historyQ, err := HistoryQFromRequest(r)
	if err != nil {
		return nil, err
	}

	records, err := loadTransactionRecords(historyQ, qp.AccountID, int32(qp.LedgerID), qp.IncludeFailedTransactions, pq)
	if err != nil {
		return nil, errors.Wrap(err, "loading transaction records")
	}

	var response []hal.Pageable

	for _, record := range records {
		var res horizon.Transaction
		err = resourceadapter.PopulateTransaction(ctx, record.TransactionHash, &res, record)
		if err != nil {
			return nil, errors.Wrap(err, "could not populate transaction")
		}
		response = append(response, res)
	}

	return response, nil
}
