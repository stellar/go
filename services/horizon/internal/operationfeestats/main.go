// Package operationfeestats provides useful utilities concerning operation fee
// stats within stellar,specifically as a central location to store a cached snapshot
// of the state of network per operation fees and surge pricing. This package is
// intended to be at the lowest levels of horizon's dependency tree, please keep
// it free of dependencies to other horizon packages.
package operationfeestats

import (
	"sync"
)

// State represents a snapshot of horizon's view of the state of operation fee's
// on the network.
type State struct {
	Min         int64
	Mode        int64
	P10         int64
	P20         int64
	P30         int64
	P40         int64
	P50         int64
	P60         int64
	P70         int64
	P80         int64
	P90         int64
	P95         int64
	P99         int64
	LastBaseFee int64
	LastLedger  int64
}

// CurrentState returns the cached snapshot of operation fee state
func CurrentState() State {
	lock.RLock()
	ret := current
	lock.RUnlock()
	return ret
}

// SetState updates the cached snapshot of the operation fee state
func SetState(next State) {
	lock.Lock()
	// in case of one query taking longer than another, this makes
	// sure we don't overwrite the latest fee stats with old stats
	if current.LastLedger < next.LastLedger {
		current = next
	}
	lock.Unlock()
}

// ResetState is used only for testing purposes
func ResetState() {
	current = State{}
}

var current State
var lock sync.RWMutex
