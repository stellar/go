package main_test

import (
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	batchSize = 321
)

func TestMap(t *testing.T) {
	// Only file:// style URLs for the txmeta source are allowed while testing.
	parsed, err := url.Parse(txmetaSource)
	require.NoErrorf(t, err, "%s is not a valid URL", txmetaSource)
	if parsed.Scheme != "file" {
		t.Logf("%s is not local txmeta source", txmetaSource)
		t.Skip()
	}
	txmetaPath := strings.Replace(txmetaSource, "file://", "", 1)

	// What ledger range are we working with? The maps *require* starting at a
	// checkpoint ledger, so we need to break them up accordingly.
	startLedger, endLedger := GetFixtureLedgerRange(t)
	if !IsCheckpoint(startLedger) {
		startLedger = NextCheckpoint(startLedger)
	}
	batchCount := (endLedger - startLedger + batchSize) / batchSize // ceil(ledgerCount / batchSize)

	require.Truef(t, batchCount == 1 || IsCheckpoint(startLedger+batchSize+1),
		"expected batch size (%d) to result in checkpoint blocks, "+
			"but start+batchSize+1 (%d+%d+1=%d) is not a checkpoint",
		batchSize, startLedger, batchSize+startLedger+1)

	// First, execute the map jobs in parallel and dump the resulting indices to
	// a temporary directory.

	tempDir := filepath.Join(t.TempDir(), "indices-map")
	mapTestCmd := exec.Command("./map.sh", txmetaPath, tempDir)
	mapTestCmd.Env = append(os.Environ(),
		"BATCH_SIZE="+strconv.FormatUint(batchSize, 10),
		"FIRST_LEDGER="+strconv.FormatUint(uint64(startLedger), 10),
		"LAST_LEDGER="+strconv.FormatUint(uint64(endLedger), 10))
	t.Logf("Running %d map jobs: %s", batchCount, mapTestCmd.String())
	stdout, err := mapTestCmd.CombinedOutput()

	t.Logf("Tried writing indices to %s:", tempDir)
	t.Log(string(stdout))
	require.NoError(t, err)

	// Then, build the *same* indices using the single-process tester.
	t.Logf("Building baseline for ledger range [%d, %d]", startLedger, endLedger)
	hashes, participants := CreateBaselineIndices(t, txmetaSource, startLedger, endLedger)
	require.NotNil(t, hashes)
	require.NotNil(t, participants)

	// Now, walk through the mapped indices and ensure that at least one of the
	// jobs reported the same indices for tx TOIDs and participation.

	stores := make([]index.Store, batchCount)
	for i := range stores {
		indexUrl := filepath.Join(
			"file://",
			tempDir,
			"job_"+strconv.FormatUint(uint64(i), 10),
		)
		index, err := index.Connect(indexUrl)
		require.NoError(t, err)
		require.NotNil(t, index)
		stores[i] = index

		t.Logf("Connected to index #%d at %s", i+1, indexUrl)
	}

	assertParticipantsEqual(t, keys(participants), stores)
	for account, checkpoints := range participants {
		assertParticipantCheckpointsEqual(t, account, checkpoints, stores)
	}
}

func assertParticipantsEqual(t *testing.T,
	expectedAccountSet []string,
	indexGroup []index.Store,
) {
	indexGroupAccountSet := make(map[string]struct{}, len(expectedAccountSet))
	for _, store := range indexGroup {
		accounts, err := store.ReadAccounts()
		require.NoError(t, err)

		for _, account := range accounts {
			indexGroupAccountSet[account] = struct{}{}
		}
	}

	assert.Lenf(t, indexGroupAccountSet, len(expectedAccountSet),
		"quantity of accounts across indices doesn't match")

	mappedAccountSet := keys(indexGroupAccountSet)
	require.ElementsMatch(t, expectedAccountSet, mappedAccountSet)
}

func assertParticipantCheckpointsEqual(t *testing.T,
	account string,
	expected []uint32,
	indexGroup []index.Store,
) {
	// Ensure that all of the active checkpoints reported by the index match
	// the ones we tracked while ingesting the range ourselves.

	foundCheckpoints := make(map[uint32]struct{}, len(expected))
	for _, store := range indexGroup {
		var err error
		var lastActiveCheckpoint uint32 = 0
		for {
			lastActiveCheckpoint, err = store.NextActive(account, "all/all", lastActiveCheckpoint)
			if err == io.EOF {
				break
			}
			require.NoError(t, err) // still an error since it shouldn't happen

			foundCheckpoints[lastActiveCheckpoint] = struct{}{}
			lastActiveCheckpoint += 1 // hit next active one
		}
	}

	// Error out if there were any extraneous checkpoints found.
	for chk := range foundCheckpoints {
		require.Containsf(t, expected, chk,
			"found unexpected checkpoint %d", int(chk))
	}

	// Make sure everything got marked as expected in at least one index.
	for _, item := range expected {
		require.Containsf(t, foundCheckpoints, item,
			"failed to find %d for %s (found %v)",
			int(item), account, foundCheckpoints)
	}
}

func keys[T any](dict map[string]T) []string {
	result := make([]string, 0, len(dict))
	for key := range dict {
		result = append(result, key)
	}
	return result
}
