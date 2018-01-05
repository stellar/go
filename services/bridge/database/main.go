package database

import (
	"time"

	"github.com/stellar/go/support/db"
)

type Database interface {
	// GetListenerLastCursorValue returns start processed cursor value. Returns
	// empty string if none.
	GetListenerLastCursorValue() (string, error)
	// InsertReceivedPayment inserts a new ReceivedPayment to the DB.
	InsertReceivedPayment(payment ReceivedPayment) error
	// GetReceivedPaymentByOperationID gets received payment with the given
	// operation ID.
	GetReceivedPaymentByOperationID(operationID string) (ReceivedPayment, error)
	// UpdateReceivedPaymentStatus updates the status of a payment.
	UpdateReceivedPaymentStatus(operationID string, status string) error
}

var (
	receivedPaymentTableName = "received_payment"
)

// ReceivedPayment represents payment received by the gateway server
type ReceivedPayment struct {
	ID            int64     `db:"id" json:"id"`
	OperationID   string    `db:"operation_id" json:"operation_id"`
	TransactionID string    `db:"transaction_id" json:"transaction_id"`
	ProcessedAt   time.Time `db:"processed_at" json:"processed_at"`
	PagingToken   string    `db:"paging_token" json:"paging_token"`
	Status        string    `db:"status" json:"status"`
}

type PostgresDatabase struct {
	session *db.Session
}
