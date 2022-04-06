package integration

import (
	"strconv"
	"testing"
	"time"

	sdk "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
)

func TestTransactionPreconditionsMinSeq(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{})
	if itest.GetEffectiveProtocolVersion() < 19 {
		t.Skip("Can't run with protocol < 19")
	}
	master := itest.Master()
	masterAccount := itest.MasterAccount()
	currentAccountSeq, err := masterAccount.GetSequenceNumber()
	tt.NoError(err)

	// Ensure that the minSequence of the transaction is enough
	// but the sequence isn't
	txParams := buildTXParams(master, masterAccount, currentAccountSeq, currentAccountSeq+100)

	// this errors because the tx.seqNum is more than +1 from sourceAccoubnt.seqNum
	_, err = itest.SubmitTransaction(master, txParams)
	tt.Error(err)

	// Now the transaction should be submitted without problems
	txParams.Preconditions.MinSequenceNumber = &currentAccountSeq
	itest.MustSubmitTransaction(master, txParams)
}

func TestTransactionPreconditionsTimeBounds(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{})
	if itest.GetEffectiveProtocolVersion() < 19 {
		t.Skip("Can't run with protocol < 19")
	}
	master := itest.Master()
	masterAccount := itest.MasterAccount()
	currentAccountSeq, err := masterAccount.GetSequenceNumber()
	tt.NoError(err)
	txParams := buildTXParams(master, masterAccount, currentAccountSeq, currentAccountSeq+1)

	// this errors because the min time is > current tx submit time
	txParams.Preconditions.TimeBounds.MinTime = time.Now().Unix() + 3600
	txParams.Preconditions.TimeBounds.MaxTime = time.Now().Unix() + 7200
	_, err = itest.SubmitTransaction(master, txParams)
	tt.Error(err)

	// this errors because the max time is < current tx submit time
	txParams.Preconditions.TimeBounds.MinTime = 0
	txParams.Preconditions.TimeBounds.MaxTime = time.Now().Unix() - 3600
	_, err = itest.SubmitTransaction(master, txParams)
	tt.Error(err)

	// Now the transaction should be submitted without problems, min < current tx submit time < max
	txParams.Preconditions.TimeBounds.MinTime = time.Now().Unix() - 3600
	txParams.Preconditions.TimeBounds.MaxTime = time.Now().Unix() + 3600
	itest.MustSubmitTransaction(master, txParams)
}

func buildTXParams(master *keypair.Full, masterAccount txnbuild.Account, sourceAccountSeq int64, txSequence int64) txnbuild.TransactionParams {

	ops := []txnbuild.Operation{
		&txnbuild.BumpSequence{
			BumpTo: sourceAccountSeq + 10,
		},
	}

	return txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{
			AccountID: masterAccount.GetAccountID(),
			Sequence:  txSequence,
		},
		// Phony operation to run
		Operations: ops,
		BaseFee:    txnbuild.MinBaseFee,
		Memo:       nil,
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewInfiniteTimeout(),
		},
	}
}

func TestTransactionPreconditionsAccountFields(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{})
	if itest.GetEffectiveProtocolVersion() < 19 {
		t.Skip("Can't run with protocol < 19")
	}
	master := itest.Master()
	masterAccount := itest.MasterAccount()
	currentAccountSeq, err := masterAccount.GetSequenceNumber()
	tt.NoError(err)

	tx := itest.MustSubmitOperations(masterAccount, master,
		&txnbuild.BumpSequence{
			BumpTo: currentAccountSeq + 10,
		},
	)

	// refresh master account
	account, err := itest.Client().AccountDetail(sdk.AccountRequest{AccountID: master.Address()})
	assert.NoError(t, err)

	// Check the new fields
	tt.Equal(uint32(tx.Ledger), account.SequenceLedger)
	tt.Equal(strconv.FormatInt(tx.LedgerCloseTime.Unix(), 10), account.SequenceTime)
}
