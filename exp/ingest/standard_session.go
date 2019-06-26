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
