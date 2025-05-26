package xdr

import (
	"fmt"
)

func (t *TransactionMeta) GetContractEvents(opIndex uint32) ([]ContractEvent, error) {
	switch t.V {
	case 1, 2:
		return nil, nil
	// For TxMetaV3, events appear in the TxMetaV3.SorobanMeta.Events
	case 3:
		return t.MustV3().SorobanMeta.Events, nil
	// TxMetaV4 includes unified CAP-67 events that appear at the operation level
	// To fetch soroban contract events from TxMetaV4, you will need to pass in the operationIndex 0.
	case 4:
		return t.MustV4().Operations[opIndex].Events, nil
	default:
		return nil, fmt.Errorf("unsupported TransactionMeta version: %v", t.V)
	}
}

// GetDiagnosticEventsV3 returns all contract events emitted by a Soroban Transaction in TxMetaV3
/*
	In TransactionMetaV3, soroban transaction events and contract events appear in the SorobanMeta struct, i.e. at the top level
	In TransactionMetaV4 and onwards, there is a more granular breakdown, becuase of CAP-67 unified events
	- Contract events will also be present in the "operation []OperationMetaV2" in  structure.
	- Classic operations will also have contract events.
	- For smart contract transactions, contract events are going to show up in its respective operation index in []OperationMetaV2
	- Additionally, if its a smart contract transaction, the contract events will also be included in the "DiagnosticEvents []DiagnosticEvent" structure
*/
func (t *TransactionMeta) GetDiagnosticEventsV3() []DiagnosticEvent {
	sorobanMeta := t.MustV3().SorobanMeta
	if sorobanMeta == nil {
		return nil
	}

	diagnosticEvents := sorobanMeta.DiagnosticEvents
	if len(diagnosticEvents) > 0 {
		// all contract events and diag events for a single operation(by its index in the tx) were available
		// in tx meta's DiagnosticEvents, no need to look anywhere else for events
		return diagnosticEvents
	}

	// tx meta only provided contract events, no diagnostic events, we convert the contract
	// event to a diagnostic event, to fit the response interface.
	contractEvents := sorobanMeta.Events
	convertedDiagnosticEvents := make([]DiagnosticEvent, len(contractEvents))
	for i, event := range contractEvents {
		convertedDiagnosticEvents[i] = DiagnosticEvent{
			InSuccessfulContractCall: true,
			Event:                    event,
		}
	}
	return convertedDiagnosticEvents
}

// TODO: add comments
func (t *TransactionMeta) GetDiagnosticEventsV4() []DiagnosticEvent {
	return t.MustV4().DiagnosticEvents
}

// TODO: add comments
func (t *TransactionMeta) GetTransactionEvents() []TransactionEvent {
	switch t.V {
	case 1, 2, 3:
		return nil
	case 4:
		return t.MustV4().Events
	default:
		panic(fmt.Errorf("unsupported TransactionMeta version: %v", t.V))
	}
}
