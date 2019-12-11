package history

import (
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/xdr"
)

// TransactionOperation represents the data for a single operation within a transaction
type TransactionOperation struct {
	Index       uint32
	Transaction io.LedgerTransaction
	Operation   xdr.Operation
}

// ID returns the ID for the operation.
func (op *TransactionOperation) ID(sequence uint32) int64 {
	return toid.New(
		int32(sequence),
		int32(op.Transaction.Index),
		int32(op.Index),
	).ToInt64()
}

// TxID returns the id for the transaction related with this operation.
func (op *TransactionOperation) TxID() int64 {
	return 0
}

// Order returns the order of this operation within the transaction's operations.
func (op *TransactionOperation) Order() int32 {
	return int32(op.Index)
}

// // SourceAccount returns the operation's source account.
// func (op *TransactionOperation) SourceAccount() xdr.AccountId {
// 	return nil
// }

// // OperationType returns the operation type.
// func (op *TransactionOperation) OperationType() xdr.OperationType {
// 	return nil

// }

// // Details returns the operation details as a map which can be stored as JSON.
// func (op *TransactionOperation) Details() map[string]interface{} {
// }
