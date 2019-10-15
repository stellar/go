package db

import "sync"

// wrMutex works exactly like `sync.RWMutex` except that lock can be held by an
// arbitrary number of writers (exec) or a single reader (query).
type wrMutex struct {
	m sync.RWMutex
}

func (w *wrMutex) execLock() {
	w.m.RLock()
}

func (w *wrMutex) execUnlock() {
	w.m.RUnlock()
}

func (w *wrMutex) queryLock() {
	w.m.Lock()
}

func (w *wrMutex) queryUnlock() {
	w.m.Unlock()
}
