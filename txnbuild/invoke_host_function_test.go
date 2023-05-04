package txnbuild

import (
	"testing"

	"github.com/stellar/go/xdr"

	"github.com/stretchr/testify/assert"
)

func TestCreateInvokeHostFunctionValid(t *testing.T) {
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp1.Address(), int64(41137196761100))

	invokeHostFunctionOp := InvokeHostFunctions{
		Functions: []xdr.HostFunction{
			{
				Args: xdr.HostFunctionArgs{
					Type:           xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
					InvokeContract: &xdr.ScVec{},
				},
				Auth: []xdr.ContractAuth{},
			},
		},
		SourceAccount: sourceAccount.AccountID,
	}

	assert.NoError(t, invokeHostFunctionOp.Validate())
}

func TestCreateInvokeHostFunctionInvalid(t *testing.T) {
	invokeHostFunctionOp := InvokeHostFunctions{
		Functions: []xdr.HostFunction{
			{
				Args: xdr.HostFunctionArgs{
					Type:           xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
					InvokeContract: &xdr.ScVec{},
				},
				Auth: []xdr.ContractAuth{},
			},
		},
		SourceAccount: "invalid account value",
	}

	assert.Error(t, invokeHostFunctionOp.Validate())
}

func TestInvokeHostFunctionRoundTrip(t *testing.T) {
	val := xdr.Int32(4)
	wasmId := xdr.Hash{1, 2, 3, 4}
	i64 := xdr.Int64(45)
	accountId := xdr.MustAddress("GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H")
	invokeHostFunctionOp := &InvokeHostFunctions{
		Functions: []xdr.HostFunction{
			{
				Args: xdr.HostFunctionArgs{
					Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
					InvokeContract: &xdr.ScVec{
						xdr.ScVal{
							Type: xdr.ScValTypeScvI32,
							I32:  &val,
						},
					},
				},
				Auth: []xdr.ContractAuth{
					{
						AddressWithNonce: &xdr.AddressWithNonce{
							Address: xdr.ScAddress{
								Type:      xdr.ScAddressTypeScAddressTypeAccount,
								AccountId: &accountId,
							},
							Nonce: 0,
						},
						RootInvocation: xdr.AuthorizedInvocation{
							ContractId:     xdr.Hash{0xaa, 0xbb},
							FunctionName:   "foo",
							Args:           nil,
							SubInvocations: nil,
						},
						SignatureArgs: nil,
					},
				},
			},
		},
		Ext: xdr.TransactionExt{
			V: 1,
			SorobanData: &xdr.SorobanTransactionData{
				Resources: xdr.SorobanResources{
					Footprint: xdr.LedgerFootprint{
						ReadOnly: []xdr.LedgerKey{
							{
								Type: xdr.LedgerEntryTypeContractData,
								ContractData: &xdr.LedgerKeyContractData{
									ContractId: xdr.Hash{1, 2, 3},
									Key: xdr.ScVal{
										Type: xdr.ScValTypeScvContractExecutable,
										Exec: &xdr.ScContractExecutable{
											Type:   xdr.ScContractExecutableTypeSccontractExecutableWasmRef,
											WasmId: &wasmId,
										},
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
										Type: xdr.ScValTypeScvI64,
										I64:  &i64,
									},
								},
							},
						},
					},
					Instructions:              0,
					ReadBytes:                 0,
					WriteBytes:                0,
					ExtendedMetaDataSizeBytes: 0,
				},
				RefundableFee: 1,
				Ext: xdr.ExtensionPoint{
					V: 0,
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
