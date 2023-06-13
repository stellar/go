package ingest

import (
	"fmt"

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
	FeeChanges xdr.LedgerEntryChanges
	UnsafeMeta xdr.TransactionMeta
}

func (t *LedgerTransaction) txInternalError() bool {
	return t.Result.Result.Result.Code == xdr.TransactionResultCodeTxInternalError
}

// GetFeeChanges returns a developer friendly representation of LedgerEntryChanges
// connected to fees.
func (t *LedgerTransaction) GetFeeChanges() []Change {
	return GetChangesFromLedgerEntryChanges(t.FeeChanges)
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
		txChanges := GetChangesFromLedgerEntryChanges(v1Meta.TxChanges)
		changes = append(changes, txChanges...)

		// Ignore operations meta if txInternalError https://github.com/stellar/go/issues/2111
		if t.txInternalError() {
			return changes, nil
		}

		for _, operationMeta := range v1Meta.Operations {
			opChanges := GetChangesFromLedgerEntryChanges(
				operationMeta.Changes,
			)
			changes = append(changes, opChanges...)
		}
	case 2, 3:
		var (
			beforeChanges, afterChanges xdr.LedgerEntryChanges
			operationMeta               []xdr.OperationMeta
		)

		switch t.UnsafeMeta.V {
		case 2:
			v2Meta := t.UnsafeMeta.MustV2()
			beforeChanges = v2Meta.TxChangesBefore
			afterChanges = v2Meta.TxChangesAfter
			operationMeta = v2Meta.Operations
		case 3:
			v3Meta := t.UnsafeMeta.MustV3()
			beforeChanges = v3Meta.TxChangesBefore
			afterChanges = v3Meta.TxChangesAfter
			operationMeta = v3Meta.Operations
		default:
			panic("Invalid meta version, expected 2 or 3")
		}

		txChangesBefore := GetChangesFromLedgerEntryChanges(beforeChanges)
		changes = append(changes, txChangesBefore...)

		// Ignore operations meta and txChangesAfter if txInternalError
		// https://github.com/stellar/go/issues/2111
		if t.txInternalError() {
			return changes, nil
		}

		for _, operationMeta := range operationMeta {
			opChanges := GetChangesFromLedgerEntryChanges(
				operationMeta.Changes,
			)
			changes = append(changes, opChanges...)
		}

		txChangesAfter := GetChangesFromLedgerEntryChanges(afterChanges)
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
	changes := []Change{}

	if t.UnsafeMeta.V == 0 {
		return changes, errors.New("TransactionMeta.V=0 not supported")
	}

	// Ignore operations meta if txInternalError https://github.com/stellar/go/issues/2111
	if t.txInternalError() {
		return changes, nil
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
		return changes, errors.New("Unsupported TransactionMeta version")
	}

	return operationChanges(operationMeta, operationIndex), nil
}

func operationChanges(ops []xdr.OperationMeta, index uint32) []Change {
	if int(index) >= len(ops) {
		return []Change{}
	}

	operationMeta := ops[index]
	return GetChangesFromLedgerEntryChanges(
		operationMeta.Changes,
	)
}

// GetDiagnosticEvents returns all contract events emitted by a given operation.
func (t *LedgerTransaction) GetDiagnosticEvents() ([]xdr.DiagnosticEvent, error) {
	// Ignore operations meta if txInternalError https://github.com/stellar/go/issues/2111
	if t.txInternalError() {
		return nil, nil
	}

	switch t.UnsafeMeta.V {
	case 1:
		return nil, nil
	case 2:
		return nil, nil
	case 3:
		diagnosticEvents := t.UnsafeMeta.MustV3().DiagnosticEvents
		if len(diagnosticEvents) > 0 {
			// all contract events and diag events for a single operation(by it's index in the tx) were available
			// in tx meta's DiagnosticEvents, no need to look anywhere else for events
			return diagnosticEvents, nil
		}

		contractEvents := t.UnsafeMeta.MustV3().Events
		if len(contractEvents) == 0 {
			// no events were present in this tx meta
			return nil, nil
		}

		// tx meta only provided contract events, no diagnostic events, we convert the contract
		// event to a diagnostic event, to fit the response interface.
		convertedDiagnosticEvents := make([]xdr.DiagnosticEvent, len(contractEvents))
		for i, event := range contractEvents {
			convertedDiagnosticEvents[i] = xdr.DiagnosticEvent{
				InSuccessfulContractCall: true,
				Event:                    event,
			}
		}
		return convertedDiagnosticEvents, nil
	default:
		return nil, fmt.Errorf("unsupported TransactionMeta version: %v", t.UnsafeMeta.V)
	}
}
