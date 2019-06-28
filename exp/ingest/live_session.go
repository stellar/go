package ingest

import (
	"time"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/support/errors"
)

var _ Session = &LiveSession{}

func (s *LiveSession) Run() error {
	s.shutdown = make(chan bool)

	err := s.validate()
	if err != nil {
		return errors.Wrap(err, "Validation error")
	}

	s.ensureRunOnce()

	historyAdapter := adapters.MakeHistoryArchiveAdapter(s.Archive)

	currentLedger, err := historyAdapter.GetLatestLedgerSequence()
	if err != nil {
		return errors.Wrap(err, "Error getting the latest ledger sequence")
	}

	err = s.initState(historyAdapter, currentLedger)
	if err != nil {
		return errors.Wrap(err, "initState error")
	}

	// Exit early if Shutdown() was called.
	select {
	case <-s.shutdown:
		return nil
	default:
		// Continue
	}

	// `currentLedger` is incremented because applied state is AFTER the
	// current value of `currentLedger`
	currentLedger++

	ledgerAdapter := &adapters.LedgerBackendAdapter{
		Backend: s.LedgerBackend,
	}

	for {
		ledgerReader, err := ledgerAdapter.GetLedger(currentLedger)
		if err != nil {
			if err == io.ErrNotFound {
				time.Sleep(time.Second)
				continue
			}

			return errors.Wrap(err, "Error getting ledger")
		}

		errChan := s.LedgerPipeline.Process(ledgerReader)
		select {
		case err := <-errChan:
			if err != nil {
				return errors.Wrap(err, "Ledger pipeline errored")
			}
		case <-s.shutdown:
			s.LedgerPipeline.Shutdown()
			return nil
		}

		currentLedger++
	}

	return nil
}

func (s *LiveSession) validate() error {
	switch {
	case s.Archive == nil:
		return errors.New("Archive not set")
	case s.LedgerBackend == nil:
		return errors.New("LedgerBackend not set")
	case s.StatePipeline == nil:
		return errors.New("State pipeline not set")
	case s.LedgerPipeline == nil:
		return errors.New("Ledger pipeline not set")
	}

	return nil
}

func (s *LiveSession) initState(historyAdapter *adapters.HistoryArchiveAdapter, sequence uint32) error {
	stateReader, err := historyAdapter.GetState(sequence)
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
