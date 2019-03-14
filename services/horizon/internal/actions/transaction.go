package actions

import (
	"context"
	"net/url"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/httpx"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/xdr"
)

type TransactionParams struct {
	AccountFilter string
	LedgerFilter  int32
	PagingParams  db2.PageQuery
	IncludeFailed bool
}

func TransactionPageByAccount(ctx context.Context, hq *history.Q, addr string, includeFailedTx bool, pq db2.PageQuery) (hal.Page, error) {
	page := hal.Page{
		Cursor: pq.Cursor,
		Order:  pq.Order,
		Limit:  pq.Limit,
	}
	records, err := loadTransactionRecordByAccount(hq, addr, includeFailedTx, pq)
	if err != nil {
		return page, errors.Wrap(err, "loading transaction records by account")
	}

	for _, record := range records {
		var res horizon.Transaction
		resourceadapter.PopulateTransaction(ctx, &res, record)
		page.Add(res)
	}

	page.FullURL = fullURL(ctx)
	page.PopulateLinks()

	return page, nil
}

func loadTransactionRecordByAccount(hq *history.Q, addr string, includeFailedTx bool, pq db2.PageQuery) ([]history.Transaction, error) {
	var records []history.Transaction

	txs := hq.Transactions()
	txs.ForAccount(addr)

	if includeFailedTx {
		txs.IncludeFailed()
	}

	err := txs.Page(pq).Select(&records)
	if err != nil {
		return nil, errors.Wrap(err, "getting transaction records by account")
	}

	for _, t := range records {
		if !includeFailedTx {
			if !t.IsSuccessful() {
				return nil, errors.Errorf("Corrupted data! `include_failed=false` but returned transaction is failed: %s", t.TransactionHash)
			}

			var resultXDR xdr.TransactionResult
			err = xdr.SafeUnmarshalBase64(t.TxResult, &resultXDR)
			if err != nil {
				return nil, errors.Wrap(err, "unmarshalling tx result")
			}

			if resultXDR.Result.Code != xdr.TransactionResultCodeTxSuccess {
				return nil, errors.Errorf("Corrupted data! `include_failed=false` but returned transaction is failed: %s %s", t.TransactionHash, t.TxResult)
			}
		}
	}

	return records, nil
}

func fullURL(ctx context.Context) *url.URL {
	url := httpx.BaseURL(ctx)
	r := httpx.RequestFromContext(ctx)
	url.Path = r.URL.Path
	url.RawQuery = r.URL.RawQuery
	return url
}
