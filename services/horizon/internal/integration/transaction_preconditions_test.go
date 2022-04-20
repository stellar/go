package integration

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	sdk "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
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
	txParams := buildTXParams(master, masterAccount, currentAccountSeq+100)

	// this errors because the tx.seqNum is more than +1 from sourceAccoubnt.seqNum
	_, err = itest.SubmitTransaction(master, txParams)
	tt.Error(err)

	// Now the transaction should be submitted without problems
	txParams.Preconditions.MinSequenceNumber = &currentAccountSeq
	tx := itest.MustSubmitTransaction(master, txParams)

	txHistory, err := itest.Client().TransactionDetail(tx.Hash)
	assert.NoError(t, err)
	assert.Equal(t, txHistory.Preconditions.MinAccountSequence, strconv.FormatInt(*txParams.Preconditions.MinSequenceNumber, 10))

	// Test the transaction submission queue by sending transactions out of order
	// and making sure they are all executed properly
	masterAccount = itest.MasterAccount()
	currentAccountSeq, err = masterAccount.GetSequenceNumber()
	tt.NoError(err)

	seqs := []struct {
		minSeq int64
		seq    int64
	}{
		{0, currentAccountSeq + 9},                 // sent first, executed second
		{0, currentAccountSeq + 10},                // sent second, executed third
		{currentAccountSeq, currentAccountSeq + 8}, // sent third, executed first
	}

	// Send the transactions in parallel since otherwise they are admitted sequentially
	var results []horizon.Transaction
	var resultsMx sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(seqs))
	for _, s := range seqs {
		sLocal := s
		go func() {
			params := buildTXParams(master, masterAccount, sLocal.seq)
			if sLocal.minSeq > 0 {
				params.Preconditions.MinSequenceNumber = &sLocal.minSeq
			}
			result := itest.MustSubmitTransaction(master, params)
			resultsMx.Lock()
			results = append(results, result)
			resultsMx.Unlock()
			wg.Done()
		}()
		// Space out requests to ensure the queue receives the transactions
		// in the planned order
		time.Sleep(time.Millisecond * 50)
	}
	wg.Wait()

	tt.Len(results, len(seqs))
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
	txParams := buildTXParams(master, masterAccount, currentAccountSeq+1)

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
	historyMaxTime, err := time.Parse(time.RFC3339, txHistory.Preconditions.TimeBounds.MaxTime)
	assert.NoError(t, err)
	historyMinTime, err := time.Parse(time.RFC3339, txHistory.Preconditions.TimeBounds.MinTime)
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
	txParams := buildTXParams(master, masterAccount, currentAccountSeq+1)

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

func TestTransactionPreconditionsLedgerBounds(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{})
	if itest.GetEffectiveProtocolVersion() < 19 {
		t.Skip("Can't run with protocol < 19")
	}
	master := itest.Master()
	masterAccount := itest.MasterAccount()
	currentAccountSeq, err := masterAccount.GetSequenceNumber()
	tt.NoError(err)

	// build a tx with seqnum based on master.seqNum+1 as source account
	txParams := buildTXParams(master, masterAccount, currentAccountSeq+1)

	coreSequence, err := itest.GetCurrentCoreLedgerSequence()
	tt.NoError(err)

	// this txsub will error because the tx preconditions require a min ledger sequence number that
	// hasn't been realized yet on the network ledger
	txParams.Preconditions.LedgerBounds = &txnbuild.LedgerBounds{
		MinLedger: uint32(coreSequence + 1000),
		MaxLedger: uint32(coreSequence + 2000),
	}
	_, err = itest.SubmitMultiSigTransaction([]*keypair.Full{master}, txParams)
	tt.Error(err)

	txParams.Preconditions.LedgerBounds = &txnbuild.LedgerBounds{
		MinLedger: uint32(coreSequence - 1),
		MaxLedger: uint32(coreSequence + 2000),
	}
	// Now the transaction should be submitted without problems, the latest network
	// ledger sequence should be above the min and below the max of preconditions ledgerbounds
	tx, err := itest.SubmitMultiSigTransaction([]*keypair.Full{master}, txParams)
	tt.NoError(err)

	//verify roundtrip to network and back through the horizon api returns same precondition values
	txHistory, err := itest.Client().TransactionDetail(tx.Hash)
	assert.NoError(t, err)
	assert.Equal(t, txHistory.Preconditions.LedgerBounds.MaxLedger, txParams.Preconditions.LedgerBounds.MaxLedger)
	assert.Equal(t, txHistory.Preconditions.LedgerBounds.MinLedger, txParams.Preconditions.LedgerBounds.MinLedger)
}

func TestTransactionPreconditionsMinSequenceNumberAge(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{})
	if itest.GetEffectiveProtocolVersion() < 19 {
		t.Skip("Can't run with protocol < 19")
	}
	master := itest.Master()
	masterAccount := itest.MasterAccount()
	currentAccountSeq, err := masterAccount.GetSequenceNumber()
	tt.NoError(err)
	latestMasterAccount := itest.MustGetAccount(master)

	ledgerRequest := sdk.LedgerRequest{Order: sdk.OrderDesc, Limit: 1}
	ledgers, err := itest.Client().Ledgers(ledgerRequest)
	tt.NoError(err)
	tt.Len(ledgers.Embedded.Records, 1)

	// gather up the current sequence times
	signedAcctSeqTime, err := strconv.ParseInt(latestMasterAccount.SequenceTime, 10, 64)
	tt.NoError(err)
	tt.GreaterOrEqual(signedAcctSeqTime, int64(0))
	acctSeqTime := uint64(signedAcctSeqTime)
	networkSeqTime := uint64(ledgers.Embedded.Records[0].ClosedAt.UTC().Unix())

	// build a tx with seqnum based on master.seqNum+1 as source account
	txParams := buildTXParams(master, masterAccount, currentAccountSeq+1)

	// this txsub will error because the tx preconditions require a min sequence age
	// which has been set 10000 seconds greater than the current difference between
	// network ledger sequence time and account sequnece time
	txParams.Preconditions.MinSequenceNumberAge = networkSeqTime - acctSeqTime + 10000
	_, err = itest.SubmitMultiSigTransaction([]*keypair.Full{master}, txParams)
	tt.Error(err)

	txParams.Preconditions.MinSequenceNumberAge = networkSeqTime - acctSeqTime - 1
	// Now the transaction should be submitted without problems, the min sequence age
	// is set to be one second less then the current difference between network time and account sequence time.
	tx, err := itest.SubmitMultiSigTransaction([]*keypair.Full{master}, txParams)
	tt.NoError(err)

	//verify roundtrip to network and back through the horizon api returns same precondition values
	txHistory, err := itest.Client().TransactionDetail(tx.Hash)
	assert.NoError(t, err)
	assert.EqualValues(t, txHistory.Preconditions.MinAccountSequenceAge,
		fmt.Sprint(uint64(txParams.Preconditions.MinSequenceNumberAge)))
}

func TestTransactionPreconditionsMinSequenceNumberLedgerGap(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{})
	if itest.GetEffectiveProtocolVersion() < 19 {
		t.Skip("Can't run with protocol < 19")
	}
	master := itest.Master()
	masterAccount := itest.MasterAccount()
	currentAccountSeq, err := masterAccount.GetSequenceNumber()
	tt.NoError(err)

	// gather up the current sequence number
	networkLedger, err := itest.GetCurrentCoreLedgerSequence()
	tt.NoError(err)

	// build a tx with seqnum based on master.seqNum+1 as source account
	txParams := buildTXParams(master, masterAccount, currentAccountSeq+1)

	// this txsub will error because the tx preconditions require a min sequence gap
	// which has been set 10000 sequnce numbers greater than the current difference between
	// network ledger sequence and account sequnece numbers
	txParams.Preconditions.MinSequenceNumberLedgerGap = uint32(int64(networkLedger) - currentAccountSeq + 10000)
	_, err = itest.SubmitMultiSigTransaction([]*keypair.Full{master}, txParams)
	tt.Error(err)

	txParams.Preconditions.MinSequenceNumberLedgerGap = uint32(int64(networkLedger) - currentAccountSeq - 1)
	// Now the transaction should be submitted without problems, the min sequence gap
	// is set to be one less then the current difference between network sequence and account sequence number.
	tx, err := itest.SubmitMultiSigTransaction([]*keypair.Full{master}, txParams)
	tt.NoError(err)

	//verify roundtrip to network and back through the horizon api returns same precondition values
	txHistory, err := itest.Client().TransactionDetail(tx.Hash)
	assert.NoError(t, err)
	assert.Equal(t, txHistory.Preconditions.MinAccountSequenceLedgerGap, txParams.Preconditions.MinSequenceNumberLedgerGap)
}

func buildTXParams(master *keypair.Full, masterAccount txnbuild.Account, txSequence int64) txnbuild.TransactionParams {

	return txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{
			AccountID: masterAccount.GetAccountID(),
			Sequence:  txSequence,
		},
		// Phony operation to run
		Operations: []txnbuild.Operation{
			&txnbuild.Payment{
				Destination: master.Address(),
				Amount:      "10",
				Asset:       txnbuild.NativeAsset{},
			},
		},
		BaseFee: txnbuild.MinBaseFee,
		Memo:    nil,
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewInfiniteTimeout(),
		},
	}
}

func TestTransactionPreconditionsAccountV3Fields(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{})
	if itest.GetEffectiveProtocolVersion() < 19 {
		t.Skip("Can't run with protocol < 19")
	}
	master := itest.Master()
	masterAccount := itest.MasterAccount()

	// Submit phony operation
	tx := itest.MustSubmitOperations(masterAccount, master,
		&txnbuild.Payment{
			Destination: master.Address(),
			Amount:      "10",
			Asset:       txnbuild.NativeAsset{},
		},
	)

	// refresh master account
	account, err := itest.Client().AccountDetail(sdk.AccountRequest{AccountID: master.Address()})
	assert.NoError(t, err)

	// Check that the account response has the new AccountV3 fields
	tt.Equal(uint32(tx.Ledger), account.SequenceLedger)
	tt.Equal(strconv.FormatInt(tx.LedgerCloseTime.Unix(), 10), account.SequenceTime)
}
