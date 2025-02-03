package ingest

import "fmt"

type ExtendFootprintTtlDetail struct {
	Type             string   `json:"type"`
	ExtendTo         uint32   `json:"extend_to"`
	LedgerKeyHash    []string `json:"ledger_key_hash"`
	ContractID       string   `json:"contract_id"`
	ContractCodeHash string   `json:"contract_code_hash"`
}

func (o *LedgerOperation) ExtendFootprintTtlDetails() (ExtendFootprintTtlDetail, error) {
	op, ok := o.Operation.Body.GetExtendFootprintTtlOp()
	if !ok {
		return ExtendFootprintTtlDetail{}, fmt.Errorf("could not access ExtendFootprintTtl info for this operation (index %d)", o.OperationIndex)
	}

	extendFootprintTtlDetail := ExtendFootprintTtlDetail{
		Type:          "extend_footprint_ttl",
		ExtendTo:      uint32(op.ExtendTo),
		LedgerKeyHash: o.Transaction.LedgerKeyHashFromTxEnvelope(),
	}

	var contractID string
	contractID, ok = o.Transaction.contractIdFromTxEnvelope()
	if ok {
		extendFootprintTtlDetail.ContractID = contractID
	}

	var contractCodeHash string
	contractCodeHash, ok = o.Transaction.ContractCodeHashFromTxEnvelope()
	if ok {
		extendFootprintTtlDetail.ContractCodeHash = contractCodeHash
	}

	return extendFootprintTtlDetail, nil
}
