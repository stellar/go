package main_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/network"
	"github.com/stellar/go/toid"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/exp/lighthorizon/index"
)

const (
	txmetaSource = "file://./fixtures/"
	elderLedger  = 1410048
)

/**
 * There are three parts to testing this correctly:
 *  - test that single-process indexing works
 *  - test that single-process w/ multi-worker works
 *  - test map-reduce against the single-process results
 *
 * Therefore, if any of these fail, the subsequent ones are unreliable.
 */

func TestSingleProcess(tt *testing.T) {
	const (
		ledgerCount  = 64*5 - 1 // exactly 5 checkpoints of ledger data
		latestLedger = elderLedger + ledgerCount
	)

	tt.Logf("Validating single-process builder on %d ledgers", ledgerCount)

	for workerCount := 1; workerCount <= 16; workerCount *= 2 {
		tt.Run(
			fmt.Sprintf("workers/%d", workerCount),
			func(t *testing.T) {
				tmpDir := filepath.Join("file://", t.TempDir())
				t.Logf("Storing indices in %s", tmpDir)

				ctx := context.Background()
				_, err := index.BuildIndices(
					ctx,
					txmetaSource,
					tmpDir,
					network.TestNetworkPassphrase,
					elderLedger,
					latestLedger,
					[]string{
						"accounts",
						"transactions",
					},
					workerCount,
				)
				require.NoError(t, err)

				hashes, participants := CreateBaselineIndices(t,
					txmetaSource, elderLedger, latestLedger)

				store, err := index.Connect(tmpDir)
				require.NoError(t, err)
				require.NotNil(t, store)

				// Ensure the participants reported by the index and the ones we
				// tracked while ingesting the ledger range match.
				AssertParticipantsEqual(t, participants, store)

				// Ensure the transactions reported by the index match the ones
				// tracked when ingesting the ledger range ourselves.
				AssertTxsEqual(t, hashes, store)
			},
		)
	}
}

func AssertTxsEqual(t *testing.T, expected map[string]int64, actual index.Store) {
	for hash, knownTOID := range expected {
		rawHash, err := hex.DecodeString(hash)
		require.NoError(t, err, "bug")
		require.Len(t, rawHash, 32)

		tempBuf := [32]byte{}
		copy(tempBuf[:], rawHash[:])

		rawTOID, err := actual.TransactionTOID(tempBuf)
		require.NoErrorf(t, err, "expected TOID for tx hash %s", hash)

		require.Equalf(t, knownTOID, rawTOID,
			"expected TOID %v, got %v",
			toid.Parse(knownTOID), toid.Parse(rawTOID))
	}
}

func AssertParticipantsEqual(t *testing.T, expected map[string][]uint32, actual index.Store) {
	accounts, err := actual.ReadAccounts()

	require.NoError(t, err)
	require.Len(t, accounts, len(expected))
	for account := range expected {
		require.Contains(t, accounts, account)
	}

	for account, knownCheckpoints := range expected {
		// Ensure that the "everything" index exists for the account.
		index, err := actual.Read(account)
		require.NoError(t, err)
		require.Contains(t, index, "all/all")

		// Ensure that all of the active checkpoints reported by the index match the ones we
		// tracked while ingesting the range ourselves.
		activeCheckpoints := []uint32{}
		lastActiveCheckpoint := uint32(0)
		for {
			lastActiveCheckpoint, err = actual.NextActive(account, "all/all", lastActiveCheckpoint)
			if err == io.EOF {
				break
			}
			require.NoError(t, err)

			activeCheckpoints = append(activeCheckpoints, lastActiveCheckpoint)
			lastActiveCheckpoint += 1 // hit next active one
		}

		require.Equalf(t, knownCheckpoints, activeCheckpoints,
			"incorrect checkpoints for %s", account)
	}
}

// CreateBaselineIndices will connect to a dump of ledger txmeta for the given
// ledger range and build two maps from scratch (i.e. without using the indexer)
// by ingesting them manually:
//
//   - a map of tx hashes to TOIDs
//   - a map of accounts to a list of checkpoints they were active in
//
// These should be used as a baseline comparison of the indexer, ensuring that
// all of the data is identical.
func CreateBaselineIndices(
	t *testing.T,
	txmetaSource string,
	startLedger, endLedger uint32,
) (
	map[string]int64, // map of "tx hash": TOID
	map[string][]uint32, // map of "account": {checkpoint, checkpoint, ...}
) {
	ctx := context.Background()
	backend, err := historyarchive.ConnectBackend(
		txmetaSource,
		historyarchive.ConnectOptions{
			Context:           ctx,
			NetworkPassphrase: network.TestNetworkPassphrase,
			S3Region:          "us-east-1",
		},
	)
	require.NoError(t, err)
	ledgerBackend := ledgerbackend.NewHistoryArchiveBackend(backend)
	defer ledgerBackend.Close()

	participation := make(map[string][]uint32)
	hashes := make(map[string]int64)

	for ledgerSeq := startLedger; ledgerSeq <= endLedger; ledgerSeq++ {
		ledger, err := ledgerBackend.GetLedger(ctx, uint32(ledgerSeq))
		require.NoError(t, err)
		require.EqualValues(t, ledgerSeq, ledger.LedgerSequence())

		reader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(
			network.TestNetworkPassphrase, ledger)
		require.NoError(t, err)

		for {
			tx, err := reader.Read()
			if err == io.EOF {
				break
			}
			require.NoError(t, err)

			participants, err := index.GetParticipants(tx)
			require.NoError(t, err)

			for _, participant := range participants {
				checkpoint := 1 + (ledger.LedgerSequence() / 64)

				// Track the checkpoint in which activity occurred,
				// keeping the list duplicate-free.
				if list, ok := participation[participant]; ok {
					if list[len(list)-1] != checkpoint {
						participation[participant] = append(list, checkpoint)
					}
				} else {
					participation[participant] = []uint32{checkpoint}
				}
			}

			// Track the ledger sequence in which every tx occurred.
			hash := hex.EncodeToString(tx.Result.TransactionHash[:])
			hashes[hash] = toid.New(
				int32(ledger.LedgerSequence()),
				int32(tx.Index),
				0,
			).ToInt64()
		}
	}

	return hashes, participation
}
