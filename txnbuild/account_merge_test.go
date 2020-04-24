package txnbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccountMergeValidate(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484298))

	accountMerge := AccountMerge{
		Destination: "GBAV",
	}

	_, err := NewTransaction(
		TransactionParams{
			SourceAccount: &sourceAccount,
			Operations:    []Operation{&accountMerge},
			Timebounds:    NewInfiniteTimeout(),
			BaseFee:       MinBaseFee,
		},
	)
	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.AccountMerge operation: Field: Destination, Error: GBAV is not a valid stellar public key"
		assert.Contains(t, err.Error(), expected)
	}
}
