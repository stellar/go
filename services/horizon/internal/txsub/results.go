package txsub

import (
	"database/sql"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
)

func txResultByHash(db HorizonDB, hash string) (history.Transaction, error) {
	// query history database
	var hr history.Transaction
	err := db.TransactionByHash(&hr, hash)
	if err == nil {
		return txResultFromHistory(hr)
	}

	if !db.NoRows(err) {
		return hr, err
	}

	// if no result was found in either db, return ErrNoResults
	return hr, ErrNoResults
}

func txResultFromHistory(tx history.Transaction) (history.Transaction, error) {
	var txResult xdr.TransactionResult
	err := xdr.SafeUnmarshalBase64(tx.TxResult, &txResult)
	if err == nil {
		if !txResult.Successful() {
			err = &FailedTransactionError{
				ResultXDR: tx.TxResult,
			}
		}
	}

	return tx, err
}

func checkTxAlreadyExists(db HorizonDB, hash, sourceAddress string) (history.Transaction, uint64, error) {
	err := db.BeginTx(&sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return history.Transaction{}, 0, err
	}
	defer db.Rollback()

	tx, err := txResultByHash(db, hash)
	if err == ErrNoResults {
		var sequenceNumbers map[string]uint64
		sequenceNumbers, err = db.GetSequenceNumbers([]string{sourceAddress})
		if err != nil {
			return tx, 0, err
		}

		num, ok := sequenceNumbers[sourceAddress]
		if !ok {
			return tx, 0, ErrNoAccount
		}
		return tx, num, ErrNoResults
	}
	return tx, 0, err
}
