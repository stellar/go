package history

import (
	"encoding/json"

	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
)

// QOperations defines exp_history_operation related queries.
type QOperations interface {
	NewOperationBatchInsertBuilder(maxBatchSize int) OperationBatchInsertBuilder
	CheckExpOperations(seq int32) (bool, error)
}

// OperationBatchInsertBuilder is used to insert operations into the
// exp_history_operations table
type OperationBatchInsertBuilder interface {
	Add(
		operation TransactionOperation,
	) error
	Exec() error
}

// operationBatchInsertBuilder is a simple wrapper around db.BatchInsertBuilder
type operationBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

// NewOperationBatchInsertBuilder constructs a new TransactionBatchInsertBuilder instance
func (q *Q) NewOperationBatchInsertBuilder(maxBatchSize int) OperationBatchInsertBuilder {
	return &operationBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("exp_history_operations"),
			MaxBatchSize: maxBatchSize,
		},
	}
}

// Add adds a new transaction to the batch
func (i *operationBatchInsertBuilder) Add(
	operation TransactionOperation,
) error {
	detailsJSON, err := json.Marshal(operation.Details())
	if err != nil {
		return errors.Wrap(err, "Error marshaling details")
	}
	return i.builder.Row(map[string]interface{}{
		"id":                operation.ID(),
		"transaction_id":    operation.TransactionID(),
		"application_order": operation.Order(),
		"type":              operation.OperationType(),
		"details":           detailsJSON,
		"source_account":    operation.SourceAccount().Address(),
	})
}

func (i *operationBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}
