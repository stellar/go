package database

import (
	"database/sql"

	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
)

func (d *PostgresDatabase) Open(dsn string) error {
	var err error
	d.session, err = db.Open("postgres", dsn)
	if err != nil {
		return err
	}

	return nil
}

func (d *PostgresDatabase) getTable(name string, session *db.Session) *db.Table {
	if session == nil {
		session = d.session
	}

	return &db.Table{
		Name:    name,
		Session: session,
	}
}

func (d *PostgresDatabase) GetListenerLastCursorValue() (string, error) {
	var payment ReceivedPayment
	// TODO: `1=1`. We should be able to get row without WHERE clause.
	// When it's set to nil: `pq: syntax error at or near "ORDER""`
	receivedPaymentTable := d.getTable(receivedPaymentTableName, nil)
	err := receivedPaymentTable.Get(&payment, "1=1").OrderBy("id DESC").Exec()
	if err != nil {
		switch errors.Cause(err) {
		case sql.ErrNoRows:
			return "", nil
		default:
			return "", errors.Wrap(err, "Error getting events last ID")
		}
	}

	return payment.PagingToken, nil
}

func (d *PostgresDatabase) InsertReceivedPayment(payment ReceivedPayment) error {
	receivedPaymentTable := d.getTable(receivedPaymentTableName, nil)
	_, err := receivedPaymentTable.Insert(payment).Exec()
	return err
}

func (d *PostgresDatabase) GetReceivedPaymentByOperationID(id string) (ReceivedPayment, error) {
	var payment ReceivedPayment
	receivedPaymentTable := d.getTable(receivedPaymentTableName, nil)
	err := receivedPaymentTable.Get(&payment, "operation_id = ?", id).Exec()
	return payment, err
}

func (d *PostgresDatabase) UpdateReceivedPaymentStatus(operationID string, status string) error {
	receivedPaymentTable := d.getTable(receivedPaymentTableName, nil)
	_, err := receivedPaymentTable.Update(nil, map[string]interface{}{"operation_id": operationID}).Set("status", status).Exec()
	return err
}
