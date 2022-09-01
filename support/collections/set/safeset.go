package set

import (
	"sync"

	"golang.org/x/exp/constraints"
)

// SafeSet is a simple, thread-safe set implementation.
type SafeSet[T constraints.Ordered] struct {
	Set[T]
	lock sync.RWMutex
}

func NewSafeSet[T constraints.Ordered](capacity int) *SafeSet[T] {
	return &SafeSet[T]{
		Set:  NewSet[T](capacity),
		lock: sync.RWMutex{},
	}
}

func (s *SafeSet[T]) Add(item T) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.Set.Add(item)
}

func (s *SafeSet[T]) AddSlice(items []T) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.Set.AddSlice(items)
}

func (s *SafeSet[T]) Remove(item T) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.Set.Remove(item)
}

func (s *SafeSet[T]) Contains(item T) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.Set.Contains(item)
}

func (s *SafeSet[T]) Slice() []T {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.Set.Slice()
}
