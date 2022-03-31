package integration

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
)

func TestTransactionPreconditionsMinSeq(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{})
	master := itest.Master()

	masterAccount := itest.MasterAccount()
	currentAccountSeq, err := masterAccount.GetSequenceNumber()
	tt.NoError(err)

	// Ensure that the minSequence of the transaction is enough
	// but the sequence isn't
	txParams := txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{
			AccountID: masterAccount.GetAccountID(),
			Sequence:  currentAccountSeq + 100,
		},
		// Phony operation to run
		Operations: []txnbuild.Operation{&txnbuild.BumpSequence{
			BumpTo: currentAccountSeq + 10,
		}},
		BaseFee: txnbuild.MinBaseFee,
		Memo:    nil,
		Preconditions: txnbuild.Preconditions{
			TimeBounds:        txnbuild.NewInfiniteTimeout(),
			MinSequenceNumber: &currentAccountSeq,
		},
	}

	itest.MustSubmitTransaction(master, txParams)

}
