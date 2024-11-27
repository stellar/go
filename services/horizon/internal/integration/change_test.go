package integration

import (
	"context"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestChangeDataForTxWithOneOperation(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{}) // set config to 21
	master := itest.Master()

	keys, accounts := itest.CreateAccounts(1, "1000")
	accAkeys, _ := keys[0], accounts[0]
	//accBkeys, accB := keys[1], accounts[1]

	paymentToA := txnbuild.Payment{
		Destination: accAkeys.Address(),
		Amount:      "100",
		Asset:       txnbuild.NativeAsset{},
	}

	// Submit a transaction
	txResp := itest.MustSubmitOperations(itest.MasterAccount(), master, &paymentToA)
	tt.True(txResp.Successful)
	//txHash := txResp.Hash
	ledgerSeq := uint32(txResp.Ledger)

	// Stop horizon
	itest.StopHorizon()

	archive, err := historyarchive.Connect(
		itest.GetHorizonIngestConfig().HistoryArchiveURLs[0],
		historyarchive.ArchiveOptions{
			NetworkPassphrase:   itest.GetHorizonIngestConfig().NetworkPassphrase,
			CheckpointFrequency: itest.GetHorizonIngestConfig().CheckpointFrequency,
		})
	tt.NoError(err)

	var latestCheckpoint uint32
	publishedNextCheckpoint := func() bool {
		has, requestErr := archive.GetRootHAS()
		if requestErr != nil {
			t.Logf("request to fetch checkpoint failed: %v", requestErr)
			return false
		}
		latestCheckpoint = has.CurrentLedger
		return latestCheckpoint > ledgerSeq
	}

	// Ensure that a checkpoint has been created with the ledgerNumber you want in it
	tt.Eventually(publishedNextCheckpoint, 10*time.Second, time.Second)

	t.Log("---------- STARTING CAPTIVE CORE ---------")

	ledgerSeqToLedgers := getLedgersFromArchive(itest, ledgerSeq)
	t.Logf("----- length of hashmap is %v", len(ledgerSeqToLedgers))
}

func getLedgersFromArchive(itest *integration.Test, maxLedger uint32) map[uint32]xdr.LedgerCloseMeta {
	t := itest.CurrentTest()
	captiveCore, err := itest.GetDefaultCaptiveCoreInstance()
	defer captiveCore.Close()

	ctx := context.Background()
	require.NoError(t, err)

	startingLedger := uint32(2)

	err = captiveCore.PrepareRange(ctx, ledgerbackend.UnboundedRange(startingLedger))
	if err != nil {
		t.Fatalf("failed to prepare range: %v", err)
	}

	t.Logf("Ledger Range ----- [%v, %v]", startingLedger, maxLedger)

	var seqToLedgersMap = make(map[uint32]xdr.LedgerCloseMeta)
	for ledgerSeq := startingLedger; ledgerSeq <= maxLedger; ledgerSeq++ {
		ledger, err := captiveCore.GetLedger(ctx, ledgerSeq)
		if err != nil {
			t.Fatalf("failed to get ledgerNum: %v, error: %v", ledgerSeq, err)
		}
		seqToLedgersMap[ledgerSeq] = ledger
		itest.CurrentTest().Logf("processed ledger ---- %v", ledgerSeq)
	}

	return seqToLedgersMap
}
