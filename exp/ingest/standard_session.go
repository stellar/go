package ingest

func (s *standardSession) ensureRunOnce() {
	s.doneMutex.Lock()
	if s.done {
		panic("Session already running or done...")
	}
	s.done = true
	s.doneMutex.Unlock()
}

func (s *standardSession) Shutdown() {
	close(s.shutdown)
}

func (s *standardSession) QueryLock() {
	s.rwLock.RLock()
}

func (s *standardSession) QueryUnlock() {
	s.rwLock.RUnlock()
}

func (s *standardSession) UpdateLock() {
	s.rwLock.Lock()
}

func (s *standardSession) UpdateUnlock() {
	s.rwLock.Unlock()
}

func (s *standardSession) GetLatestProcessedLedger() uint32 {
	return s.latestProcessedLedger
}
