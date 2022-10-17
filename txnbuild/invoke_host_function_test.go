package txnbuild

import (
	"testing"

	"github.com/stellar/go/xdr"

	"github.com/stretchr/testify/assert"
)

func TestCreateInvokeHostFunctionValid(t *testing.T) {
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp1.Address(), int64(41137196761100))

	invokeHostFunctionOp := InvokeHostFunction{
		Function:      xdr.HostFunctionHostFnInvokeContract,
		Parameters:    xdr.ScVec{},
		Footprint:     xdr.LedgerFootprint{},
		SourceAccount: sourceAccount.AccountID,
	}

	assert.NoError(t, invokeHostFunctionOp.Validate())
}

func TestCreateInvokeHostFunctionInvalid(t *testing.T) {
	invokeHostFunctionOp := InvokeHostFunction{
		Function:      xdr.HostFunctionHostFnInvokeContract,
		Parameters:    xdr.ScVec{},
		Footprint:     xdr.LedgerFootprint{},
		SourceAccount: "invalid account value",
	}

	assert.Error(t, invokeHostFunctionOp.Validate())
}

func TestInvokeHostFunctionRoundTrip(t *testing.T) {
	val := xdr.Int32(4)
	code := []byte{1, 2, 3, 4}
	obj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoContractCode,
		ContractCode: &xdr.ScContractCode{
			Type: xdr.ScContractCodeTypeSccontractCodeWasm,
			Wasm: &code,
		},
	}
	i64 := xdr.Int64(45)
	rwObj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoI64,
		I64:  &i64,
	}
	invokeHostFunctionOp := &InvokeHostFunction{
		Function: xdr.HostFunctionHostFnInvokeContract,
		Parameters: xdr.ScVec{
			xdr.ScVal{
				Type: xdr.ScValTypeScvI32,
				I32:  &val,
			},
		},
		Footprint: xdr.LedgerFootprint{
			ReadOnly: []xdr.LedgerKey{
				{
					Type: xdr.LedgerEntryTypeContractData,
					ContractData: &xdr.LedgerKeyContractData{
						ContractId: xdr.Hash{1, 2, 3},
						Key: xdr.ScVal{
							Type: xdr.ScValTypeScvObject,
							Obj:  &obj,
						},
					},
				},
			},
			ReadWrite: []xdr.LedgerKey{
				{
					Type: xdr.LedgerEntryTypeContractData,
					ContractData: &xdr.LedgerKeyContractData{
						ContractId: xdr.Hash{1, 2, 3},
						Key: xdr.ScVal{
							Type: xdr.ScValTypeScvObject,
							Obj:  &rwObj,
						},
					},
				},
			},
		},
		SourceAccount: "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
	}
	testOperationsMarshallingRoundtrip(t, []Operation{invokeHostFunctionOp}, false)

	// with muxed accounts
	invokeHostFunctionOp.SourceAccount = "MA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVAAAAAAAAAAAAAJLK"
	testOperationsMarshallingRoundtrip(t, []Operation{invokeHostFunctionOp}, true)
}
