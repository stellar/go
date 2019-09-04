package txnbuild

import (
	"testing"

	"github.com/stellar/go/network"
	"github.com/stretchr/testify/assert"
)

func TestAccountMergeValidate(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484298))

	accountMerge := AccountMerge{
		Destination: "GBAV",
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&accountMerge},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	err := tx.Build()
	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.AccountMerge struct: Field: Destination, Error: public key is invalid"
		assert.Contains(t, err.Error(), expected)
	}
}
