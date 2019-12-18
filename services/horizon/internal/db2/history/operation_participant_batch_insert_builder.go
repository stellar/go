package history

import (
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
)

// OperationParticipantBatchInsertBuilder is used to insert a transaction's operations into the
// exp_history_operations table
type OperationParticipantBatchInsertBuilder interface {
	Add(
		transaction io.LedgerTransaction,
		sequence uint32,
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

// Add adds a transaction's operations to the batch
func (i *operationParticipantBatchInsertBuilder) Add(
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

		participants, err := operation.Participants()
		if err != nil {
			return errors.Wrapf(err, "reading operation %v participants", operation.ID())
		}

		for _, participant := range participants {
			accountID, err := addressTOID(participant.Address())
			if err != nil {
				return errors.Wrapf(err, "creating a exp_history_account for %v", participant.Address())
			}

			err = i.builder.Row(map[string]interface{}{
				"history_operation_id": operation.ID(),
				"history_account_id":   accountID,
			})
			if err != nil {
				return errors.Wrap(err, "Error batch inserting operation rows")
			}
		}
	}

	return nil
}

func addressTOID(address string) (int64, error) {
	// fixed values for test
	db := map[string]int64{
		"GBUT7HKGKIRQBN7CLNOTOSFKY2X2N62FABMARJI7LFMQZQZU5ZZYHXXG": 1,
		"GBN2NQDPELW7QZZBZQFTACZ7SZZVSTVU4BRFP2Q7NXFE4PKPTAB4AY4S": 2,
	}

	// Temporary workaround while we get Q.GetCreateAccountID
	id, ok := db[address]

	if !ok {
		id = 0
	}
	return id, nil
}

func (i *operationParticipantBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}
