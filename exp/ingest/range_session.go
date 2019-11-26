package ingest

import (
	"time"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
)

var _ Session = &RangeSession{}

func (s *RangeSession) Run() error {
	s.standardSession.shutdown = make(chan bool)

	err := s.validate()
	if err != nil {
		return errors.Wrap(err, "Validation error")
	}

	s.setRunningState(true)
	defer s.setRunningState(false)

	ledgerAdapter := &adapters.LedgerBackendAdapter{
		Backend: s.LedgerBackend,
	}

	currentLedger := s.FromLedger

	if s.StatePipeline != nil {
		err = initState(
			s.FromLedger,
			s.Archive,
			s.LedgerBackend,
			s.TempSet,
			s.StatePipeline,
			s.StateReporter,
			s.standardSession.shutdown,
			s.MaxStreamRetries,
		)
		if err != nil {
			return errors.Wrap(err, "initState error")
		}

		s.latestSuccessfullyProcessedLedger = currentLedger
	}

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

	return s.resume(currentLedger, ledgerAdapter)
}

// GetArchive returns the archive configured for the current session
func (s *RangeSession) GetArchive() historyarchive.ArchiveInterface {
	return s.Archive
}

// Resume resumes the session from `ledgerSequence`.
//
// WARNING: it's likely that developers will use `GetLatestSuccessfullyProcessedLedger()`
// to get the latest successfuly processed ledger after `Resume` returns error.
// It's critical to understand that `GetLatestSuccessfullyProcessedLedger()` will
// return `(0, false)` when no ledgers have been successfully processed, ex.
// error while trying to process a ledger after application restart.
// You should always check if the second returned value is equal `false` before
// overwriting your local variable.
func (s *RangeSession) Resume(ledgerSequence uint32) error {
	s.standardSession.shutdown = make(chan bool)

	ledgerAdapter := &adapters.LedgerBackendAdapter{
		Backend: s.LedgerBackend,
	}

	return s.resume(ledgerSequence, ledgerAdapter)
}

func (s *RangeSession) resume(ledgerSequence uint32, ledgerAdapter *adapters.LedgerBackendAdapter) error {
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
					return errors.Errorf("Gap detected (ledger %d does not exist but %d is latest)", ledgerSequence, latestLedger)
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

		if s.LedgerReporter != nil {
			s.LedgerReporter.OnNewLedger(ledgerSequence)
			ledgerReader = reporterLedgerReader{ledgerReader, s.LedgerReporter}
		}

		errChan := s.LedgerPipeline.Process(ledgerReader)
		select {
		case err2 := <-errChan:
			if err2 != nil {
				// Return with no errors if pipeline shutdown
				if err2 == pipeline.ErrShutdown {
					s.LedgerReporter.OnEndLedger(nil, true)
					return nil
				}

				if s.LedgerReporter != nil {
					s.LedgerReporter.OnEndLedger(err2, false)
				}
				return errors.Wrap(err2, "Ledger pipeline errored")
			}
		case <-s.standardSession.shutdown:
			if s.LedgerReporter != nil {
				s.LedgerReporter.OnEndLedger(nil, true)
			}
			s.LedgerPipeline.Shutdown()
			return nil
		}

		if s.LedgerReporter != nil {
			s.LedgerReporter.OnEndLedger(nil, false)
		}
		s.latestSuccessfullyProcessedLedger = ledgerSequence

		// We reached the final ledger.
		if ledgerSequence == s.ToLedger {
			return nil
		}

		ledgerSequence++
	}

	return nil
}

// GetLatestSuccessfullyProcessedLedger returns the last SUCCESSFULLY processed
// ledger. Returns (0, false) if no ledgers have been successfully processed yet
// to prevent situations where `GetLatestSuccessfullyProcessedLedger()` value is
// not properly checked in a loop resulting in ingesting ledger 0+1=1.
// Please check `Resume` godoc to understand possible implications.
func (s *RangeSession) GetLatestSuccessfullyProcessedLedger() (ledgerSequence uint32, processed bool) {
	if s.latestSuccessfullyProcessedLedger == 0 {
		return 0, false
	}
	return s.latestSuccessfullyProcessedLedger, true
}

func (s *RangeSession) validate() error {
	switch {
	case s.FromLedger == 0 || s.ToLedger == 0:
		return errors.New("FromLedger and ToLedger must be set")
	case s.FromLedger > s.ToLedger:
		return errors.New("FromLedger must be less than of equal to ToLedger")
	case s.StatePipeline != nil && !historyarchive.IsCheckpoint(s.FromLedger):
		return errors.New("FromLedger must be a checkpoint ledger if StatePipeline is not nil")
	case s.LedgerBackend == nil:
		return errors.New("LedgerBackend not set")
	case s.StatePipeline != nil && s.Archive == nil:
		return errors.New("Archive not set but required by StatePipeline")
	case s.LedgerPipeline == nil:
		return errors.New("Ledger pipeline not set")
	}

	return nil
}
