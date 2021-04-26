package codes

import (
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestForOperationResultCoversForAllOpTypes(t *testing.T) {
	for typ, s := range xdr.OperationTypeToStringMap {
		result := xdr.OperationResult{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationType(typ),
			},
		}
		f := func() {
			ForOperationResult(result)
		}
		// it must panic because the operation result is not set
		assert.Panics(t, f, s)
	}
	// make sure the check works for an unknown operation type
	result := xdr.OperationResult{
		Code: xdr.OperationResultCodeOpInner,
		Tr: &xdr.OperationResultTr{
			Type: xdr.OperationType(200000),
		},
	}
	f := func() {
		ForOperationResult(result)
	}
	// it doesn't panic because it doesn't branch out into the operation type
	assert.NotPanics(t, f)
}

func TestString(t *testing.T) {
	tests := []struct {
		Input    interface{}
		Expected string
		Err      error
	}{
		{xdr.TransactionResultCodeTxSuccess, "tx_success", nil},
		{xdr.OperationResultCodeOpBadAuth, "op_bad_auth", nil},
		{xdr.CreateAccountResultCodeCreateAccountLowReserve, "op_low_reserve", nil},
		{xdr.PaymentResultCodePaymentSrcNoTrust, "op_src_no_trust", nil},
		{xdr.SetOptionsResultCodeSetOptionsAuthRevocableRequired, "op_auth_revocable_required", nil},
		{xdr.ClawbackResultCodeClawbackNotClawbackEnabled, "op_not_clawback_enabled", nil},
		{0, "", ErrUnknownCode},
	}

	for _, test := range tests {
		actual, err := String(test.Input)

		if test.Err != nil {
			assert.NotNil(t, err)
			assert.Equal(t, test.Err.Error(), err.Error())
		} else {
			assert.Nil(t, err)
			assert.Equal(t, test.Expected, actual)
		}
	}
}

//TODO: op_inner refers to inner result code
//TODO: non op_inner uses the outer result code
//TODO: one test for each operation type
