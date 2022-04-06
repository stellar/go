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
	tx := itest.MustSubmitTransaction(master, txParams)

	txHistory, err := itest.Client().TransactionDetail(tx.Hash)
	assert.NoError(t, err)
	assert.Equal(t, txHistory.Preconditions.MinAccountSequence, strconv.FormatInt(*txParams.Preconditions.MinSequenceNumber, 10))
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
	tx := itest.MustSubmitTransaction(master, txParams)

	txHistory, err := itest.Client().TransactionDetail(tx.Hash)
	assert.NoError(t, err)
	historyMaxTime, err := time.Parse(time.RFC3339, txHistory.Preconditions.Timebounds.MaxTime)
	assert.NoError(t, err)
	historyMinTime, err := time.Parse(time.RFC3339, txHistory.Preconditions.Timebounds.MinTime)
	assert.NoError(t, err)

	assert.Equal(t, historyMaxTime.UTC().Unix(), txParams.Preconditions.TimeBounds.MaxTime)
	assert.Equal(t, historyMinTime.UTC().Unix(), txParams.Preconditions.TimeBounds.MinTime)
}

func TestTransactionPreconditionsExtraSigners(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{})
	if itest.GetEffectiveProtocolVersion() < 19 {
		t.Skip("Can't run with protocol < 19")
	}
	master := itest.Master()
	masterAccount := itest.MasterAccount()

	// create a new signed payload signer
	addtlSigners, addtlAccounts := itest.CreateAccounts(1, "1000")

	// build a tx with seqnum based on master.seqNum+1 as source account
	latestMasterAccount := itest.MustGetAccount(master)
	currentAccountSeq, err := latestMasterAccount.GetSequenceNumber()
	tt.NoError(err)
	txParams := buildTXParams(master, masterAccount, currentAccountSeq, currentAccountSeq+1)

	// this errors because the tx preconditions require extra signer that
	// didn't sign this tx
	txParams.Preconditions.ExtraSigners = []string{addtlAccounts[0].GetAccountID()}
	_, err = itest.SubmitMultiSigTransaction([]*keypair.Full{master}, txParams)
	tt.Error(err)

	// Now the transaction should be submitted without problems, the extra signer specified
	// has also signed this transaction.
	txParams.Preconditions.ExtraSigners = []string{addtlAccounts[0].GetAccountID()}
	tx, err := itest.SubmitMultiSigTransaction([]*keypair.Full{master, addtlSigners[0]}, txParams)
	tt.NoError(err)

	txHistory, err := itest.Client().TransactionDetail(tx.Hash)
	assert.NoError(t, err)
	assert.ElementsMatch(t, txHistory.Preconditions.ExtraSigners, txParams.Preconditions.ExtraSigners)
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
