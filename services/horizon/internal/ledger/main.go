// Package ledger provides useful utilities concerning ledgers within stellar,
// specifically as a central location to store a cached snapshot of the state of
// both horizon's and stellar-core's views of the ledger.  This package is
// intended to be at the lowest levels of horizon's dependency tree, please keep
// it free of dependencies to other horizon packages.
package ledger

import (
	"sync"
)

// Status represents a snapshot of both horizon's and stellar-core's view of the
// ledger.
type Status struct {
	HistoryLatest    int32
	HistoryElder     int32
	ExpHistoryLatest uint32
}

// State is an in-memory data structure which holds a snapshot of both
// horizon's and stellar-core's view of the the network
type State struct {
	sync.RWMutex
	current Status
}

// CurrentStatus returns the cached snapshot of ledger state
func (c *State) CurrentStatus() Status {
	c.RLock()
	defer c.RUnlock()
	ret := c.current
	return ret
}

// SetStatus updates the cached snapshot of the ledger state
func (c *State) SetStatus(next Status) {
	c.Lock()
	defer c.Unlock()
	c.current = next
}
