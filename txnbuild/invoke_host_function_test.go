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

func TestCreateInvokeHostFunctionInValid(t *testing.T) {
	invokeHostFunctionOp := InvokeHostFunction{
		Function:      xdr.HostFunctionHostFnInvokeContract,
		Parameters:    xdr.ScVec{},
		Footprint:     xdr.LedgerFootprint{},
		SourceAccount: "invalid account value",
	}

	assert.Error(t, invokeHostFunctionOp.Validate())
}
