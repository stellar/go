package ingest

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/filters"
	"github.com/stellar/go/services/horizon/internal/ingest/processors"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	logpkg "github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

type ingestionSource int

const (
	_                               = iota
	historyArchiveSource            = ingestionSource(iota)
	ledgerSource                    = ingestionSource(iota)
	logFrequency                    = 50000
	transactionsFilteredTmpGCPeriod = 5 * time.Minute
)

type horizonChangeProcessor interface {
	processors.ChangeProcessor
	Commit(context.Context) error
}

type horizonTransactionProcessor interface {
	processors.LedgerTransactionProcessor
}

type horizonLazyLoader interface {
	Exec(ctx context.Context, session db.SessionInterface) error
}

type statsChangeProcessor struct {
	*ingest.StatsChangeProcessor
}

func (statsChangeProcessor) Commit(ctx context.Context) error {
	return nil
}

type ledgerStats struct {
	changeStats          ingest.StatsChangeProcessorResults
	changeDurations      processorsRunDurations
	transactionStats     processors.StatsLedgerTransactionProcessorResults
	transactionDurations processorsRunDurations
	tradeStats           processors.TradeStats
}

type ProcessorRunnerInterface interface {
	SetHistoryAdapter(historyAdapter historyArchiveAdapterInterface)
	EnableMemoryStatsLogging()
	DisableMemoryStatsLogging()
	RunGenesisStateIngestion() (ingest.StatsChangeProcessorResults, error)
	RunHistoryArchiveIngestion(
		checkpointLedger uint32,
		skipChecks bool,
		ledgerProtocolVersion uint32,
		bucketListHash xdr.Hash,
	) (ingest.StatsChangeProcessorResults, error)
	RunTransactionProcessorsOnLedger(ledger xdr.LedgerCloseMeta) (
		transactionStats processors.StatsLedgerTransactionProcessorResults,
		transactionDurations processorsRunDurations,
		tradeStats processors.TradeStats,
		err error,
	)
	RunTransactionProcessorsOnLedgers(ledgers []xdr.LedgerCloseMeta) error
	RunAllProcessorsOnLedger(ledger xdr.LedgerCloseMeta) (
		stats ledgerStats,
		err error,
	)
}

var _ ProcessorRunnerInterface = (*ProcessorRunner)(nil)

type ProcessorRunner struct {
	config Config

	ctx                   context.Context
	historyQ              history.IngestionQ
	session               db.SessionInterface
	historyAdapter        historyArchiveAdapterInterface
	logMemoryStats        bool
	filters               filters.Filters
	lastTransactionsTmpGC time.Time
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

func buildChangeProcessor(
	historyQ history.IngestionQ,
	changeStats *ingest.StatsChangeProcessor,
	source ingestionSource,
	ledgerSequence uint32,
	networkPassphrase string,
) *groupChangeProcessors {
	statsChangeProcessor := &statsChangeProcessor{
		StatsChangeProcessor: changeStats,
	}

	useLedgerCache := source == ledgerSource
	return newGroupChangeProcessors([]horizonChangeProcessor{
		statsChangeProcessor,
		processors.NewAccountDataProcessor(historyQ),
		processors.NewAccountsProcessor(historyQ),
		processors.NewOffersProcessor(historyQ, ledgerSequence),
		processors.NewAssetStatsProcessor(historyQ, networkPassphrase, useLedgerCache),
		processors.NewSignersProcessor(historyQ, useLedgerCache),
		processors.NewTrustLinesProcessor(historyQ),
		processors.NewClaimableBalancesChangeProcessor(historyQ),
		processors.NewLiquidityPoolsChangeProcessor(historyQ, ledgerSequence),
	})
}

func (s *ProcessorRunner) buildTransactionProcessor(ledgersProcessor *processors.LedgersProcessor) *groupTransactionProcessors {
	accountLoader := history.NewAccountLoader()
	assetLoader := history.NewAssetLoader()
	lpLoader := history.NewLiquidityPoolLoader()
	cbLoader := history.NewClaimableBalanceLoader()

	lazyLoaders := []horizonLazyLoader{accountLoader, assetLoader, lpLoader, cbLoader}
	statsLedgerTransactionProcessor := processors.NewStatsLedgerTransactionProcessor()

	tradeProcessor := processors.NewTradeProcessor(accountLoader,
		lpLoader, assetLoader, s.historyQ.NewTradeBatchInsertBuilder())

	processors := []horizonTransactionProcessor{
		statsLedgerTransactionProcessor,
		processors.NewEffectProcessor(accountLoader, s.historyQ.NewEffectBatchInsertBuilder(), s.config.NetworkPassphrase),
		ledgersProcessor,
		processors.NewOperationProcessor(s.historyQ.NewOperationBatchInsertBuilder(), s.config.NetworkPassphrase),
		tradeProcessor,
		processors.NewParticipantsProcessor(accountLoader,
			s.historyQ.NewTransactionParticipantsBatchInsertBuilder(), s.historyQ.NewOperationParticipantBatchInsertBuilder()),
		processors.NewTransactionProcessor(s.historyQ.NewTransactionBatchInsertBuilder()),
		processors.NewClaimableBalancesTransactionProcessor(cbLoader,
			s.historyQ.NewTransactionClaimableBalanceBatchInsertBuilder(), s.historyQ.NewOperationClaimableBalanceBatchInsertBuilder()),
		processors.NewLiquidityPoolsTransactionProcessor(lpLoader,
			s.historyQ.NewTransactionLiquidityPoolBatchInsertBuilder(), s.historyQ.NewOperationLiquidityPoolBatchInsertBuilder())}

	return newGroupTransactionProcessors(processors, lazyLoaders, statsLedgerTransactionProcessor, tradeProcessor)
}

func (s *ProcessorRunner) buildTransactionFilterer() *groupTransactionFilterers {
	var f []processors.LedgerTransactionFilterer
	if s.config.EnableIngestionFiltering {
		f = append(f, s.filters.GetFilters(s.historyQ, s.ctx)...)
	}

	return newGroupTransactionFilterers(f)
}

func (s *ProcessorRunner) buildFilteredOutProcessor() *groupTransactionProcessors {
	// when in online mode, the submission result processor must always run (regardless of filtering)
	var p []horizonTransactionProcessor
	if s.config.EnableIngestionFiltering {
		txSubProc := processors.NewTransactionFilteredTmpProcessor(s.historyQ.NewTransactionFilteredTmpBatchInsertBuilder())
		p = append(p, txSubProc)
	}

	return newGroupTransactionProcessors(p, nil, nil, nil)
}

// checkIfProtocolVersionSupported checks if this Horizon version supports the
// protocol version of a ledger with the given sequence number.
func (s *ProcessorRunner) checkIfProtocolVersionSupported(ledgerProtocolVersion uint32) error {
	if ledgerProtocolVersion > MaxSupportedProtocolVersion {
		return fmt.Errorf(
			"This Horizon version does not support protocol version %d. "+
				"The latest supported protocol version is %d. Please upgrade to the latest Horizon version.",
			ledgerProtocolVersion,
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
func (s *ProcessorRunner) validateBucketList(ledgerSequence uint32, ledgerBucketHashList xdr.Hash) error {
	historyBucketListHash, err := s.historyAdapter.BucketListHash(ledgerSequence)
	if err != nil {
		return errors.Wrap(err, "Error getting bucket list hash")
	}

	if !bytes.Equal(historyBucketListHash[:], ledgerBucketHashList[:]) {
		return fmt.Errorf(
			"Bucket list hash of history archive and ledger header does not match: %#x %#x",
			historyBucketListHash,
			ledgerBucketHashList,
		)
	}

	return nil
}

func (s *ProcessorRunner) RunGenesisStateIngestion() (ingest.StatsChangeProcessorResults, error) {
	return s.RunHistoryArchiveIngestion(1, false, 0, xdr.Hash{})
}

func (s *ProcessorRunner) RunHistoryArchiveIngestion(
	checkpointLedger uint32,
	skipChecks bool,
	ledgerProtocolVersion uint32,
	bucketListHash xdr.Hash,
) (ingest.StatsChangeProcessorResults, error) {
	changeStats := ingest.StatsChangeProcessor{}
	changeProcessor := buildChangeProcessor(
		s.historyQ,
		&changeStats,
		historyArchiveSource,
		checkpointLedger,
		s.config.NetworkPassphrase,
	)

	if checkpointLedger == 1 {
		if err := changeProcessor.ProcessChange(s.ctx, ingest.GenesisChange(s.config.NetworkPassphrase)); err != nil {
			return changeStats.GetResults(), errors.Wrap(err, "Error ingesting genesis ledger")
		}
	} else {
		if !skipChecks {
			if err := s.checkIfProtocolVersionSupported(ledgerProtocolVersion); err != nil {
				return changeStats.GetResults(), errors.Wrap(err, "Error while checking for supported protocol version")
			}

			if err := s.validateBucketList(checkpointLedger, bucketListHash); err != nil {
				return changeStats.GetResults(), errors.Wrap(err, "Error validating bucket list from HAS")
			}
		}

		changeReader, err := s.historyAdapter.GetState(s.ctx, checkpointLedger)
		if err != nil {
			return changeStats.GetResults(), errors.Wrap(err, "Error creating HAS reader")
		}

		defer changeReader.Close()

		log.WithField("sequence", checkpointLedger).
			Info("Processing entries from History Archive Snapshot")

		err = processors.StreamChanges(s.ctx, changeProcessor, newloggingChangeReader(
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

	if err := changeProcessor.Commit(s.ctx); err != nil {
		return changeStats.GetResults(), errors.Wrap(err, "Error committing changes from processor")
	}

	return changeStats.GetResults(), nil
}

func (s *ProcessorRunner) runChangeProcessorOnLedger(
	changeProcessor horizonChangeProcessor, ledger xdr.LedgerCloseMeta,
) error {
	var changeReader ingest.ChangeReader
	var err error
	changeReader, err = ingest.NewLedgerChangeReaderFromLedgerCloseMeta(s.config.NetworkPassphrase, ledger)
	if err != nil {
		return errors.Wrap(err, "Error creating ledger change reader")
	}
	changeReader = newloggingChangeReader(
		changeReader,
		"ledger",
		ledger.LedgerSequence(),
		logFrequency,
		s.logMemoryStats,
	)
	if err = processors.StreamChanges(s.ctx, changeProcessor, changeReader); err != nil {
		return errors.Wrap(err, "Error streaming changes from ledger")
	}

	err = changeProcessor.Commit(s.ctx)
	if err != nil {
		return errors.Wrap(err, "Error committing changes from processor")
	}

	return nil
}

func (s *ProcessorRunner) streamLedger(ledger xdr.LedgerCloseMeta,
	groupFilterers *groupTransactionFilterers,
	groupFilteredOutProcessors *groupTransactionProcessors,
	groupProcessors *groupTransactionProcessors) error {
	var (
		transactionReader *ingest.LedgerTransactionReader
	)

	if err := s.checkIfProtocolVersionSupported(ledger.ProtocolVersion()); err != nil {
		err = errors.Wrap(err, "Error while checking for supported protocol version")
		return err
	}

	startTime := time.Now()
	transactionReader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(s.config.NetworkPassphrase, ledger)
	if err != nil {
		err = errors.Wrap(err, "Error creating ledger reader")
		return err
	}

	err = processors.StreamLedgerTransactions(s.ctx,
		groupFilterers,
		groupFilteredOutProcessors,
		groupProcessors,
		transactionReader,
		ledger,
	)
	if err != nil {
		return errors.Wrap(err, "Error streaming changes from ledger")
	}

	transactionStats := groupProcessors.transactionStatsProcessor.GetResults()
	transactionStats.TransactionsFiltered = groupFilterers.droppedTransactions

	tradeStats := groupProcessors.tradeProcessor.GetStats()

	log.WithFields(transactionStats.Map()).
		WithFields(tradeStats.Map()).
		WithFields(logpkg.F{
			"sequence": ledger.LedgerSequence(),
			"state":    false,
			"ledger":   true,
			"commit":   false,
			"duration": time.Since(startTime).Seconds(),
		}).Info("Processed ledger")

	return nil
}

func (s *ProcessorRunner) RunTransactionProcessorsOnLedger(ledger xdr.LedgerCloseMeta) (
	transactionStats processors.StatsLedgerTransactionProcessorResults,
	transactionDurations processorsRunDurations,
	tradeStats processors.TradeStats,
	err error,
) {
	// ensure capture of the ledger to history regardless of whether it has transactions.
	ledgersProcessor := processors.NewLedgerProcessor(s.historyQ.NewLedgerBatchInsertBuilder(), CurrentVersion)
	ledgersProcessor.ProcessLedger(ledger)

	groupTransactionFilterers := s.buildTransactionFilterer()
	groupFilteredOutProcessors := s.buildFilteredOutProcessor()
	groupTransactionProcessors := s.buildTransactionProcessor(ledgersProcessor)

	err = s.streamLedger(ledger,
		groupTransactionFilterers,
		groupFilteredOutProcessors,
		groupTransactionProcessors,
	)
	if err != nil {
		err = errors.Wrap(err, "Error streaming changes from ledger")
		return
	}

	err = s.flushProcessors(groupFilteredOutProcessors, groupTransactionProcessors)
	if err != nil {
		return
	}

	transactionStats = groupTransactionProcessors.transactionStatsProcessor.GetResults()
	transactionStats.TransactionsFiltered = groupTransactionFilterers.droppedTransactions

	transactionDurations = groupTransactionProcessors.processorsRunDurations
	for key, duration := range groupFilteredOutProcessors.processorsRunDurations {
		transactionDurations[key] = duration
	}
	for key, duration := range groupTransactionFilterers.processorsRunDurations {
		transactionDurations[key] = duration
	}

	tradeStats = groupTransactionProcessors.tradeProcessor.GetStats()

	return
}

func (s *ProcessorRunner) RunTransactionProcessorsOnLedgers(ledgers []xdr.LedgerCloseMeta) (err error) {
	ledgersProcessor := processors.NewLedgerProcessor(s.historyQ.NewLedgerBatchInsertBuilder(), CurrentVersion)

	groupTransactionFilterers := s.buildTransactionFilterer()
	groupFilteredOutProcessors := s.buildFilteredOutProcessor()
	groupTransactionProcessors := s.buildTransactionProcessor(ledgersProcessor)

	startTime := time.Now()
	curHeap, sysHeap := getMemStats()
	log.WithFields(logpkg.F{
		"currentHeapSizeMB": curHeap,
		"systemHeapSizeMB":  sysHeap,
		"ledgerBatchSize":   len(ledgers),
		"state":             false,
		"ledger":            true,
		"commit":            false,
	}).Infof("Running processors for batch of %v ledgers", len(ledgers))

	for _, ledger := range ledgers {
		// ensure capture of the ledger to history regardless of whether it has transactions.
		ledgersProcessor.ProcessLedger(ledger)

		err = s.streamLedger(ledger,
			groupTransactionFilterers,
			groupFilteredOutProcessors,
			groupTransactionProcessors,
		)
		if err != nil {
			err = errors.Wrap(err, "Error streaming changes during ledger batch")
			return
		}
		groupTransactionProcessors.ResetStats()
		groupFilteredOutProcessors.ResetStats()
		groupTransactionFilterers.ResetStats()
	}

	err = s.flushProcessors(groupFilteredOutProcessors, groupTransactionProcessors)
	if err != nil {
		return
	}
	curHeap, sysHeap = getMemStats()
	log.WithFields(logpkg.F{
		"currentHeapSizeMB": curHeap,
		"systemHeapSizeMB":  sysHeap,
		"ledgers":           len(ledgers),
		"state":             false,
		"ledger":            true,
		"commit":            false,
		"duration":          time.Since(startTime).Seconds(),
	}).Infof("Flushed processors for batch of %v ledgers", len(ledgers))

	return nil
}

func (s *ProcessorRunner) flushProcessors(groupFilteredOutProcessors *groupTransactionProcessors, groupTransactionProcessors *groupTransactionProcessors) (err error) {
	if s.config.EnableIngestionFiltering {
		err = groupFilteredOutProcessors.Flush(s.ctx, s.session)
		if err != nil {
			err = errors.Wrap(err, "Error flushing temp filtered tx from processor")
			return
		}
		if time.Since(s.lastTransactionsTmpGC) > transactionsFilteredTmpGCPeriod {
			s.historyQ.DeleteTransactionsFilteredTmpOlderThan(s.ctx, uint64(transactionsFilteredTmpGCPeriod.Seconds()))
		}
	}

	err = groupTransactionProcessors.Flush(s.ctx, s.session)
	if err != nil {
		err = errors.Wrap(err, "Error flushing changes from processor")
	}
	return
}

func (s *ProcessorRunner) RunAllProcessorsOnLedger(ledger xdr.LedgerCloseMeta) (
	stats ledgerStats,
	err error,
) {
	changeStatsProcessor := ingest.StatsChangeProcessor{}

	if err = s.checkIfProtocolVersionSupported(ledger.ProtocolVersion()); err != nil {
		err = errors.Wrap(err, "Error while checking for supported protocol version")
		return
	}

	groupChangeProcessors := buildChangeProcessor(
		s.historyQ,
		&changeStatsProcessor,
		ledgerSource,
		ledger.LedgerSequence(),
		s.config.NetworkPassphrase,
	)
	err = s.runChangeProcessorOnLedger(groupChangeProcessors, ledger)
	if err != nil {
		return
	}

	transactionStats, transactionDurations, tradeStats, err := s.RunTransactionProcessorsOnLedger(ledger)

	stats.changeStats = changeStatsProcessor.GetResults()
	stats.changeDurations = groupChangeProcessors.processorsRunDurations
	stats.transactionStats = transactionStats
	stats.transactionDurations = transactionDurations
	stats.tradeStats = tradeStats

	return
}
