package history

import (
	"encoding/json"

	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// OperationBatchInsertBuilder is used to insert operations into the
// exp_history_operations table
type OperationBatchInsertBuilder interface {
	Add(
		id int64,
		txid int64,
		order int32,
		typ xdr.OperationType,
		details map[string]interface{},
		source xdr.AccountId,
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
	id int64,
	txid int64,
	order int32,
	typ xdr.OperationType,
	details map[string]interface{},
	source xdr.AccountId,
) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return errors.Wrap(err, "Error marshaling details")
	}
	return i.builder.Row(map[string]interface{}{
		"id":                id,
		"transaction_id":    txid,
		"application_order": order,
		"type":              typ,
		"details":           detailsJSON,
		"source_account":    source.Address(),
	})
}

func (i *operationBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}
