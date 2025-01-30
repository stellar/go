package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

func TestReplay(t *testing.T) {
	itest := integration.NewTest(t, integration.Config{})
	senderKP, senderAccount := itest.CreateAccount("10000000")
	recipientKP, _ := itest.CreateAccount("10000000")

	tx := itest.MustSubmitOperations(
		senderAccount,
		senderKP,
		&txnbuild.Payment{
			SourceAccount: senderKP.Address(),
			Destination:   recipientKP.Address(),
			Asset:         txnbuild.NativeAsset{},
			Amount:        "1000",
		},
	)
	require.True(t, tx.Successful)

	ccConfig, err := itest.CreateCaptiveCoreConfig()
	require.NoError(t, err)

	captiveCore, err := ledgerbackend.NewCaptive(ccConfig)
	require.NoError(t, err)
	defer captiveCore.Close()

	replayConfig := ledgerbackend.ReplayBackendConfig{
		LedgersFilePath:       filepath.Join("testdata", "load-test-ledgers.xdr"),
		LedgerEntriesFilePath: filepath.Join("testdata", "load-test-accounts.xdr"),
		LedgerCloseDuration:   3 * time.Second / 2,
	}
	replayBackend, err := ledgerbackend.NewReplayBackend(replayConfig, captiveCore)
	require.NoError(t, err)

	var generatedLedgers []xdr.LedgerCloseMeta
	var generatedLedgerEntries []xdr.LedgerEntry

	ledgersFile, err := os.Open(replayConfig.LedgersFilePath)
	require.NoError(t, err)
	ledgerEntriesFile, err := os.Open(replayConfig.LedgerEntriesFilePath)
	require.NoError(t, err)
	_, err = xdr.Unmarshal(ledgersFile, &generatedLedgers)
	require.NoError(t, err)
	_, err = xdr.Unmarshal(ledgerEntriesFile, &generatedLedgerEntries)
	require.NoError(t, err)
	require.NoError(t, ledgersFile.Close())
	require.NoError(t, ledgerEntriesFile.Close())

	startLedger := uint32(tx.Ledger - 1)
	endLedger := startLedger + uint32(len(generatedLedgers))
	waitForLedgerInArchive(t, 6*time.Minute, endLedger)
	require.NoError(t, replayBackend.PrepareRange(context.Background(), ledgerbackend.BoundedRange(startLedger, endLedger)))

	_, err = replayBackend.GetLedger(context.Background(), startLedger-1)
	require.EqualError(t, err, fmt.Sprintf("sequence number %v out of range", startLedger-1))

	_, err = replayBackend.GetLedger(context.Background(), endLedger+1)
	require.EqualError(t, err, fmt.Sprintf("sequence number %v out of range", endLedger+1))

	var ledgers []xdr.LedgerCloseMeta
	for cur := startLedger; cur <= endLedger; cur++ {
		startTime := time.Now()
		ledger, err := replayBackend.GetLedger(context.Background(), cur)
		duration := time.Since(startTime)
		require.NoError(t, err)
		ledgers = append(ledgers, ledger)
		require.WithinDuration(t, startTime.Add(replayConfig.LedgerCloseDuration), startTime.Add(duration), time.Millisecond*100)
	}

	originalLedgers := getLedgers(itest, startLedger, endLedger)

	changes := extractChanges(t, ledgers[0:1])
	expectedChanges := extractChanges(t, []xdr.LedgerCloseMeta{originalLedgers[startLedger]})
	for i := range generatedLedgerEntries {
		expectedChanges = append(expectedChanges, ingest.Change{
			Type:   generatedLedgerEntries[i].Data.Type,
			Post:   &generatedLedgerEntries[i],
			Reason: ingest.LedgerEntryChangeReasonUpgrade,
		})
	}
	requireChangesAreEqual(t, expectedChanges, changes)

	for cur := startLedger + 1; cur <= endLedger; cur++ {
		i := int(cur - startLedger)
		changes = extractChanges(t, ledgers[i:i+1])
		expectedChanges = extractChanges(t, []xdr.LedgerCloseMeta{originalLedgers[cur], generatedLedgers[i-1]})
		requireChangesAreEqual(t, expectedChanges, changes)
	}
}
