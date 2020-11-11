package ledger

import (
	"sync"
	"testing"
)

func Test_HistoryDBLedgerSourceCurrentLedger(t *testing.T) {
	cache := &Cache{
		RWMutex: sync.RWMutex{},
		current: State{ExpHistoryLatest: 3},
	}

	ledgerSource := HistoryDBSource{
		updateFrequency: 0,
		cache:           cache,
	}

	currentLedger := ledgerSource.CurrentLedger()
	if currentLedger != 3 {
		t.Errorf("CurrentLedger = %d, want 3", currentLedger)
	}
}

func Test_HistoryDBLedgerSourceNextLedger(t *testing.T) {
	cache := &Cache{
		RWMutex: sync.RWMutex{},
		current: State{ExpHistoryLatest: 3},
	}

	ledgerSource := HistoryDBSource{
		updateFrequency: 0,
		cache:           cache,
	}

	ledgerChan := ledgerSource.NextLedger(0)

	nextLedger := <-ledgerChan
	if nextLedger != 3 {
		t.Errorf("NextLedger = %d, want 3", nextLedger)
	}
}
