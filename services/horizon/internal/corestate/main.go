package corestate

import (
	"sync"

	"github.com/stellar/go/protocols/stellarcore"
)

type State struct {
	Synced                       bool
	CurrentProtocolVersion       int32
	CoreSupportedProtocolVersion int32
	CoreVersion                  string
}

type Store struct {
	sync.RWMutex
	State
}

func (c *Store) Set(resp *stellarcore.InfoResponse) {
	c.Lock()
	defer c.Unlock()
	c.Synced = resp.IsSynced()
	c.CoreVersion = resp.Info.Build
	c.CurrentProtocolVersion = int32(resp.Info.Ledger.Version)
	c.CoreSupportedProtocolVersion = int32(resp.Info.ProtocolVersion)
}

func (c *Store) Get() State {
	c.RLock()
	defer c.RUnlock()
	return c.State
}
