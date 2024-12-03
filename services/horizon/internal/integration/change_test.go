package integration

import (
	"context"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"io"
	"sort"
	"testing"
	"time"
)

func TestProtocolUpgradeChanges(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{SkipHorizonStart: true})

	upgradedLedgerAppx, _ := itest.GetUpgradedLedgerSeqAppx()
	waitForLedgerInArchive(t, 15*time.Second, upgradedLedgerAppx)

	ledgerSeqToLedgers := getLedgers(itest, 2, upgradedLedgerAppx)

	// It is important to find the "exact" ledger which is representative of protocol upgrade
	// and the one before it, to check for upgrade related changes
	upgradedLedgerSeq := getExactUpgradedLedgerSeq(ledgerSeqToLedgers, itest.Config().ProtocolVersion)
	prevLedgerToUpgrade := upgradedLedgerSeq - 1

	prevLedgerChanges := getChangesFromLedger(itest, ledgerSeqToLedgers[prevLedgerToUpgrade])
	prevLedgerChangeMap := changeReasonCountMap(prevLedgerChanges)
	upgradedLedgerChanges := getChangesFromLedger(itest, ledgerSeqToLedgers[upgradedLedgerSeq])
	upgradedLedgerChangeMap := changeReasonCountMap(upgradedLedgerChanges)

	tt.Zero(prevLedgerChangeMap[ingest.LedgerEntryChangeReasonUpgrade])
	tt.NotZero(upgradedLedgerChangeMap[ingest.LedgerEntryChangeReasonUpgrade])
	for _, change := range upgradedLedgerChanges {
		tt.Equal(change.Ledger.LedgerSequence(), upgradedLedgerSeq)
		tt.Empty(change.Transaction)
		tt.NotEmpty(change.LedgerUpgrade)
	}
}

func TestOneTxOneOperationChanges(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{})

	master := itest.Master()
	keys, _ := itest.CreateAccounts(2, "1000")
	srcAcc, destAcc := keys[0], keys[1]

	operation := txnbuild.Payment{
		SourceAccount: srcAcc.Address(),
		Destination:   destAcc.Address(),
		Asset:         txnbuild.NativeAsset{},
		Amount:        "900",
	}
	txResp, err := itest.SubmitMultiSigOperations(itest.MasterAccount(), []*keypair.Full{master, srcAcc}, &operation)
	tt.NoError(err)

	ledgerSeq := uint32(txResp.Ledger)
	waitForLedgerInArchive(t, 15*time.Second, ledgerSeq)

	ledger := getLedgers(itest, ledgerSeq, ledgerSeq)[ledgerSeq]
	changes := getChangesFromLedger(itest, ledger)

	reasonCntMap := changeReasonCountMap(changes)
	tt.Equal(2, reasonCntMap[ingest.LedgerEntryChangeReasonOperation])
	tt.Equal(1, reasonCntMap[ingest.LedgerEntryChangeReasonTransaction])
	tt.Equal(1, reasonCntMap[ingest.LedgerEntryChangeReasonFee])

	reasonToChangeMap := changeReasonToChangeMap(changes)
	// Assert Transaction Hash and Ledger Sequence are accurate in all changes
	for _, change := range changes {
		tt.Equal(change.Transaction.Hash.HexString(), txResp.Hash)
		tt.Equal(change.Transaction.Ledger.LedgerSequence(), ledgerSeq)
	}

	accountFromEntry := func(e *xdr.LedgerEntry) xdr.AccountEntry {
		return e.Data.MustAccount()
	}

	changeForAccount := func(changes []ingest.Change, target string) ingest.Change {
		for _, change := range changes {
			acc := change.Pre.Data.MustAccount()
			if acc.AccountId.Address() == target {
				return change
			}
		}
		return ingest.Change{}
	}

	feeRelatedChange := reasonToChangeMap[ingest.LedgerEntryChangeReasonFee][0]
	txRelatedChange := reasonToChangeMap[ingest.LedgerEntryChangeReasonTransaction][0]
	operationChanges := reasonToChangeMap[ingest.LedgerEntryChangeReasonOperation]

	tt.Equal(accountFromEntry(feeRelatedChange.Pre).AccountId.Address(), master.Address())
	tt.Equal(accountFromEntry(txRelatedChange.Pre).AccountId.Address(), master.Address())
	tt.True(containsAccount(operationChanges, srcAcc.Address()))
	tt.True(containsAccount(operationChanges, destAcc.Address()))
	// MasterAccount shouldnt show up in operation level changes
	tt.False(containsAccount(operationChanges, master.Address()))

	tt.True(accountFromEntry(feeRelatedChange.Pre).Balance > accountFromEntry(feeRelatedChange.Post).Balance)
	tt.True(accountFromEntry(txRelatedChange.Pre).SeqNum < accountFromEntry(txRelatedChange.Post).SeqNum)

	srcAccChange := changeForAccount(operationChanges, srcAcc.Address())
	destAccChange := changeForAccount(operationChanges, destAcc.Address())

	tt.True(accountFromEntry(srcAccChange.Pre).Balance < accountFromEntry(srcAccChange.Post).Balance)
	tt.True(accountFromEntry(destAccChange.Pre).Balance > accountFromEntry(destAccChange.Post).Balance)

}

// Helper function to check if a specific XX exists in the list
func containsAccount(slice []ingest.Change, target string) bool {
	for _, change := range slice {
		addr := ledgerKey(change).MustAccount().AccountId.Address()
		if addr == target {
			return true
		}
	}
	return false
}

func ledgerKey(c ingest.Change) xdr.LedgerKey {
	var l xdr.LedgerKey
	if c.Pre != nil {
		l, _ = c.Pre.LedgerKey()
	}
	l, _ = c.Post.LedgerKey()
	return l
}

func getChangesFromLedger(itest *integration.Test, ledger xdr.LedgerCloseMeta) []ingest.Change {
	t := itest.CurrentTest()
	changeReader, err := ingest.NewLedgerChangeReaderFromLedgerCloseMeta(itest.GetPassPhrase(), ledger)
	changes := make([]ingest.Change, 0)
	defer changeReader.Close()
	if err != nil {
		t.Fatalf("unable to create ledger change reader: %v", err)
	}
	for {
		change, err := changeReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unable to read ledger change: %v", err)
		}
		changes = append(changes, change)
	}
	return changes
}

func getLedgers(itest *integration.Test, startingLedger uint32, endLedger uint32) map[uint32]xdr.LedgerCloseMeta {
	t := itest.CurrentTest()

	ccConfig, cleanupFn, err := itest.CreateCaptiveCoreConfig()
	if err != nil {
		t.Fatalf("unable to create captive core config: %v", err)
	}
	defer cleanupFn()

	captiveCore, err := ledgerbackend.NewCaptive(*ccConfig)
	if err != nil {
		t.Fatalf("unable to create captive core: %v", err)
	}
	defer captiveCore.Close()

	ctx := context.Background()
	err = captiveCore.PrepareRange(ctx, ledgerbackend.BoundedRange(startingLedger, endLedger))
	if err != nil {
		t.Fatalf("failed to prepare range: %v", err)
	}

	var seqToLedgersMap = make(map[uint32]xdr.LedgerCloseMeta)
	for ledgerSeq := startingLedger; ledgerSeq <= endLedger; ledgerSeq++ {
		ledger, err := captiveCore.GetLedger(ctx, ledgerSeq)
		if err != nil {
			t.Fatalf("failed to get ledgerNum: %v, error: %v", ledgerSeq, err)
		}
		seqToLedgersMap[ledgerSeq] = ledger
	}

	return seqToLedgersMap
}

func changeReasonCountMap(changes []ingest.Change) map[ingest.LedgerEntryChangeReason]int {
	changeMap := make(map[ingest.LedgerEntryChangeReason]int)
	for _, change := range changes {
		changeMap[change.Reason]++
	}
	return changeMap
}

func changeReasonToChangeMap(changes []ingest.Change) map[ingest.LedgerEntryChangeReason][]ingest.Change {
	changeMap := make(map[ingest.LedgerEntryChangeReason][]ingest.Change)
	for _, change := range changes {
		changeMap[change.Reason] = append(changeMap[change.Reason], change)
	}
	return changeMap
}

func waitForLedgerInArchive(t *testing.T, waitTime time.Duration, ledgerSeq uint32) {
	archive, err := integration.GetHistoryArchive()
	if err != nil {
		t.Fatalf("could not get history archive: %v", err)
	}

	var latestCheckpoint uint32
	var f = func() bool {
		has, requestErr := archive.GetRootHAS()
		if requestErr != nil {
			t.Logf("Request to fetch checkpoint failed: %v", requestErr)
			return false
		}
		latestCheckpoint = has.CurrentLedger
		return latestCheckpoint >= ledgerSeq
	}

	assert.Eventually(t, f, waitTime, 1*time.Second)
}

func getExactUpgradedLedgerSeq(ledgerMap map[uint32]xdr.LedgerCloseMeta, version uint32) uint32 {
	keys := make([]int, 0, len(ledgerMap))
	for key := range ledgerMap {
		keys = append(keys, int(key))
	}
	sort.Ints(keys)

	for _, key := range keys {
		if ledgerMap[uint32(key)].ProtocolVersion() == version {
			return uint32(key)
		}
	}
	return 0
}
