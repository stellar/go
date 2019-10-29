package hubble

import (
	"sync"

	"github.com/stellar/go/exp/ingest"
	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
)

var _ ingest.Session = &TestingSession{}

// TestingSession is a basic implementation of the `ingest.Session`
// interface, used
type TestingSession struct {
	standardSession

	Archive *historyarchive.Archive
	StatePipeline *pipeline.StatePipeline
	TempSet io.TempSet
}

// Run runs a Hubble session.
func (s *TestingSession) Run() error {
	s.setRunningState(true)
	defer s.setRunningState(false)
	s.shutdown = make(chan bool)

	historyAdapter := adapters.MakeHistoryArchiveAdapter(s.Archive)

	var err error
	sequence, err := historyAdapter.GetLatestLedgerSequence()
	if err != nil {
		return errors.Wrap(err, "Error getting the latest ledger sequence")
	}

	entryType := "State"
	err = s.processEntry(historyAdapter, sequence, entryType)
	if err != nil {
		return errors.Wrap(err, "processState errored")
	}

	s.standardSession.latestSuccessfullyProcessedLedger = sequence
	return nil
	
}

// Resume resumes a Hubble session.
func (s *TestingSession) Resume(ledgerSequence uint32) error {
	panic("Can't resume TestingSession!")
}

func (s *TestingSession) processEntry(historyAdapter *adapters.HistoryArchiveAdapter, sequence uint32, entryType string) error {
	var tempSet io.TempSet = &io.MemoryTempSet{}
	if s.TempSet != nil {
		tempSet = s.TempSet
	}

	switch entryType {
	case "State":
		return s.processState(historyAdapter, sequence, tempSet)
	case "Created":
		return s.processCreated(historyAdapter, sequence, tempSet)
	case "Removed":
		return s.processRemoved(historyAdapter, sequence, tempSet)
	case "Updated":
		return s.processUpdated(historyAdapter, sequence, tempSet)
	default:
		return errors.New("entryType not supported in processEntry")
	}
}

func (s *TestingSession) processState(historyAdapter *adapters.HistoryArchiveAdapter, sequence uint32, tempSet io.TempSet) error {
	stateReader, err := historyAdapter.GetState(sequence, tempSet)
	if err != nil {
		return errors.Wrap(err, "Error getting state from history archive")
	}
	errChan := s.StatePipeline.Process(stateReader)
	select {
	case err := <-errChan:
		if err != nil {
			return errors.Wrap(err, "State pipeline errored")
		}
	case <-s.shutdown:
		s.StatePipeline.Shutdown()
	}
	return nil
}

func (s *TestingSession) processCreated(historyAdapter *adapters.HistoryArchiveAdapter, sequence uint32, tempSet io.TempSet) error {
	return nil
}

func (s *TestingSession) processRemoved(historyAdapter *adapters.HistoryArchiveAdapter, sequence uint32, tempSet io.TempSet) error {
	return nil
}

func (s *TestingSession) processUpdated(historyAdapter *adapters.HistoryArchiveAdapter, sequence uint32, tempSet io.TempSet) error {
	return nil
}


// TODO: Export the below from `ingest`, so people can roll their own sessions.
type standardSession struct {
	shutdown                          chan bool
	rwLock                            sync.RWMutex
	latestSuccessfullyProcessedLedger uint32

	runningMutex sync.Mutex
	running      bool
}

func (s *standardSession) setRunningState(newState bool) {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if s.running && newState {
		panic("Session is running...")
	}
	s.running = newState
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

func (s *standardSession) GetLatestSuccessfullyProcessedLedger() uint32 {
	return s.latestSuccessfullyProcessedLedger
}
