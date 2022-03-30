package integration

import (
	"os"
	"testing"

	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
)

var protocol19Config = integration.Config{
	ProtocolVersion: 19,
	CoreDockerImage: "stellar/stellar-core:18.4.1-875.95d896a49.focal-v19unsafe",
}

func TestTransactionPreconditionsMinSeq(t *testing.T) {
	if os.Getenv("HORIZON_INTEGRATION_ENABLE_NEXT_PROTOCOL") != "" {
		t.Skip()
	}
	tt := assert.New(t)
	itest := integration.NewTest(t, protocol19Config)
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
		BaseFee: 0,
		Memo:    nil,
		Preconditions: txnbuild.Preconditions{
			MinSequenceNumber: &currentAccountSeq,
		},
	}

	itest.MustSubmitTransaction(master, txParams)

}
