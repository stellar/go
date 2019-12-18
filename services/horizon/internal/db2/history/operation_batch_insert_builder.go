package history

import (
	"encoding/json"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
)

// OperationBatchInsertBuilder is used to insert a transaction's operations into the
// exp_history_operations table
type OperationBatchInsertBuilder interface {
	Add(
		transaction io.LedgerTransaction,
		sequence uint32,
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

// Add adds a transaction's operations to the batch
func (i *operationBatchInsertBuilder) Add(
	transaction io.LedgerTransaction,
	sequence uint32,
) error {
	for opi, op := range transaction.Envelope.Tx.Operations {
		operation := transactionOperationWrapper{
			index:          uint32(opi),
			transaction:    transaction,
			operation:      op,
			ledgerSequence: sequence,
		}

		detailsJSON, err := json.Marshal(operation.Details())
		if err != nil {
			return errors.Wrap(err, "Error marshaling details")
		}
		err = i.builder.Row(map[string]interface{}{
			"id":                operation.ID(),
			"transaction_id":    operation.TransactionID(),
			"application_order": operation.Order(),
			"type":              operation.OperationType(),
			"details":           detailsJSON,
			"source_account":    operation.SourceAccount().Address(),
		})
		if err != nil {
			return errors.Wrap(err, "Error batch inserting operation rows")
		}
	}

	return nil
}

func (i *operationBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}
