package operation

import "fmt"

type RestoreFootprintDetail struct {
	Type             string   `json:"type"`
	LedgerKeyHash    []string `json:"ledger_key_hash"`
	ContractID       string   `json:"contract_id"`
	ContractCodeHash string   `json:"contract_code_hash"`
}

func (o *LedgerOperation) RestoreFootprintDetails() (RestoreFootprintDetail, error) {
	_, ok := o.Operation.Body.GetRestoreFootprintOp()
	if !ok {
		return RestoreFootprintDetail{}, fmt.Errorf("could not access RestoreFootprint info for this operation (index %d)", o.OperationIndex)
	}

	restoreFootprintDetail := RestoreFootprintDetail{
		Type:          "restore_footprint",
		LedgerKeyHash: o.Transaction.LedgerKeyHashesFromSorobanFootprint(),
	}

	var contractID string
	contractID, ok = o.Transaction.ContractIdFromTxEnvelope()
	if ok {
		restoreFootprintDetail.ContractID = contractID
	}

	var contractCodeHash string
	contractCodeHash, ok = o.Transaction.ContractCodeHashFromSorobanFootprint()
	if ok {
		restoreFootprintDetail.ContractCodeHash = contractCodeHash
	}

	return restoreFootprintDetail, nil
}
