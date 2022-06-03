package main

import "sync"

// SafeStringSet is a simple thread-safe set.
type SafeStringSet struct {
    lock sync.Mutex
    set  map[string]struct{}
}

func NewSafeStringSet() *SafeStringSet {
    return &SafeStringSet{
        lock: sync.Mutex{},
        set:  map[string]struct{}{},
    }
}

func (set *SafeStringSet) Contains(key string) bool {
    defer set.lock.Unlock()
    set.lock.Lock()
    _, ok := set.set[key]
    return ok
}

func (set *SafeStringSet) Add(key string) {
    defer set.lock.Unlock()
    set.lock.Lock()
    set.set[key] = struct{}{}
}
