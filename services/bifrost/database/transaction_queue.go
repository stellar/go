package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/stellar/go/services/bifrost/queue"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
)

var (
	_                         queue.Queue = &PostgresDatabase{}
	DefaultTransactionLockTTL             = 2 * time.Minute
)

// QueueAdd implements queue.Queue interface. If element already exists in a queue, it should
// return nil.
func (d *PostgresDatabase) QueueAdd(tx queue.Transaction) error {
	transactionsQueueTable := d.getTable(transactionsQueueTableName, nil)
	transactionQueue := fromQueueTransaction(tx)
	_, err := transactionsQueueTable.Insert(transactionQueue).Exec()
	if err != nil {
		if isDuplicateError(err) {
			return nil
		}
	}
	return err
}

// IsEmpty returns a snapshot status of the queue.
func (d *PostgresDatabase) IsEmpty() (bool, error) {
	var result bool
	err := WithTx(d.session, ReadOnly(func(s *db.Session) error {
		rows, err := s.Query(squirrel.Select("count(id) = 0").
			From(transactionsQueueTableName).
			Where("pooled = false AND (locked_until is null OR locked_until < ?)"+
				" AND failure_count < ?", time.Now(), maxProcessingFailureCount))
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			return rows.Scan(&result)
		}
		return nil
	}))
	return result, err
}

// QueuePool receives and removes the head of this queue. Returns nil if no elements found.
// QueuePool implements queue.Queue interface.
func (d *PostgresDatabase) WithQueuedTransaction(transactionHandler func(queue.Transaction) error) error {
	var row transactionsQueueRow
	var sessionToken string
	err := WithTx(d.session, func(s *db.Session) error {
		err := s.Get(&row, squirrel.Select("transaction_id, asset_code, amount, stellar_public_key, failure_count").
			From(transactionsQueueTableName).
			Where("pooled = false AND (locked_until is null OR locked_until < ?)"+
				" AND failure_count < ?", time.Now(), maxProcessingFailureCount).
			OrderBy("failure_count ASC, id ASC").
			Suffix("FOR UPDATE SKIP Locked").
			Limit(1))
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil
			}
			return errors.Wrap(err, "failed to get transaction from the queue")
		}

		// set processing lock
		sessionToken, err = newToken()
		if err != nil {
			return errors.Wrap(err, "failed to create session token")
		}
		transactionsQueueTable := d.getTable(transactionsQueueTableName, s)
		where := map[string]interface{}{"transaction_id": row.TransactionID, "asset_code": row.AssetCode}
		_, err = transactionsQueueTable.Update(nil, where).
			Set("locked_until", time.Now().Add(DefaultTransactionLockTTL)).
			Set("locked_token", sessionToken).
			Exec()
		return err
	})
	switch {
	case err != nil:
		return errors.Wrap(err, "failed to find and lock transaction")
	case row.TransactionID == "": // transient
		return nil
	}
	// process callback without surrounding transaction
	transaction := row.toQueueTransaction()
	defer d.releaseTransactionLock(transaction.TransactionID, sessionToken)

	if err := transactionHandler(transaction); err != nil {
		if err != context.DeadlineExceeded {
			// increase failure counter
			_ = WithTx(d.session, func(s *db.Session) error {
				where := map[string]interface{}{"transaction_id": row.TransactionID, "asset_code": row.AssetCode, "locked_token": sessionToken}
				transactionsQueueTable := d.getTable(transactionsQueueTableName, s)
				_, err := transactionsQueueTable.Update(nil, where).Set("failure_count", row.FailureCount+1).Exec()
				return err
			})
		}
		return errors.Wrap(err, "failed to process transaction")
	}

	// update pooled status
	err = WithTx(d.session, func(s *db.Session) error {
		where := map[string]interface{}{"transaction_id": row.TransactionID, "asset_code": row.AssetCode, "locked_token": sessionToken}
		transactionsQueueTable := d.getTable(transactionsQueueTableName, s)
		_, err := transactionsQueueTable.Update(nil, where).Set("pooled", true).Exec()
		return err
	})
	if err != nil {
		return errors.Wrap(err, "failed to set pooled status for transaction")
	}
	return nil
}

func (d *PostgresDatabase) releaseTransactionLock(transactionID string, sessionToken string) error {
	return WithTx(d.session, func(s *db.Session) error {
		transactionsQueueTable := d.getTable(transactionsQueueTableName, s)
		where := map[string]interface{}{"transaction_id": transactionID, "locked_token": sessionToken}
		_, err := transactionsQueueTable.Update(nil, where).
			Set("locked_until", nil).
			Set("locked_token", nil).
			Exec()
		return err
	})
}

func newToken() (string, error) {
	raw := make([]byte, 8)
	_, err := rand.Read(raw)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%X", raw), nil
}
