// Package results provides an implementation of the txsub.ResultProvider interface
// backed using the SQL databases used by both stellar core and horizon
package results

import (
	"context"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/txsub"
	"github.com/stellar/go/xdr"
)

// DB provides transaction submission results by querying the
// connected horizon and, if set, stellar core databases.
type DB struct {
	History *history.Q
}

var _ txsub.ResultProvider = &DB{}

// ResultByHash implements txsub.ResultProvider
func (rp *DB) ResultByHash(ctx context.Context, hash string) txsub.Result {
	// query history database
	var hr history.Transaction
	err := rp.History.TransactionByHash(&hr, hash)
	if err == nil {
		return txResultFromHistory(hr)
	}

	if !rp.History.NoRows(err) {
		return txsub.Result{Err: err}
	}

	// if no result was found in either db, return ErrNoResults
	return txsub.Result{Err: txsub.ErrNoResults}
}

func txResultFromHistory(tx history.Transaction) txsub.Result {
	var txResult xdr.TransactionResult
	err := xdr.SafeUnmarshalBase64(tx.TxResult, &txResult)
	if err == nil {
		if !txResult.Successful() {
			err = &txsub.FailedTransactionError{
				ResultXDR: tx.TxResult,
			}
		}
	}

	return txsub.Result{Err: err, Transaction: tx}
}
