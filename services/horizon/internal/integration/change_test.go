package integration

import (
	"context"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
	"time"
)

func TestProtocolUpgradeChanges(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{SkipHorizonStart: true, SkipProtocolUpgrade: true})
	archive, err := integration.GetHistoryArchive()
	tt.NoError(err)

	// Manually invoke command to upgrade protocol
	itest.UpgradeProtocol(itest.Config().ProtocolVersion)
	upgradedLedgerSeq, _ := itest.GetUpgradeLedgerSeq()

	publishedNextCheckpoint := publishedNextCheckpoint(archive, upgradedLedgerSeq, t)
	// Ensure that a checkpoint has been created with the ledgerNumber you want in it
	tt.Eventually(publishedNextCheckpoint, 15*time.Second, time.Second)

	prevLedgerToUpgrade := upgradedLedgerSeq - 1
	ledgerSeqToLedgers := getLedgersFromArchive(itest, prevLedgerToUpgrade, upgradedLedgerSeq)

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
	keyA, keyB := keys[0], keys[1]

	operation := txnbuild.Payment{
		SourceAccount: keyA.Address(),
		Destination:   keyB.Address(),
		Asset:         txnbuild.NativeAsset{},
		Amount:        "900",
	}
	txResp, err := itest.SubmitMultiSigOperations(itest.MasterAccount(), []*keypair.Full{master, keyA}, &operation)
	tt.NoError(err)
	ledgerSeq := uint32(txResp.Ledger)

	archive, err := integration.GetHistoryArchive()
	tt.NoError(err)

	publishedNextCheckpoint := publishedNextCheckpoint(archive, ledgerSeq, t)
	// Ensure that a checkpoint has been created with the ledgerNumber you want in it
	tt.Eventually(publishedNextCheckpoint, 15*time.Second, time.Second)

	ledger := getLedgersFromArchive(itest, ledgerSeq, ledgerSeq)[ledgerSeq]
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

	tt.Equal(
		ledgerKey(reasonToChangeMap[ingest.LedgerEntryChangeReasonFee][0]).MustAccount().AccountId.Address(),
		master.Address())
	tt.Equal(
		ledgerKey(reasonToChangeMap[ingest.LedgerEntryChangeReasonTransaction][0]).MustAccount().AccountId.Address(),
		master.Address())
	tt.True(containsAccount(reasonToChangeMap[ingest.LedgerEntryChangeReasonOperation], keyA.Address()))
	tt.True(containsAccount(reasonToChangeMap[ingest.LedgerEntryChangeReasonOperation], keyB.Address()))
	// MasterAccount shouldnt show up in operation level changes
	tt.False(containsAccount(reasonToChangeMap[ingest.LedgerEntryChangeReasonOperation], master.Address()))
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

func getLedgersFromArchive(itest *integration.Test, startingLedger uint32, endLedger uint32) map[uint32]xdr.LedgerCloseMeta {
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

func publishedNextCheckpoint(archive *historyarchive.Archive, ledgerSeq uint32, t *testing.T) func() bool {
	return func() bool {
		var latestCheckpoint uint32
		has, requestErr := archive.GetRootHAS()
		if requestErr != nil {
			t.Logf("Request to fetch checkpoint failed: %v", requestErr)
			return false
		}
		latestCheckpoint = has.CurrentLedger
		return latestCheckpoint >= ledgerSeq
	}
}

func changeReasonToChangeMap(changes []ingest.Change) map[ingest.LedgerEntryChangeReason][]ingest.Change {
	changeMap := make(map[ingest.LedgerEntryChangeReason][]ingest.Change)
	for _, change := range changes {
		changeMap[change.Reason] = append(changeMap[change.Reason], change)
	}
	return changeMap
}
