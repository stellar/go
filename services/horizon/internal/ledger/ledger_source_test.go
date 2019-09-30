package ledger

import (
	"testing"
)

func Test_HistoryDBLedgerSourceCurrentLedger(t *testing.T) {
	state := State{
		ExpHistoryLatest: 3,
	}

	ledgerSource := HistoryDBSource{
		updateFrequency: 0,
		currentState: func() State {
			return state
		},
	}

	currentLedger := ledgerSource.CurrentLedger()
	if currentLedger != 3 {
		t.Errorf("CurrentLedger = %d, want 3", currentLedger)
	}
}

func Test_HistoryDBLedgerSourceNextLedger(t *testing.T) {
	state := State{
		ExpHistoryLatest: 3,
	}

	ledgerSource := HistoryDBSource{
		updateFrequency: 0,
		currentState: func() State {
			return state
		},
	}

	ledgerChan := ledgerSource.NextLedger(0)

	nextLedger := <-ledgerChan
	if nextLedger != 3 {
		t.Errorf("NextLedger = %d, want 3", nextLedger)
	}
}
