package io

// MemoryStateReaderTempStore is an in-memory implementation of
// StateReaderTempStore. As of July 2019 this requires up to ~4GB of
// memory for pubnet ledger state processing. The internal structure is
// dereferenced after the store is closed.
type MemoryStateReaderTempStore struct {
	m map[string]bool
}

func (s *MemoryStateReaderTempStore) Open() error {
	s.m = make(map[string]bool)
	return nil
}

func (s *MemoryStateReaderTempStore) Add(key string) error {
	s.m[key] = true
	return nil
}

func (s *MemoryStateReaderTempStore) Preload(keys []string) error {
	// Everything in memory - no preloading needed
	return nil
}

func (s *MemoryStateReaderTempStore) Exist(key string) (bool, error) {
	return s.m[key], nil
}

func (s *MemoryStateReaderTempStore) Close() error {
	s.m = nil
	return nil
}
