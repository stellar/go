package operation

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

type InvokeHostFunctionDetail struct {
	Function            string                `json:"function"`
	Type                string                `json:"type"`
	LedgerKeyHash       []string              `json:"ledger_key_hash"`
	ContractID          string                `json:"contract_id"`
	ContractCodeHash    string                `json:"contract_code_hash"`
	Parameters          []interface{}         `json:"parameters"`
	AssetBalanceChanges []BalanceChangeDetail `json:"asset_balance_changes"`
	From                string                `json:"from"`
	Address             string                `json:"address"`
	AssetCode           string                `json:"asset_code"`
	AssetIssuer         string                `json:"asset_issuer"`
	AssetType           string                `json:"asset_type"`
}

func (o *LedgerOperation) InvokeHostFunctionDetails() (InvokeHostFunctionDetail, error) {
	op, ok := o.Operation.Body.GetInvokeHostFunctionOp()
	if !ok {
		return InvokeHostFunctionDetail{}, fmt.Errorf("could not access InvokeHostFunction info for this operation (index %d)", o.OperationIndex)
	}

	invokeHostFunctionDetail := InvokeHostFunctionDetail{
		Function: op.HostFunction.Type.String(),
	}

	switch op.HostFunction.Type {
	case xdr.HostFunctionTypeHostFunctionTypeInvokeContract:
		invokeArgs := op.HostFunction.MustInvokeContract()
		args := make([]xdr.ScVal, 0, len(invokeArgs.Args)+2)
		args = append(args, xdr.ScVal{Type: xdr.ScValTypeScvAddress, Address: &invokeArgs.ContractAddress})
		args = append(args, xdr.ScVal{Type: xdr.ScValTypeScvSymbol, Sym: &invokeArgs.FunctionName})
		args = append(args, invokeArgs.Args...)

		invokeHostFunctionDetail.Type = "invoke_contract"

		contractId, err := invokeArgs.ContractAddress.String()
		if err != nil {
			return InvokeHostFunctionDetail{}, err
		}

		invokeHostFunctionDetail.LedgerKeyHash = o.Transaction.LedgerKeyHashesFromSorobanFootprint()
		invokeHostFunctionDetail.ContractID = contractId

		var contractCodeHash string
		contractCodeHash, ok = o.Transaction.ContractCodeHashFromSorobanFootprint()
		if ok {
			invokeHostFunctionDetail.ContractCodeHash = contractCodeHash
		}

		invokeHostFunctionDetail.Parameters, err = o.serializeParameters(args)
		if err != nil {
			return InvokeHostFunctionDetail{}, err
		}

		balanceChanges, err := o.parseAssetBalanceChangesFromContractEvents()
		if err != nil {
			return InvokeHostFunctionDetail{}, err
		}

		invokeHostFunctionDetail.AssetBalanceChanges = balanceChanges

	case xdr.HostFunctionTypeHostFunctionTypeCreateContract:
		args := op.HostFunction.MustCreateContract()

		invokeHostFunctionDetail.Type = "create_contract"

		preImageDetails, err := switchContractIdPreimage(args.ContractIdPreimage)
		if err != nil {
			return InvokeHostFunctionDetail{}, nil
		}

		o.getCreateContractDetails(&invokeHostFunctionDetail, preImageDetails)
	case xdr.HostFunctionTypeHostFunctionTypeUploadContractWasm:
		invokeHostFunctionDetail.Type = "upload_wasm"
		invokeHostFunctionDetail.LedgerKeyHash = o.Transaction.LedgerKeyHashesFromSorobanFootprint()

		var contractCodeHash string
		contractCodeHash, ok = o.Transaction.ContractCodeHashFromSorobanFootprint()
		if ok {
			invokeHostFunctionDetail.ContractCodeHash = contractCodeHash
		}
	case xdr.HostFunctionTypeHostFunctionTypeCreateContractV2:
		args := op.HostFunction.MustCreateContractV2()

		invokeHostFunctionDetail.Type = "create_contract_v2"

		preImageDetails, err := switchContractIdPreimage(args.ContractIdPreimage)
		if err != nil {
			return InvokeHostFunctionDetail{}, err
		}

		o.getCreateContractDetails(&invokeHostFunctionDetail, preImageDetails)

		// ConstructorArgs is a list of ScVals
		// This will initially be handled the same as InvokeContractParams until a different
		// model is found necessary.
		invokeHostFunctionDetail.Parameters, err = o.serializeParameters(args.ConstructorArgs)
		if err != nil {
			return InvokeHostFunctionDetail{}, err
		}
	default:
		return InvokeHostFunctionDetail{}, fmt.Errorf("unknown host function type: %s", op.HostFunction.Type)
	}

	return invokeHostFunctionDetail, nil
}

func (o *LedgerOperation) getCreateContractDetails(invokeHostFunctionDetail *InvokeHostFunctionDetail, preImageDetails PreImageDetails) {
	var ok bool
	invokeHostFunctionDetail.LedgerKeyHash = o.Transaction.LedgerKeyHashesFromSorobanFootprint()

	var contractID string
	contractID, ok = o.Transaction.ContractIdFromTxEnvelope()
	if ok {
		invokeHostFunctionDetail.ContractID = contractID
	}

	var contractCodeHash string
	contractCodeHash, ok = o.Transaction.ContractCodeHashFromSorobanFootprint()
	if ok {
		invokeHostFunctionDetail.ContractCodeHash = contractCodeHash
	}

	invokeHostFunctionDetail.From = preImageDetails.From
	invokeHostFunctionDetail.Address = preImageDetails.Address
	invokeHostFunctionDetail.AssetCode = preImageDetails.AssetCode
	invokeHostFunctionDetail.AssetIssuer = preImageDetails.AssetIssuer
	invokeHostFunctionDetail.AssetType = preImageDetails.AssetType
}
