package actions

import (
	"context"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/xdr"
)

// TransactionPage returns a page containing the transaction records of an
// account/ledger identified by accountID/ledgerID into a page based on pq and
// includeFailedTx.
func TransactionPage(ctx context.Context, hq *history.Q, accountID string, ledgerID int32, includeFailedTx bool, pq db2.PageQuery) (hal.Page, error) {
	records, err := loadTransactionRecords(hq, accountID, ledgerID, includeFailedTx, pq)
	if err != nil {
		return hal.Page{}, errors.Wrap(err, "loading transaction records")
	}

	page := hal.Page{
		Cursor: pq.Cursor,
		Order:  pq.Order,
		Limit:  pq.Limit,
	}

	for _, record := range records {
		// TODO: make PopulateTransaction return horizon.Transaction directly.
		var res horizon.Transaction
		err = resourceadapter.PopulateTransaction(ctx, record.TransactionHash, &res, record)
		if err != nil {
			return hal.Page{}, errors.Wrap(err, "could not populate transaction")
		}
		page.Add(res)
	}

	page.FullURL = FullURL(ctx)
	page.PopulateLinks()
	return page, nil
}

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

// StreamTransactions streams transaction records of an account/ledger
// identified by accountID/ledgerID based on pq and includeFailedTx.
func StreamTransactions(ctx context.Context, s *sse.Stream, hq *history.Q, accountID string, ledgerID int32, includeFailedTx bool, pq db2.PageQuery) error {
	allRecords, err := loadTransactionRecords(hq, accountID, ledgerID, includeFailedTx, pq)
	if err != nil {
		return errors.Wrap(err, "loading transaction records")
	}

	s.SetLimit(int(pq.Limit))
	records := allRecords[s.SentCount():]
	for _, record := range records {
		var res horizon.Transaction
		err = resourceadapter.PopulateTransaction(ctx, record.TransactionHash, &res, record)
		if err != nil {
			return errors.Wrap(err, "could not populate transaction")
		}
		s.Send(sse.Event{ID: res.PagingToken(), Data: res})
	}

	return nil
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
