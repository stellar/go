package ingest

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LedgerTransaction represents the data for a single transaction within a ledger.
type LedgerTransaction struct {
	Index    uint32
	Envelope xdr.TransactionEnvelope
	Result   xdr.TransactionResultPair
	// FeeChanges and UnsafeMeta are low level values, do not use them directly unless
	// you know what you are doing.
	// Use LedgerTransaction.GetChanges() for higher level access to ledger
	// entry changes.
	FeeChanges    xdr.LedgerEntryChanges
	UnsafeMeta    xdr.TransactionMeta
	LedgerVersion uint32
	Ledger        xdr.LedgerCloseMeta // This is read-only and not to be modified by downstream functions
	Hash          xdr.Hash
}

func (t *LedgerTransaction) txInternalError() bool {
	return t.Result.Result.Result.Code == xdr.TransactionResultCodeTxInternalError
}

// GetFeeChanges returns a developer friendly representation of LedgerEntryChanges
// connected to fees.
func (t *LedgerTransaction) GetFeeChanges() []Change {
	changes := GetChangesFromLedgerEntryChanges(t.FeeChanges)
	for i := range changes {
		changes[i].Reason = LedgerEntryChangeReasonFee
		changes[i].Transaction = t
	}
	return changes
}

func (t *LedgerTransaction) getTransactionChanges(ledgerEntryChanges xdr.LedgerEntryChanges) []Change {
	changes := GetChangesFromLedgerEntryChanges(ledgerEntryChanges)
	for i := range changes {
		changes[i].Reason = LedgerEntryChangeReasonTransaction
		changes[i].Transaction = t
	}
	return changes
}

// GetChanges returns a developer friendly representation of LedgerEntryChanges.
// It contains transaction changes and operation changes in that order. If the
// transaction failed with TxInternalError, operations and txChangesAfter are
// omitted. It doesn't support legacy TransactionMeta.V=0.
func (t *LedgerTransaction) GetChanges() ([]Change, error) {
	var changes []Change

	// Transaction meta
	switch t.UnsafeMeta.V {
	case 0:
		return changes, errors.New("TransactionMeta.V=0 not supported")
	case 1:
		v1Meta := t.UnsafeMeta.MustV1()
		// The var `txChanges` reflect the ledgerEntryChanges that are changed because of the transaction as a whole
		txChanges := t.getTransactionChanges(v1Meta.TxChanges)
		changes = append(changes, txChanges...)

		// Ignore operations meta if txInternalError https://github.com/stellar/go/issues/2111
		if t.txInternalError() && t.LedgerVersion <= 12 {
			return changes, nil
		}

		// These changes reflect the ledgerEntry changes that were caused by the operations in the transaction
		// Populate the operationInfo for these changes in the `Change` struct

		operationMeta := v1Meta.Operations
		//	operationMeta is a list of lists.
		//	Each element in operationMeta is a list of ledgerEntryChanges
		//	caused by the operation at that index of the element
		for opIdx := range operationMeta {
			opChanges := t.operationChanges(v1Meta.Operations, uint32(opIdx))
			changes = append(changes, opChanges...)
		}
	case 2, 3:
		var (
			txBeforeChanges, txAfterChanges xdr.LedgerEntryChanges
			operationMeta                   []xdr.OperationMeta
		)

		switch t.UnsafeMeta.V {
		case 2:
			v2Meta := t.UnsafeMeta.MustV2()
			txBeforeChanges = v2Meta.TxChangesBefore
			txAfterChanges = v2Meta.TxChangesAfter
			operationMeta = v2Meta.Operations
		case 3:
			v3Meta := t.UnsafeMeta.MustV3()
			txBeforeChanges = v3Meta.TxChangesBefore
			txAfterChanges = v3Meta.TxChangesAfter
			operationMeta = v3Meta.Operations
		default:
			panic("Invalid meta version, expected 2 or 3")
		}

		txChangesBefore := t.getTransactionChanges(txBeforeChanges)
		changes = append(changes, txChangesBefore...)

		// Ignore operations meta and txChangesAfter if txInternalError
		// https://github.com/stellar/go/issues/2111
		if t.txInternalError() && t.LedgerVersion <= 12 {
			return changes, nil
		}

		//	operationMeta is a list of lists.
		//	Each element in operationMeta is a list of ledgerEntryChanges
		//	caused by the operation at that index of the element
		for opIdx := range operationMeta {
			opChanges := t.operationChanges(operationMeta, uint32(opIdx))
			changes = append(changes, opChanges...)
		}

		txChangesAfter := t.getTransactionChanges(txAfterChanges)
		changes = append(changes, txChangesAfter...)
	default:
		return changes, errors.New("Unsupported TransactionMeta version")
	}

	return changes, nil
}

// GetOperation returns an operation by index.
func (t *LedgerTransaction) GetOperation(index uint32) (xdr.Operation, bool) {
	ops := t.Envelope.Operations()
	if int(index) >= len(ops) {
		return xdr.Operation{}, false
	}
	return ops[index], true
}

// GetOperationChanges returns a developer friendly representation of LedgerEntryChanges.
// It contains only operation changes.
func (t *LedgerTransaction) GetOperationChanges(operationIndex uint32) ([]Change, error) {
	if t.UnsafeMeta.V == 0 {
		return []Change{}, errors.New("TransactionMeta.V=0 not supported")
	}

	// Ignore operations meta if txInternalError https://github.com/stellar/go/issues/2111
	if t.txInternalError() && t.LedgerVersion <= 12 {
		return []Change{}, nil
	}

	var operationMeta []xdr.OperationMeta
	switch t.UnsafeMeta.V {
	case 1:
		operationMeta = t.UnsafeMeta.MustV1().Operations
	case 2:
		operationMeta = t.UnsafeMeta.MustV2().Operations
	case 3:
		operationMeta = t.UnsafeMeta.MustV3().Operations
	default:
		return []Change{}, errors.New("Unsupported TransactionMeta version")
	}

	return t.operationChanges(operationMeta, operationIndex), nil
}

func (t *LedgerTransaction) operationChanges(ops []xdr.OperationMeta, index uint32) []Change {
	if int(index) >= len(ops) {
		return []Change{}
	}

	operationMeta := ops[index]
	changes := GetChangesFromLedgerEntryChanges(operationMeta.Changes)

	for i := range changes {
		changes[i].Reason = LedgerEntryChangeReasonOperation
		changes[i].Transaction = t
		changes[i].OperationIndex = index
	}
	return changes
}

// GetDiagnosticEvents returns all contract events emitted by a given operation.
func (t *LedgerTransaction) GetDiagnosticEvents() ([]xdr.DiagnosticEvent, error) {
	return t.UnsafeMeta.GetDiagnosticEvents()
}
