package ingest

import (
	"time"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/support/errors"
)

var _ Session = &LiveSession{}

func (s *LiveSession) Run() error {
	s.standardSession.shutdown = make(chan bool)

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

	s.standardSession.latestProcessedLedger = currentLedger

	// Exit early if Shutdown() was called.
	select {
	case <-s.standardSession.shutdown:
		return nil
	default:
		// Continue
	}

	// `currentLedger` is incremented because applied state is AFTER the
	// current value of `currentLedger`
	currentLedger++

	return s.resume(currentLedger)
}

func (s *LiveSession) Resume(ledgerSequence uint32) error {
	s.standardSession.shutdown = make(chan bool)
	return s.resume(ledgerSequence)
}

func (s *LiveSession) resume(ledgerSequence uint32) error {
	ledgerAdapter := &adapters.LedgerBackendAdapter{
		Backend: s.LedgerBackend,
	}

	for {
		ledgerReader, err := ledgerAdapter.GetLedger(ledgerSequence)
		if err != nil {
			if err == io.ErrNotFound {
				// Ensure that there are no gaps. This is "just in case". There shouldn't
				// be any gaps if CURSOR in core is updated and core version is v11.2.0+.
				var latestLedger uint32
				latestLedger, err = ledgerAdapter.GetLatestLedgerSequence()
				if err != nil {
					return err
				}

				if latestLedger > ledgerSequence {
					return errors.New("Gap detected")
				}

				select {
				case <-s.standardSession.shutdown:
					s.LedgerPipeline.Shutdown()
					return nil
				case <-time.After(time.Second):
					// TODO make the idle time smaller
				}

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
		case <-s.standardSession.shutdown:
			s.LedgerPipeline.Shutdown()
			return nil
		}

		s.standardSession.latestProcessedLedger = ledgerSequence
		ledgerSequence++
	}

	return nil
}

func (s *LiveSession) GetLatestProcessedLedger() uint32 {
	return s.standardSession.latestProcessedLedger
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
	case <-s.standardSession.shutdown:
		s.StatePipeline.Shutdown()
	}

	return nil
}
