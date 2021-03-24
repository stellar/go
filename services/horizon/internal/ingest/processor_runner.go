package ingest

import (
	"bytes"
	"context"
	"fmt"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/processors"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type ingestionSource int

const (
	_                    = iota
	historyArchiveSource = ingestionSource(iota)
	ledgerSource         = ingestionSource(iota)
	logFrequency         = 100000
)

type horizonChangeProcessor interface {
	processors.ChangeProcessor
	Commit() error
}

type horizonTransactionProcessor interface {
	processors.LedgerTransactionProcessor
	Commit() error
}

type statsChangeProcessor struct {
	*ingest.StatsChangeProcessor
}

func (statsChangeProcessor) Commit() error {
	return nil
}

type statsLedgerTransactionProcessor struct {
	*processors.StatsLedgerTransactionProcessor
}

func (statsLedgerTransactionProcessor) Commit() error {
	return nil
}

type ProcessorRunnerInterface interface {
	SetLedgerBackend(ledgerBackend ledgerbackend.LedgerBackend)
	SetHistoryAdapter(historyAdapter historyArchiveAdapterInterface)
	EnableMemoryStatsLogging()
	DisableMemoryStatsLogging()
	RunHistoryArchiveIngestion(checkpointLedger uint32) (ingest.StatsChangeProcessorResults, error)
	RunTransactionProcessorsOnLedger(sequence uint32) (
		transactionStats processors.StatsLedgerTransactionProcessorResults,
		transactionDurations processorsRunDurations,
		err error,
	)
	RunAllProcessorsOnLedger(sequence uint32) (
		changeStats ingest.StatsChangeProcessorResults,
		changeDurations processorsRunDurations,
		transactionStats processors.StatsLedgerTransactionProcessorResults,
		transactionDurations processorsRunDurations,
		err error,
	)
}

var _ ProcessorRunnerInterface = (*ProcessorRunner)(nil)

type ProcessorRunner struct {
	config Config

	ctx            context.Context
	historyQ       history.IngestionQ
	historyAdapter historyArchiveAdapterInterface
	ledgerBackend  ledgerbackend.LedgerBackend
	logMemoryStats bool
}

func (s *ProcessorRunner) SetLedgerBackend(ledgerBackend ledgerbackend.LedgerBackend) {
	s.ledgerBackend = ledgerBackend
}

func (s *ProcessorRunner) SetHistoryAdapter(historyAdapter historyArchiveAdapterInterface) {
	s.historyAdapter = historyAdapter
}

func (s *ProcessorRunner) EnableMemoryStatsLogging() {
	s.logMemoryStats = true
}

func (s *ProcessorRunner) DisableMemoryStatsLogging() {
	s.logMemoryStats = false
}

func (s *ProcessorRunner) buildChangeProcessor(
	changeStats *ingest.StatsChangeProcessor,
	source ingestionSource,
	sequence uint32,
) *groupChangeProcessors {
	statsChangeProcessor := &statsChangeProcessor{
		StatsChangeProcessor: changeStats,
	}

	useLedgerCache := source == ledgerSource
	return newGroupChangeProcessors([]horizonChangeProcessor{
		statsChangeProcessor,
		processors.NewAccountDataProcessor(s.historyQ),
		processors.NewAccountsProcessor(s.historyQ),
		processors.NewOffersProcessor(s.historyQ, sequence),
		processors.NewAssetStatsProcessor(s.historyQ, useLedgerCache),
		processors.NewSignersProcessor(s.historyQ, useLedgerCache),
		processors.NewTrustLinesProcessor(s.historyQ),
		processors.NewClaimableBalancesChangeProcessor(s.historyQ),
	})
}

func (s *ProcessorRunner) buildTransactionProcessor(
	ledgerTransactionStats *processors.StatsLedgerTransactionProcessor,
	ledger xdr.LedgerHeaderHistoryEntry,
) *groupTransactionProcessors {
	statsLedgerTransactionProcessor := &statsLedgerTransactionProcessor{
		StatsLedgerTransactionProcessor: ledgerTransactionStats,
	}

	sequence := uint32(ledger.Header.LedgerSeq)
	return newGroupTransactionProcessors([]horizonTransactionProcessor{
		statsLedgerTransactionProcessor,
		processors.NewEffectProcessor(s.historyQ, sequence),
		processors.NewLedgerProcessor(s.historyQ, ledger, CurrentVersion),
		processors.NewOperationProcessor(s.historyQ, sequence),
		processors.NewTradeProcessor(s.historyQ, ledger),
		processors.NewParticipantsProcessor(s.historyQ, sequence),
		processors.NewTransactionProcessor(s.historyQ, sequence),
		processors.NewClaimableBalancesTransactionProcessor(s.historyQ, sequence),
	})
}

// checkIfProtocolVersionSupported checks if this Horizon version supports the
// protocol version of a ledger with the given sequence number.
func (s *ProcessorRunner) checkIfProtocolVersionSupported(ledgerSequence uint32) error {
	exists, ledgerCloseMeta, err := s.ledgerBackend.GetLedger(ledgerSequence)
	if err != nil {
		return errors.Wrap(err, "Error getting ledger")
	}

	if !exists {
		return errors.New("cannot check if protocol version supported: ledger does not exist")
	}

	ledgerVersion := ledgerCloseMeta.V0.LedgerHeader.Header.LedgerVersion

	if ledgerVersion > MaxSupportedProtocolVersion {
		return fmt.Errorf(
			"This Horizon version does not support protocol version %d. "+
				"The latest supported protocol version is %d. Please upgrade to the latest Horizon version.",
			ledgerVersion,
			MaxSupportedProtocolVersion,
		)
	}

	return nil
}

// validateBucketList validates if the bucket list hash in history archive
// matches the one in corresponding ledger header in stellar-core backend.
// This gives you full security if data in stellar-core backend can be trusted
// (ex. you run it in your infrastructure).
// The hashes of actual buckets of this HAS file are checked using
// historyarchive.XdrStream.SetExpectedHash (this is done in
// CheckpointChangeReader).
func (s *ProcessorRunner) validateBucketList(ledgerSequence uint32) error {
	historyBucketListHash, err := s.historyAdapter.BucketListHash(ledgerSequence)
	if err != nil {
		return errors.Wrap(err, "Error getting bucket list hash")
	}

	exists, ledgerCloseMeta, err := s.ledgerBackend.GetLedger(ledgerSequence)
	if err != nil {
		return errors.Wrap(err, "Error getting ledger")
	}

	if !exists {
		return fmt.Errorf(
			"cannot validate bucket hash list. Checkpoint ledger (%d) must exist in Stellar-Core database.",
			ledgerSequence,
		)
	}

	ledgerBucketHashList := ledgerCloseMeta.V0.LedgerHeader.Header.BucketListHash

	if !bytes.Equal(historyBucketListHash[:], ledgerBucketHashList[:]) {
		return fmt.Errorf(
			"Bucket list hash of history archive and ledger header does not match: %#x %#x",
			historyBucketListHash,
			ledgerBucketHashList,
		)
	}

	return nil
}

func (s *ProcessorRunner) RunHistoryArchiveIngestion(checkpointLedger uint32) (ingest.StatsChangeProcessorResults, error) {
	changeStats := ingest.StatsChangeProcessor{}
	changeProcessor := s.buildChangeProcessor(&changeStats, historyArchiveSource, checkpointLedger)

	if checkpointLedger == 1 {
		if err := changeProcessor.ProcessChange(ingest.GenesisChange(s.config.NetworkPassphrase)); err != nil {
			return changeStats.GetResults(), errors.Wrap(err, "Error ingesting genesis ledger")
		}
	} else {
		if err := s.checkIfProtocolVersionSupported(checkpointLedger); err != nil {
			return changeStats.GetResults(), errors.Wrap(err, "Error while checking for supported protocol version")
		}

		if err := s.validateBucketList(checkpointLedger); err != nil {
			return changeStats.GetResults(), errors.Wrap(err, "Error validating bucket list from HAS")
		}

		changeReader, err := s.historyAdapter.GetState(s.ctx, checkpointLedger)
		if err != nil {
			return changeStats.GetResults(), errors.Wrap(err, "Error creating HAS reader")
		}

		defer changeReader.Close()

		log.WithField("ledger", checkpointLedger).
			Info("Processing entries from History Archive Snapshot")

		err = processors.StreamChanges(changeProcessor, newloggingChangeReader(
			changeReader,
			"historyArchive",
			checkpointLedger,
			logFrequency,
			s.logMemoryStats,
		))
		if err != nil {
			return changeStats.GetResults(), errors.Wrap(err, "Error streaming changes from HAS")
		}
	}

	if err := changeProcessor.Commit(); err != nil {
		return changeStats.GetResults(), errors.Wrap(err, "Error commiting changes from processor")
	}

	return changeStats.GetResults(), nil
}

func (s *ProcessorRunner) runChangeProcessorOnLedger(
	changeProcessor horizonChangeProcessor, ledger uint32,
) error {
	var changeReader ingest.ChangeReader
	var err error
	changeReader, err = ingest.NewLedgerChangeReader(s.ledgerBackend, s.config.NetworkPassphrase, ledger)
	if err != nil {
		return errors.Wrap(err, "Error creating ledger change reader")
	}
	changeReader = newloggingChangeReader(
		changeReader,
		"ledger",
		ledger,
		logFrequency,
		s.logMemoryStats,
	)
	if err = processors.StreamChanges(changeProcessor, changeReader); err != nil {
		return errors.Wrap(err, "Error streaming changes from ledger")
	}

	err = changeProcessor.Commit()
	if err != nil {
		return errors.Wrap(err, "Error commiting changes from processor")
	}

	return nil
}

func (s *ProcessorRunner) RunTransactionProcessorsOnLedger(ledger uint32) (
	transactionStats processors.StatsLedgerTransactionProcessorResults,
	transactionDurations processorsRunDurations,
	err error,
) {
	var (
		ledgerTransactionStats processors.StatsLedgerTransactionProcessor
		transactionReader      *ingest.LedgerTransactionReader
	)

	transactionReader, err = ingest.NewLedgerTransactionReader(s.ledgerBackend, s.config.NetworkPassphrase, ledger)
	if err != nil {
		err = errors.Wrap(err, "Error creating ledger reader")
		return
	}

	if err = s.checkIfProtocolVersionSupported(ledger); err != nil {
		err = errors.Wrap(err, "Error while checking for supported protocol version")
		return
	}

	groupTransactionProcessors := s.buildTransactionProcessor(&ledgerTransactionStats, transactionReader.GetHeader())
	err = processors.StreamLedgerTransactions(groupTransactionProcessors, transactionReader)
	if err != nil {
		err = errors.Wrap(err, "Error streaming changes from ledger")
		return
	}

	err = groupTransactionProcessors.Commit()
	if err != nil {
		err = errors.Wrap(err, "Error commiting changes from processor")
		return
	}

	transactionStats = ledgerTransactionStats.GetResults()
	transactionDurations = groupTransactionProcessors.processorsRunDurations
	return
}

func (s *ProcessorRunner) RunAllProcessorsOnLedger(sequence uint32) (
	changeStats ingest.StatsChangeProcessorResults,
	changeDurations processorsRunDurations,
	transactionStats processors.StatsLedgerTransactionProcessorResults,
	transactionDurations processorsRunDurations,
	err error,
) {
	changeStatsProcessor := ingest.StatsChangeProcessor{}

	if err = s.checkIfProtocolVersionSupported(sequence); err != nil {
		err = errors.Wrap(err, "Error while checking for supported protocol version")
		return
	}

	groupChangeProcessors := s.buildChangeProcessor(&changeStatsProcessor, ledgerSource, sequence)
	err = s.runChangeProcessorOnLedger(groupChangeProcessors, sequence)
	if err != nil {
		return
	}

	changeStats = changeStatsProcessor.GetResults()
	changeDurations = groupChangeProcessors.processorsRunDurations

	transactionStats, transactionDurations, err =
		s.RunTransactionProcessorsOnLedger(sequence)
	if err != nil {
		return
	}

	return
}
