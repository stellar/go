package ingest

import (
	"testing"

	"github.com/stellar/go/network"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/test"
)

func TestIngest_Kahuna1(t *testing.T) {
	tt := test.Start(t).ScenarioWithoutHorizon("kahuna")
	defer tt.Finish()

	s := ingest(tt, false)

	tt.Require.NoError(s.Err)
	tt.Assert.Equal(62, s.Ingested)

	// Test that re-importing fails
	s.Err = nil
	s.Run()
	tt.Require.Error(s.Err, "Reimport didn't fail as expected")

	// Test that re-importing fails with allowing clear succeeds
	s.Err = nil
	s.ClearExisting = true
	s.Run()
	tt.Require.NoError(s.Err, "Couldn't re-import, even with clear allowed")
}

func TestIngest_Kahuna2(t *testing.T) {
	tt := test.Start(t).ScenarioWithoutHorizon("kahuna-2")
	defer tt.Finish()

	s := ingest(tt, false)

	tt.Require.NoError(s.Err)
	tt.Assert.Equal(6, s.Ingested)

	// ensure that the onetime signer is gone
	q := core.Q{Session: tt.CoreSession()}
	var signers []core.Signer

	err := q.SignersByAddress(&signers, "GD6NTRJW5Z6NCWH4USWMNEYF77RUR2MTO6NP4KEDVJATTCUXDRO3YIFS")
	tt.Require.NoError(err)

	tt.Assert.Len(signers, 1)
}

func TestTick(t *testing.T) {
	tt := test.Start(t).ScenarioWithoutHorizon("base")
	defer tt.Finish()
	sys := sys(tt, false)

	// ingest by tick
	s := sys.Tick()
	tt.Require.NoError(s.Err)
	tt.Require.Nil(sys.current)

	tt.UpdateLedgerState()
	s = sys.Tick()
	tt.Require.NotNil(s)
	tt.Require.NoError(s.Err)
}

func ingest(tt *test.T, enableAssetStats bool) *Session {
	sys := sys(tt, enableAssetStats)
	s := NewSession(sys)
	s.Cursor = NewCursor(1, ledger.CurrentState().CoreLatest, sys)
	s.Run()

	return s
}

func sys(tt *test.T, enableAssetStats bool) *System {
	return New(
		network.TestNetworkPassphrase,
		"",
		tt.CoreSession(),
		tt.HorizonSession(),
		Config{EnableAssetStats: enableAssetStats},
	)
}
