package ingest

import (
	"bytes"
	"fmt"
	"time"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
)

var _ Session = &RangeSession{}

// Run runs the session starting from the last checkpoint ledger.
// Returns nil when session has been shutdown.
func (s *RangeSession) Run() error {
	s.standardSession.shutdown = make(chan bool)

	err := s.validate()
	if err != nil {
		return errors.Wrap(err, "Validation error")
	}

	s.setRunningState(true)
	defer s.setRunningState(false)

	historyAdapter := adapters.MakeHistoryArchiveAdapter(s.Archive)
	currentLedger := s.FromLedger

	ledgerAdapter := &adapters.LedgerBackendAdapter{
		Backend: s.LedgerBackend,
	}

	if s.StatePipeline != nil {
		err = s.initState(historyAdapter, ledgerAdapter, currentLedger)
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
// Returns nil when session has been shutdown.
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

	err := s.validate()
	if err != nil {
		return errors.Wrap(err, "Validation error")
	}

	s.setRunningState(true)
	defer s.setRunningState(false)

	ledgerAdapter := &adapters.LedgerBackendAdapter{
		Backend: s.LedgerBackend,
	}

	return s.resume(ledgerSequence, ledgerAdapter)
}

// validateBucketList validates if the bucket list hash in history archive
// matches the one in corresponding ledger header in stellar-core backend.
// This gives you full security if data in stellar-core backend can be trusted
// (ex. you run it in your infrastructure).
// The hashes of actual buckets of this HAS file are checked using
// historyarchive.XdrStream.SetExpectedHash (this is done in
// SingleLedgerStateReader).
func (s *RangeSession) validateBucketList(
	ledgerSequence uint32,
	historyAdapter *adapters.HistoryArchiveAdapter,
	ledgerAdapter *adapters.LedgerBackendAdapter,
) error {
	historyBucketListHash, err := historyAdapter.BucketListHash(ledgerSequence)
	if err != nil {
		return errors.Wrap(err, "Error getting bucket list hash")
	}

	ledgerReader, err := ledgerAdapter.GetLedger(ledgerSequence)
	if err != nil {
		if err == io.ErrNotFound {
			return fmt.Errorf(
				"Cannot validate bucket hash list. Checkpoint ledger (%d) must exist in Stellar-Core database.",
				ledgerSequence,
			)
		} else {
			return errors.Wrap(err, "Error getting ledger")
		}
	}

	ledgerHeader := ledgerReader.GetHeader()
	ledgerBucketHashList := ledgerHeader.Header.BucketListHash

	if !bytes.Equal(historyBucketListHash[:], ledgerBucketHashList[:]) {
		return fmt.Errorf(
			"Bucket list hash of history archive and ledger header does not match: %#x %#x",
			historyBucketListHash,
			ledgerBucketHashList,
		)
	}

	return nil
}

func (s *RangeSession) resume(ledgerSequence uint32, ledgerAdapter *adapters.LedgerBackendAdapter) error {
	if ledgerSequence < s.FromLedger {
		return errors.New("Trying to resume from ledger before range start")
	}

	if ledgerSequence > s.ToLedger {
		return nil
	}

	for {
		ledgerReader, err := ledgerAdapter.GetLedger(ledgerSequence)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Error getting ledger %d", ledgerSequence))
		}

		if s.LedgerReporter != nil {
			s.LedgerReporter.OnNewLedger(ledgerSequence)
			ledgerReader = reporterLedgerReader{ledgerReader, s.LedgerReporter}
		}

		err = <-s.LedgerPipeline.Process(ledgerReader)
		if err != nil {
			// Return with no errors if pipeline shutdown
			if err == pipeline.ErrShutdown {
				if s.LedgerReporter != nil {
					s.LedgerReporter.OnEndLedger(nil, true)
				}
				return nil
			}

			if s.LedgerReporter != nil {
				s.LedgerReporter.OnEndLedger(err, false)
			}
			return errors.Wrap(err, "Ledger pipeline errored")
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

		// Exit early if Shutdown() was called.
		select {
		case <-s.standardSession.shutdown:
			return nil
		default:
			// Continue
		}
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
	case s.StatePipeline != nil && !historyarchive.IsCheckpoint(s.FromLedger) && s.FromLedger != 1:
		return errors.New("FromLedger must be a checkpoint ledger if StatePipeline is not nil")
	case s.LedgerBackend == nil:
		return errors.New("LedgerBackend not set")
	case s.StatePipeline != nil && s.Archive == nil:
		return errors.New("Archive not set but required by StatePipeline")
	case s.LedgerPipeline == nil:
		return errors.New("Ledger pipeline not set")
	case s.NetworkPassphrase == "":
		return errors.New("Network passphrase not set")
	}

	return nil
}

func (s *RangeSession) initState(
	historyAdapter *adapters.HistoryArchiveAdapter,
	ledgerAdapter *adapters.LedgerBackendAdapter,
	sequence uint32,
) error {
	var stateReader io.StateReader
	var err error

	if sequence == 1 {
		stateReader = &io.GenesisLedgerStateReader{
			NetworkPassphrase: s.NetworkPassphrase,
		}
	} else {
		// Validate bucket list hash
		err = s.validateBucketList(sequence, historyAdapter, ledgerAdapter)
		if err != nil {
			return errors.Wrap(err, "Error validating bucket list hash")
		}

		var tempSet io.TempSet = &io.MemoryTempSet{}
		if s.TempSet != nil {
			tempSet = s.TempSet
		}

		stateReader, err = historyAdapter.GetState(sequence, tempSet, s.MaxStreamRetries)
		if err != nil {
			return errors.Wrap(err, "Error getting state from history archive")
		}
	}
	if s.StateReporter != nil {
		s.StateReporter.OnStartState(sequence)
		stateReader = reporterStateReader{stateReader, s.StateReporter}
	}

	err = <-s.StatePipeline.Process(stateReader)
	if err != nil {
		// Return with no errors if pipeline shutdown
		if err == pipeline.ErrShutdown {
			if s.StateReporter != nil {
				s.StateReporter.OnEndState(nil, true)
			}
			return nil
		}

		if s.StateReporter != nil {
			s.StateReporter.OnEndState(err, false)
		}
		return errors.Wrap(err, "State pipeline errored")
	}

	if s.StateReporter != nil {
		s.StateReporter.OnEndState(nil, false)
	}
	return nil
}

// Shutdown gracefully stops the pipelines and the session. This method blocks
// until pipelines are gracefully shutdown.
func (s *RangeSession) Shutdown() {
	// Send shutdown signal
	s.standardSession.Shutdown()

	// Shutdown pipelines
	if s.StatePipeline != nil {
		s.StatePipeline.Shutdown()
	}
	s.LedgerPipeline.Shutdown()

	// Shutdown signals sent, block/wait until pipelines are done
	// shutting down.
	for {
		var stateRunning bool
		if s.StatePipeline != nil {
			stateRunning = s.StatePipeline.IsRunning()
		} else {
			stateRunning = false
		}
		ledgerRunning := s.LedgerPipeline.IsRunning()
		if stateRunning || ledgerRunning {
			time.Sleep(time.Second)
			continue
		}
		break
	}
}
