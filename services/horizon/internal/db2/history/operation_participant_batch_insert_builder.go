package history

import (
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// OperationParticipantBatchInsertBuilder is used to insert a transaction's operations into the
// exp_history_operations table
type OperationParticipantBatchInsertBuilder interface {
	Add(
		operationID int64,
		accountID int64,
	) error
	Exec() error
}

// operationParticipantBatchInsertBuilder is a simple wrapper around db.BatchInsertBuilder
type operationParticipantBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

// NewOperationParticipantBatchInsertBuilder constructs a new TransactionBatchInsertBuilder instance
func (q *Q) NewOperationParticipantBatchInsertBuilder(maxBatchSize int) OperationParticipantBatchInsertBuilder {
	return &operationParticipantBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("exp_history_operation_participants"),
			MaxBatchSize: maxBatchSize,
		},
	}
}

// Add adds an operation participant to the batch
func (i *operationParticipantBatchInsertBuilder) Add(
	operationID int64,
	accountID int64,
) error {
	return i.builder.Row(map[string]interface{}{
		"history_operation_id": operationID,
		"history_account_id":   accountID,
	})
}

func (i *operationParticipantBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}

// OperationsParticipants returns a map with all participants per operation
func OperationsParticipants(transaction io.LedgerTransaction, sequence uint32) (map[int64][]xdr.AccountId, error) {
	participants := map[int64][]xdr.AccountId{}

	for opi, op := range transaction.Envelope.Tx.Operations {
		operation := transactionOperationWrapper{
			index:          uint32(opi),
			transaction:    transaction,
			operation:      op,
			ledgerSequence: sequence,
		}

		p, err := operation.Participants()
		if err != nil {
			return participants, errors.Wrapf(err, "reading operation %v participants", operation.ID())
		}
		participants[operation.ID()] = p
	}

	return participants, nil
}
