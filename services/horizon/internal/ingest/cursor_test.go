package ingest

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
)

func TestCursor(t *testing.T) {
	tt := test.Start(t).ScenarioWithoutHorizon("kahuna")
	defer tt.Finish()

	//
	c := Cursor{
		FirstLedger: 7,
		LastLedger:  10,
		CoreDB:      tt.CoreSession(),
	}

	// Ledger 7
	tt.Require.True(c.NextLedger())
	tt.Require.True(c.NextTx())
	tt.Require.True(c.NextOp())
	tt.Require.False(c.NextOp())
	tt.Require.False(c.NextTx())

	// Ledger 8
	tt.Require.True(c.NextLedger())
	tt.Require.True(c.NextTx())
	tt.Require.True(c.NextOp())
	tt.Require.False(c.NextOp())
	tt.Require.True(c.NextTx())
	tt.Require.True(c.NextOp())
	tt.Require.False(c.NextOp())
	tt.Require.True(c.NextTx())
	tt.Require.True(c.NextOp())
	tt.Require.False(c.NextOp())
	tt.Require.True(c.NextTx())
	tt.Require.True(c.NextOp())
	tt.Require.False(c.NextOp())
	tt.Require.False(c.NextTx())

	// Ledger 9
	tt.Require.True(c.NextLedger())
	tt.Require.True(c.NextTx())
	tt.Require.True(c.NextOp())
	tt.Require.False(c.NextOp())
	tt.Require.False(c.NextTx())

	// Ledger 10
	tt.Require.True(c.NextLedger())
	tt.Require.True(c.NextTx())
	tt.Require.True(c.NextOp())
	tt.Require.True(c.NextOp())
	tt.Require.False(c.NextOp())
	tt.Require.False(c.NextTx())

	tt.Require.False(c.NextLedger())

	// Reverse
	c = Cursor{
		FirstLedger: 10,
		LastLedger:  7,
		CoreDB:      tt.CoreSession(),
	}

	tt.Require.True(c.NextLedger())
	tt.Require.Equal(uint32(10), c.Ledger().Sequence)
	tt.Require.True(c.NextLedger())
	tt.Require.Equal(uint32(9), c.Ledger().Sequence)
	tt.Require.True(c.NextLedger())
	tt.Require.Equal(uint32(8), c.Ledger().Sequence)
	tt.Require.True(c.NextLedger())
	tt.Require.Equal(uint32(7), c.Ledger().Sequence)

	tt.Require.False(c.NextLedger())

}
