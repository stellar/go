package ingest

import (
	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/support/errors"
)

var _ Session = &SingleLedgerSession{}

func (s *SingleLedgerSession) Run() error {
	s.ensureRunOnce()
	s.shutdown = make(chan bool)

	historyAdapter := adapters.MakeHistoryArchiveAdapter(s.Archive)

	var err error
	sequence := s.LedgerSequence
	if sequence == 0 {
		sequence, err = historyAdapter.GetLatestLedgerSequence()
		if err != nil {
			return errors.Wrap(err, "Error getting the latest ledger sequence")
		}
	}

	err = s.processState(historyAdapter, sequence)
	if err != nil {
		return errors.Wrap(err, "processState errored")
	}

	s.standardSession.latestProcessedLedger = sequence
	return nil
}

func (s *SingleLedgerSession) Resume(ledgerSequence uint32) error {
	panic("Not possible to resume SingleLedgerSession")
}

func (s *SingleLedgerSession) processState(historyAdapter *adapters.HistoryArchiveAdapter, sequence uint32) error {
	var tempStore io.StateReaderTempStore = &io.MemoryStateReaderTempStore{}
	if s.StateTempStore != nil {
		tempStore = s.StateTempStore
	}

	stateReader, err := historyAdapter.GetState(sequence, tempStore)
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
