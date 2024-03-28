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
	Name() string
	Commit(context.Context) error
}

type horizonTransactionProcessor interface {
	Name() string
	processors.LedgerTransactionProcessor
}

type horizonLazyLoader interface {
	Exec(ctx context.Context, session db.SessionInterface) error
	Name() string
	Stats() history.LoaderStats
}

type statsChangeProcessor struct {
	*ingest.StatsChangeProcessor
}

func (statsChangeProcessor) Name() string {
	return "ingest.statsChangeProcessor"
}

func (statsChangeProcessor) Commit(ctx context.Context) error {
	return nil
}

type ledgerStats struct {
	changeStats          ingest.StatsChangeProcessorResults
	changeDurations      runDurations
	transactionStats     processors.StatsLedgerTransactionProcessorResults
	transactionDurations runDurations
	loaderDurations      runDurations
	loaderStats          map[string]history.LoaderStats
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
	RunTransactionProcessorsOnLedgers(ledgers []xdr.LedgerCloseMeta, execInTx bool) error
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

	return newGroupChangeProcessors([]horizonChangeProcessor{
		statsChangeProcessor,
		processors.NewAccountDataProcessor(historyQ),
		processors.NewAccountsProcessor(historyQ),
		processors.NewOffersProcessor(historyQ, ledgerSequence),
		processors.NewAssetStatsProcessor(historyQ, networkPassphrase, source == historyArchiveSource, ledgerSequence),
		processors.NewSignersProcessor(historyQ),
		processors.NewTrustLinesProcessor(historyQ),
		processors.NewClaimableBalancesChangeProcessor(historyQ),
		processors.NewLiquidityPoolsChangeProcessor(historyQ, ledgerSequence),
	})
}

func (s *ProcessorRunner) buildTransactionProcessor(ledgersProcessor *processors.LedgersProcessor) (groupLoaders, *groupTransactionProcessors) {
	accountLoader := history.NewAccountLoader()
	assetLoader := history.NewAssetLoader()
	lpLoader := history.NewLiquidityPoolLoader()
	cbLoader := history.NewClaimableBalanceLoader()

	loaders := newGroupLoaders([]horizonLazyLoader{accountLoader, assetLoader, lpLoader, cbLoader})
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
		processors.NewTransactionProcessor(s.historyQ.NewTransactionBatchInsertBuilder(), s.config.SkipTxmeta),
		processors.NewClaimableBalancesTransactionProcessor(cbLoader,
			s.historyQ.NewTransactionClaimableBalanceBatchInsertBuilder(), s.historyQ.NewOperationClaimableBalanceBatchInsertBuilder()),
		processors.NewLiquidityPoolsTransactionProcessor(lpLoader,
			s.historyQ.NewTransactionLiquidityPoolBatchInsertBuilder(), s.historyQ.NewOperationLiquidityPoolBatchInsertBuilder())}

	return loaders, newGroupTransactionProcessors(processors, statsLedgerTransactionProcessor, tradeProcessor)
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
		txSubProc := processors.NewTransactionFilteredTmpProcessor(s.historyQ.NewTransactionFilteredTmpBatchInsertBuilder(), s.config.SkipTxmeta)
		p = append(p, txSubProc)
	}

	return newGroupTransactionProcessors(p, nil, nil)
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

	if err := registerChangeProcessors(
		nameRegistry{},
		changeProcessor,
	); err != nil {
		return ingest.StatsChangeProcessorResults{}, err
	}

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
	changeReader = ingest.NewCompactingChangeReader(changeReader)
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
		}).Info("Transaction processors finished for ledger")

	return nil
}

func (s *ProcessorRunner) runTransactionProcessorsOnLedger(registry nameRegistry, ledger xdr.LedgerCloseMeta) (
	transactionStats processors.StatsLedgerTransactionProcessorResults,
	transactionDurations runDurations,
	tradeStats processors.TradeStats,
	loaderDurations runDurations,
	loaderStats map[string]history.LoaderStats,
	err error,
) {
	// ensure capture of the ledger to history regardless of whether it has transactions.
	ledgersProcessor := processors.NewLedgerProcessor(s.historyQ.NewLedgerBatchInsertBuilder(), CurrentVersion)
	ledgersProcessor.ProcessLedger(ledger)

	groupTransactionFilterers := s.buildTransactionFilterer()
	groupFilteredOutProcessors := s.buildFilteredOutProcessor()
	loaders, groupTransactionProcessors := s.buildTransactionProcessor(ledgersProcessor)

	if err = registerTransactionProcessors(
		registry,
		loaders,
		groupTransactionFilterers,
		groupFilteredOutProcessors,
		groupTransactionProcessors,
	); err != nil {
		return
	}

	err = s.streamLedger(ledger,
		groupTransactionFilterers,
		groupFilteredOutProcessors,
		groupTransactionProcessors,
	)
	if err != nil {
		err = errors.Wrap(err, "Error streaming changes from ledger")
		return
	}

	if err = loaders.Flush(s.ctx, s.session, false); err != nil {
		return
	}

	err = s.flushProcessors(groupFilteredOutProcessors, groupTransactionProcessors, false)
	if err != nil {
		return
	}

	transactionStats = groupTransactionProcessors.transactionStatsProcessor.GetResults()
	transactionStats.TransactionsFiltered = groupTransactionFilterers.droppedTransactions

	transactionDurations = groupTransactionProcessors.processorsRunDurations
	for key, duration := range groupFilteredOutProcessors.processorsRunDurations {
		transactionDurations[key] = duration
	}
	loaderStats = loaders.stats
	loaderDurations = loaders.runDurations
	for key, duration := range groupTransactionFilterers.runDurations {
		transactionDurations[key] = duration
	}

	tradeStats = groupTransactionProcessors.tradeProcessor.GetStats()

	return
}

// nameRegistry ensures all ingestion components have a unique name
// for metrics reporting
type nameRegistry map[string]struct{}

func (n nameRegistry) add(name string) error {
	if _, ok := n[name]; ok {
		return fmt.Errorf("%s is duplicated", name)
	}
	n[name] = struct{}{}
	return nil
}

func registerChangeProcessors(
	registry nameRegistry,
	group *groupChangeProcessors,
) error {
	for _, p := range group.processors {
		if err := registry.add(p.Name()); err != nil {
			return err
		}
	}
	return nil
}

func registerTransactionProcessors(
	registry nameRegistry,
	loaders groupLoaders,
	groupTransactionFilterers *groupTransactionFilterers,
	groupFilteredOutProcessors *groupTransactionProcessors,
	groupTransactionProcessors *groupTransactionProcessors,
) error {
	for _, f := range groupTransactionFilterers.filterers {
		if err := registry.add(f.Name()); err != nil {
			return err
		}
	}
	for _, p := range groupTransactionProcessors.processors {
		if err := registry.add(p.Name()); err != nil {
			return err
		}
	}
	for _, l := range loaders.lazyLoaders {
		if err := registry.add(l.Name()); err != nil {
			return err
		}
	}
	for _, p := range groupFilteredOutProcessors.processors {
		if err := registry.add(p.Name()); err != nil {
			return err
		}
	}
	return nil
}

func (s *ProcessorRunner) RunTransactionProcessorsOnLedgers(ledgers []xdr.LedgerCloseMeta, execInTx bool) (err error) {
	ledgersProcessor := processors.NewLedgerProcessor(s.historyQ.NewLedgerBatchInsertBuilder(), CurrentVersion)

	groupTransactionFilterers := s.buildTransactionFilterer()
	groupFilteredOutProcessors := s.buildFilteredOutProcessor()
	loaders, groupTransactionProcessors := s.buildTransactionProcessor(ledgersProcessor)

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

	if err = loaders.Flush(s.ctx, s.session, execInTx); err != nil {
		return
	}

	err = s.flushProcessors(groupFilteredOutProcessors, groupTransactionProcessors, execInTx)
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

func (s *ProcessorRunner) flushProcessors(groupFilteredOutProcessors *groupTransactionProcessors, groupTransactionProcessors *groupTransactionProcessors, execInTx bool) error {
	if execInTx {
		if err := s.session.Begin(s.ctx); err != nil {
			return err
		}
		defer s.session.Rollback()
	}

	if s.config.EnableIngestionFiltering {

		if err := groupFilteredOutProcessors.Flush(s.ctx, s.session); err != nil {
			return errors.Wrap(err, "Error flushing temp filtered tx from processor")
		}
		if time.Since(s.lastTransactionsTmpGC) > transactionsFilteredTmpGCPeriod {
			if _, err := s.historyQ.DeleteTransactionsFilteredTmpOlderThan(s.ctx, uint64(transactionsFilteredTmpGCPeriod.Seconds())); err != nil {
				return errors.Wrap(err, "Error trimming filtered transactions")
			}
		}
	}

	if err := groupTransactionProcessors.Flush(s.ctx, s.session); err != nil {
		return errors.Wrap(err, "Error flushing changes from processor")
	}

	if execInTx {
		if err := s.session.Commit(); err != nil {
			return err
		}
	}
	return nil
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

	registry := nameRegistry{}
	if err = registerChangeProcessors(
		registry,
		groupChangeProcessors,
	); err != nil {
		return
	}

	err = s.runChangeProcessorOnLedger(groupChangeProcessors, ledger)
	if err != nil {
		return
	}

	transactionStats, transactionDurations, tradeStats, loaderDurations, loaderStats, err := s.runTransactionProcessorsOnLedger(registry, ledger)

	stats.changeStats = changeStatsProcessor.GetResults()
	stats.changeDurations = groupChangeProcessors.processorsRunDurations
	stats.transactionStats = transactionStats
	stats.transactionDurations = transactionDurations
	stats.tradeStats = tradeStats
	stats.loaderDurations = loaderDurations
	stats.loaderStats = loaderStats

	return
}
