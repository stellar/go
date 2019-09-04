package txnbuild

import (
	"testing"

	"github.com/stellar/go/network"
	"github.com/stretchr/testify/assert"
)

func TestBumpSequenceValidate(t *testing.T) {
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp1.Address(), int64(9606132444168199))

	bumpSequence := BumpSequence{
		BumpTo: -10,
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&bumpSequence},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	err := tx.Build()
	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.BumpSequence struct: Field: BumpTo, Error: value should be positve or zero"
		assert.Contains(t, err.Error(), expected)
	}
}
