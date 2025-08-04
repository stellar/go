package integration

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/ingest/loadtest"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

func TestLoadTestLedgerBackend(t *testing.T) {
	if integration.GetCoreMaxSupportedProtocol() < 22 {
		t.Skip("This test run does not support less than Protocol 22")
	}

	itest := integration.NewTest(t, integration.Config{
		NetworkPassphrase: loadTestNetworkPassphrase,
	})
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

	replayConfig := loadtest.LedgerBackendConfig{
		NetworkPassphrase:     itest.Config().NetworkPassphrase,
		LedgersFilePath:       filepath.Join("testdata", fmt.Sprintf("load-test-ledgers-v%d.xdr.zstd", itest.Config().ProtocolVersion)),
		LedgerEntriesFilePath: filepath.Join("testdata", fmt.Sprintf("load-test-accounts-v%d.xdr.zstd", itest.Config().ProtocolVersion)),
		LedgerCloseDuration:   3 * time.Second / 2,
		LedgerBackend:         captiveCore,
	}
	loadTestBackend := loadtest.NewLedgerBackend(replayConfig)

	var generatedLedgers []xdr.LedgerCloseMeta
	var generatedLedgerEntries []xdr.LedgerEntry

	readFile(t, replayConfig.LedgersFilePath,
		func() *xdr.LedgerCloseMeta { return &xdr.LedgerCloseMeta{} },
		func(ledger *xdr.LedgerCloseMeta) {
			generatedLedgers = append(generatedLedgers, *ledger)
		},
	)
	readFile(t, replayConfig.LedgerEntriesFilePath,
		func() *xdr.LedgerEntry { return &xdr.LedgerEntry{} },
		func(ledgerEntry *xdr.LedgerEntry) {
			generatedLedgerEntries = append(generatedLedgerEntries, *ledgerEntry)
		},
	)

	startLedger := uint32(tx.Ledger - 1)
	endLedger := startLedger + uint32(len(generatedLedgers))

	_, err = loadTestBackend.GetLatestLedgerSequence(context.Background())
	require.EqualError(t, err, "PrepareRange() must be called before GetLatestLedgerSequence()")

	prepared, err := loadTestBackend.IsPrepared(context.Background(), ledgerbackend.BoundedRange(startLedger, endLedger))
	require.NoError(t, err)
	require.False(t, prepared)

	itest.WaitForLedgerInArchive(6*time.Minute, endLedger)
	require.NoError(t, loadTestBackend.PrepareRange(context.Background(), ledgerbackend.BoundedRange(startLedger, endLedger)))

	latest, err := loadTestBackend.GetLatestLedgerSequence(context.Background())
	require.NoError(t, err)
	require.Equal(t, endLedger, latest)

	prepared, err = loadTestBackend.IsPrepared(context.Background(), ledgerbackend.BoundedRange(startLedger, endLedger))
	require.NoError(t, err)
	require.True(t, prepared)

	prepared, err = loadTestBackend.IsPrepared(context.Background(), ledgerbackend.BoundedRange(startLedger+1, endLedger))
	require.NoError(t, err)
	require.True(t, prepared)

	prepared, err = loadTestBackend.IsPrepared(context.Background(), ledgerbackend.BoundedRange(startLedger-1, endLedger))
	require.NoError(t, err)
	require.False(t, prepared)

	prepared, err = loadTestBackend.IsPrepared(context.Background(), ledgerbackend.BoundedRange(endLedger+1, endLedger+2))
	require.NoError(t, err)
	require.False(t, prepared)

	prepared, err = loadTestBackend.IsPrepared(context.Background(), ledgerbackend.UnboundedRange(startLedger))
	require.NoError(t, err)
	require.False(t, prepared)

	require.NoError(
		t,
		loadTestBackend.PrepareRange(context.Background(), ledgerbackend.BoundedRange(startLedger, endLedger)),
	)
	require.NoError(
		t,
		loadTestBackend.PrepareRange(context.Background(), ledgerbackend.BoundedRange(startLedger+1, endLedger)),
	)
	require.EqualError(
		t,
		loadTestBackend.PrepareRange(context.Background(), ledgerbackend.UnboundedRange(startLedger)),
		"PrepareRange() already called",
	)

	_, err = loadTestBackend.GetLedger(context.Background(), startLedger-1)
	require.EqualError(t, err,
		fmt.Sprintf(
			"sequence number %v is behind the ledger stream sequence %v",
			startLedger-1,
			startLedger,
		),
	)

	var ledgers []xdr.LedgerCloseMeta
	for cur := startLedger; cur <= endLedger; cur++ {
		startTime := time.Now()
		var ledger xdr.LedgerCloseMeta
		ledger, err = loadTestBackend.GetLedger(context.Background(), cur)
		duration := time.Since(startTime)
		require.NoError(t, err)
		ledgers = append(ledgers, ledger)
		require.WithinDuration(t, startTime.Add(replayConfig.LedgerCloseDuration), startTime.Add(duration), time.Millisecond*100)
	}

	prepared, err = loadTestBackend.IsPrepared(context.Background(), ledgerbackend.BoundedRange(startLedger, endLedger))
	require.NoError(t, err)
	require.False(t, prepared)

	_, err = loadTestBackend.GetLedger(context.Background(), endLedger+1)
	require.EqualError(t, err,
		fmt.Sprintf("sequence number %v is greater than the latest ledger available", endLedger+1),
	)

	require.NoError(t, loadTestBackend.Close())

	originalLedgers := getLedgers(itest, startLedger, endLedger)

	changes := extractChanges(t, itest.Config().NetworkPassphrase, ledgers[0:1])
	expectedChanges := extractChanges(
		t, itest.Config().NetworkPassphrase, []xdr.LedgerCloseMeta{originalLedgers[startLedger]},
	)
	for i := range generatedLedgerEntries {
		expectedChanges = append(expectedChanges, ingest.Change{
			Type:   generatedLedgerEntries[i].Data.Type,
			Post:   &generatedLedgerEntries[i],
			Reason: ingest.LedgerEntryChangeReasonUpgrade,
		})
	}
	checkChanges(t, expectedChanges, changes, startLedger)

	for cur := startLedger + 1; cur <= endLedger; cur++ {
		i := int(cur - startLedger)
		changes = extractChanges(t, itest.Config().NetworkPassphrase, ledgers[i:i+1])
		expectedChanges = extractChanges(
			t, itest.Config().NetworkPassphrase, []xdr.LedgerCloseMeta{originalLedgers[cur], generatedLedgers[i-1]},
		)
		checkChanges(t, expectedChanges, changes, cur)
	}
}

func checkChanges(t *testing.T, expected []ingest.Change, actual []ingest.Change, ledger uint32) {
	aByLedgerKey := groupChangesByLedgerKey(t, expected)
	bByLedgerKey := groupChangesByLedgerKey(t, actual)

	require.Equal(t, len(aByLedgerKey), len(bByLedgerKey))
	for key, aChanges := range aByLedgerKey {
		bChanges := bByLedgerKey[key]
		require.Equal(t, len(aChanges), len(bChanges))
		for i, aChange := range aChanges {
			bChange := bChanges[i]
			require.Equal(t, aChange.Reason, bChange.Reason)
			require.Equal(t, aChange.Type, bChange.Type)
			if aChange.Pre == nil {
				require.Nil(t, bChange.Pre)
			} else {
				checkChange(t, aChange.Pre, bChange.Pre, ledger, true)
			}
			if aChange.Post == nil {
				require.Nil(t, bChange.Post)
			} else {
				checkChange(t, aChange.Post, bChange.Post, ledger, false)
			}
		}
	}
}

func checkChange(t *testing.T, expected, actual *xdr.LedgerEntry, curLedger uint32, pre bool) {
	if pre {
		require.LessOrEqual(t, actual.LastModifiedLedgerSeq, curLedger)
	} else {
		require.Equal(t, uint32(actual.LastModifiedLedgerSeq), curLedger)
		if actual.Data.Type == xdr.LedgerEntryTypeTtl {
			require.GreaterOrEqual(t, actual.Data.Ttl.LiveUntilLedgerSeq, curLedger)
		}
	}
	require.NoError(t, loadtest.UpdateLedgerSeq(expected, func(u uint32) uint32 {
		return 0
	}))
	require.NoError(t, loadtest.UpdateLedgerSeq(actual, func(u uint32) uint32 {
		return 0
	}))
	requireXDREquals(t, expected, actual)
}
