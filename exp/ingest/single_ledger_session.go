package ingest

import (
	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/pipeline"
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

	return nil
}

func (s *SingleLedgerSession) SetStatePipeline(p *pipeline.StatePipeline) {
	s.statePipeline = p
}

func (s *SingleLedgerSession) SetLedgerPipeline(p *pipeline.LedgerPipeline) {
	panic("SingleLedgerSession does not accept LedgerPipeline")
}

func (s *SingleLedgerSession) processState(historyAdapter *adapters.HistoryArchiveAdapter, sequence uint32) error {
	stateReader, err := historyAdapter.GetState(sequence)
	if err != nil {
		return errors.Wrap(err, "Error getting state from history archive")
	}

	errChan := s.statePipeline.Process(stateReader)
	select {
	case err := <-errChan:
		if err != nil {
			return errors.Wrap(err, "State pipeline errored")
		}
	case <-s.shutdown:
		s.statePipeline.Shutdown()
	}

	return nil
}
