package integration

import (
	"context"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
	"time"
)

func TestProtocolUpgradeChanges(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{SkipHorizonStart: true, SkipProtocolUpgrade: true})
	archive, err := historyarchive.Connect(
		integration.HistoryArchiveUrl,
		historyarchive.ArchiveOptions{
			NetworkPassphrase:   integration.StandaloneNetworkPassphrase,
			CheckpointFrequency: integration.CheckpointFrequency,
		})
	tt.NoError(err)

	// Manually invoke command to upgrade protocol
	itest.UpgradeProtocol(itest.Config().ProtocolVersion)
	upgradedLedgerSeq, _ := itest.GetUpgradeLedgerSeq()

	var latestCheckpoint uint32
	publishedNextCheckpoint := func() bool {
		has, requestErr := archive.GetRootHAS()
		if requestErr != nil {
			t.Logf("request to fetch checkpoint failed: %v", requestErr)
			return false
		}
		latestCheckpoint = has.CurrentLedger
		return latestCheckpoint >= upgradedLedgerSeq
	}
	// Ensure that a checkpoint has been created with the ledgerNumber you want in it
	tt.Eventually(publishedNextCheckpoint, 15*time.Second, time.Second)

	prevLedgerToUpgrade := upgradedLedgerSeq - 1
	ledgerSeqToLedgers := getLedgersFromArchive(itest, prevLedgerToUpgrade, upgradedLedgerSeq)
	prevLedgerChangeMap := changeMap(getChangesFromLedger(itest, ledgerSeqToLedgers[prevLedgerToUpgrade]))
	upgradedLedgerChangeMap := changeMap(getChangesFromLedger(itest, ledgerSeqToLedgers[upgradedLedgerSeq]))

	tt.Zero(prevLedgerChangeMap[ingest.LedgerEntryChangeReasonUpgrade])
	tt.NotZero(upgradedLedgerChangeMap[ingest.LedgerEntryChangeReasonUpgrade])
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

func changeMap(changes []ingest.Change) map[ingest.LedgerEntryChangeReason]int {
	changeMap := make(map[ingest.LedgerEntryChangeReason]int)
	for _, change := range changes {
		changeMap[change.Reason]++
	}
	return changeMap
}
