package database

import (
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/stellar/go/services/bifrost/stellar"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
)

var _ stellar.SubmissionArchive = &PostgresDatabase{}

type transactionSubmission struct {
	Type          stellar.SubmissionType `db:"type"`
	TransactionID string                 `db:"transaction_id"`
	AssetCode     string                 `db:"asset_code"`
	XDR           string                 `db:"xdr"`
	CreatedAt     time.Time              `db:"created_at"`
}

// Find loads persisted XDR from the DB in a new transaction.
// Returned xdr content can be empty when not found.
func (d *PostgresDatabase) Find(txID, assetCode string, st stellar.SubmissionType) (string, error) {
	var result string
	err := WithTx(d.session, ReadOnly(func(s *db.Session) error {
		return s.Get(&result, squirrel.Select("xdr").
			From(transactionSubmissionsTableName).
			Where("transaction_id = ? AND asset_code = ? AND type = ?", txID, assetCode, st))
	}))

	if errors.Cause(err) == sql.ErrNoRows {
		return "", nil
	}
	return result, err
}

// Store persists the given XDR content for a transaction and type. The operation is executed
// in a new database transaction.
func (d *PostgresDatabase) Store(txID, assetCode string, st stellar.SubmissionType, xdr string) error {
	return WithTx(d.session, func(s *db.Session) error {
		dbTable := d.getTable(transactionSubmissionsTableName, s)
		newEntity := &transactionSubmission{
			TransactionID: txID,
			AssetCode:     assetCode,
			Type:          st,
			XDR:           xdr,
			CreatedAt:     time.Now(),
		}
		_, err := dbTable.Insert(newEntity).Exec()
		return err
	})
}

// Store deletes a persisted XDR content for a transaction and type. The operation is executed
// in a new database transaction.
func (d *PostgresDatabase) Delete(txID, assetCode string, st stellar.SubmissionType) error {
	return WithTx(d.session, func(s *db.Session) error {
		dbTable := d.getTable(transactionSubmissionsTableName, s)
		where := map[string]interface{}{"transaction_id": txID, "asset_code": assetCode, "type": st}
		_, err := dbTable.Delete(where).Exec()
		return err
	})
}
