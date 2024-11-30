package integration

import (
	"context"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCoreDump(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{SkipHorizonStart: true})
	archive, err := historyarchive.Connect(
		integration.HistoryArchiveUrl,
		historyarchive.ArchiveOptions{
			NetworkPassphrase:   integration.StandaloneNetworkPassphrase,
			CheckpointFrequency: integration.CheckpointFrequency,
		})
	tt.NoError(err)

	var latestCheckpoint uint32
	startTime := time.Now()
	publishedNextCheckpoint := func() bool {
		has, requestErr := archive.GetRootHAS()
		if requestErr != nil {
			t.Logf("request to fetch checkpoint failed: %v", requestErr)
			return false
		}
		latestCheckpoint = has.CurrentLedger
		t.Logf("Latest ledger so far: %d", latestCheckpoint)
		return latestCheckpoint >= uint32(7) // ALLOW for atleast 3 checkpoints
	}
	//time.Sleep(15 * time.Second)

	// Ensure that a checkpoint has been created with the ledgerNumber you want in it
	tt.Eventually(publishedNextCheckpoint, 45*time.Second, time.Second)
	endTime := time.Now()

	t.Logf("waited %v seconds to start captive core...", endTime.Sub(startTime).Seconds())
	t.Log("---------- STARTING CAPTIVE CORE ---------")

	ledgerSeqToLedgers := getLedgersFromArchive(itest, 2, 7)
	t.Logf("----- length of hashmap is %v", len(ledgerSeqToLedgers))
	time.Sleep(45 * time.Second)
}

func getLedgersFromArchive(itest *integration.Test, startingLedger uint32, endLedger uint32) map[uint32]xdr.LedgerCloseMeta {
	t := itest.CurrentTest()

	ccConfig, cleanpupFn, err := itest.CreateCaptiveCoreConfig()
	if err != nil {
		panic(err)
	}

	defer cleanpupFn()
	captiveCore, err := ledgerbackend.NewCaptive(*ccConfig)
	if err != nil {
		panic(err)
	}
	defer captiveCore.Close()

	ctx := context.Background()
	require.NoError(t, err)

	err = captiveCore.PrepareRange(ctx, ledgerbackend.BoundedRange(startingLedger, endLedger))
	if err != nil {
		t.Fatalf("failed to prepare range: %v", err)
	}

	t.Logf("Ledger Range ----- [%v, %v]", startingLedger, endLedger)

	var seqToLedgersMap = make(map[uint32]xdr.LedgerCloseMeta)
	for ledgerSeq := startingLedger; ledgerSeq <= endLedger; ledgerSeq++ {
		ledger, err := captiveCore.GetLedger(ctx, ledgerSeq)
		if err != nil {
			t.Fatalf("failed to get ledgerNum: %v, error: %v", ledgerSeq, err)
		}
		seqToLedgersMap[ledgerSeq] = ledger
		itest.CurrentTest().Logf("processed ledgerNum: %v, hash: %v", ledgerSeq, ledger.LedgerHash().HexString())
	}

	return seqToLedgersMap
}
