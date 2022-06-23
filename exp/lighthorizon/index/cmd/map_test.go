package main_test

import (
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stretchr/testify/require"
)

func TestMap(t *testing.T) {
	parsed, err := url.Parse(txmetaSource)
	require.NoError(t, err)
	require.Equalf(t, parsed.Scheme, "file",
		"%s is not local txmeta source", txmetaSource)

	txmetaPath := strings.Replace(txmetaSource, "file://", "", 1)

	// First, execute the map jobs in parallel and dump the resulting indices to
	// a temporary directory.

	tempDir := filepath.Join(os.TempDir(), "indices-map")
	mapTestCmd := exec.Command("./map.sh", txmetaPath, tempDir)
	t.Log("Running map jobs:", mapTestCmd.String())
	stdout, err := mapTestCmd.CombinedOutput()

	t.Logf("Tried writing indices to %s:", tempDir)
	t.Log(string(stdout))

	require.NoError(t, err)

	// Then, build the *same* indices using the single-process tester.
	//
	// FIXME: We could probably use the *actual* single-process builder here,
	// but is that too much coupling? It means if the single-process fails, the
	// map version will fail as well.

	var lastLedger uint32 = elderLedger + (64*5 - 1) // exactly 5 checkpoints of data
	hashes, participants := CreateBaselineIndices(t, txmetaSource, elderLedger, lastLedger)
	require.NotNil(t, hashes)
	require.NotNil(t, participants)

	// Now, walk through the mapped indices and ensure that at least one of the
	// jobs reported the same indices for tx TOIDs and participation.

	checkpointCount := (lastLedger - elderLedger) / 64
	stores := make([]index.Store, checkpointCount)

	for i := range stores {
		index, err := index.Connect(filepath.Join("file://", tempDir))
		require.NoError(t, err)
		require.NotNil(t, index)
		stores[i] = index
	}

	for account, checkpoints := range participants {
		assertParticipantExists(t, account, checkpoints, stores)
	}
}

func assertParticipantExists(t *testing.T,
	account string,
	expected []uint32,
	indexGroup []index.Store,
) {
	looking := make(map[uint32]struct{}, len(expected))

	for _, store := range indexGroup {
		var err error

		// Ensure that all of the active checkpoints reported by the index match
		// the ones we tracked while ingesting the range ourselves.
		activeCheckpoints := []uint32{}
		lastActiveCheckpoint := uint32(0)
		for {
			lastActiveCheckpoint, err = store.NextActive(account, "all/all", lastActiveCheckpoint)
			if err != nil {
				break
			}

			activeCheckpoints = append(activeCheckpoints, lastActiveCheckpoint)
			lastActiveCheckpoint += 1 // hit next active one
		}

		// Mark any checkpoints that we're looking as found if we find them in
		// this index, and also error out if there are extraneous ones.
		for _, chk := range activeCheckpoints {
			require.Containsf(t, expected, chk,
				"found unexpected checkpoint %d", int(chk))
			looking[chk] = struct{}{}
		}
	}

	// Make sure everything got marked as expected in at least one index.
	for _, item := range expected {
		require.Containsf(t, looking, item,
			"failed to find %d (found %v)", int(item), looking)
	}
}
