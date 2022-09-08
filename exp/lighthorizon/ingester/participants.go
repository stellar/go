package ingester

import (
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/support/collections/set"
	"github.com/stellar/go/xdr"
)

// GetTransactionParticipants takes a LedgerTransaction and returns a set of all
// participants (accounts) in the transaction. If there is any error, it will
// return nil and the error.
func GetTransactionParticipants(tx LedgerTransaction) (set.Set[string], error) {
	participants, err := index.GetTransactionParticipants(*tx.LedgerTransaction)
	if err != nil {
		return nil, err
	}
	set := set.NewSet[string](len(participants))
	set.AddSlice(participants)
	return set, nil
}

// GetOperationParticipants takes a LedgerTransaction, the Operation within the
// transaction, and the 0-based index of the operation within the transaction.
// It will return a set of all participants (accounts) in the operation. If
// there is any error, it will return nil and the error.
func GetOperationParticipants(tx LedgerTransaction, op xdr.Operation, opIndex int) (set.Set[string], error) {
	participants, err := index.GetOperationParticipants(*tx.LedgerTransaction, op, opIndex)
	if err != nil {
		return nil, err
	}

	set := set.NewSet[string](len(participants))
	set.AddSlice(participants)
	return set, nil
}
