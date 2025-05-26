package xdr

import (
	"fmt"
)

func (t *TransactionMeta) GetContractEventsForOperation(opIndex uint32) ([]ContractEvent, error) {
	switch t.V {
	case 1, 2:
		return nil, nil
	// For TxMetaV3, events appear in the TxMetaV3.SorobanMeta.Events, and we dont need to rely on opIndex
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

// GetDiagnosticEvents returns the diagnostic events as they appear in the TransactionMeta
// Please note that, depending on the configuration with which txMeta may be generated,
// it is possible that, for smart contract transactions, the list of generated diagnostic events MAY include contract events as well
// Users of this function (horizon, rpc, etc) should be careful not to double count diagnostic events and contract events in that case
func (t *TransactionMeta) GetDiagnosticEvents() ([]DiagnosticEvent, error) {
	switch t.V {
	case 1, 2:
		return nil, nil
	case 3:
		sorobanMeta := t.MustV3().SorobanMeta
		if sorobanMeta == nil {
			return nil, nil
		}
		return sorobanMeta.DiagnosticEvents, nil
	case 4:
		return t.MustV4().DiagnosticEvents, nil
	default:
		return nil, fmt.Errorf("unsupported TransactionMeta version: %v", t.V)
	}
}

// GetTransactionEvents returns the xdr.transactionEvents present in the ledger.
// For TxMetaVersions < 4, they will be empty

func (t *TransactionMeta) GetTransactionEvents() ([]TransactionEvent, error) {
	switch t.V {
	case 1, 2, 3:
		return nil, nil
	case 4:
		return t.MustV4().Events, nil
	default:
		return nil, fmt.Errorf("unsupported TransactionMeta version: %v", t.V)
	}
}
