package expingest

import (
	"bytes"
	"context"
	"fmt"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/expingest/processors"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
	"github.com/stellar/go/xdr"
)

type ingestionSource int

const (
	_                    = iota
	historyArchiveSource = ingestionSource(iota)
	ledgerSource         = ingestionSource(iota)
)

type horizonTransactionProcessor interface {
	io.LedgerTransactionProcessor
	// TODO maybe rename to Flush()
	Commit() error
}

type ProcessorRunnerInterface interface {
	RunHistoryArchiveIngestion(checkpointLedger uint32) error
	RunAllProcessorsOnLedger(sequence uint32) error
	RunTransactionProcessorsOnLedger(sequence uint32) error
	RunOrderBookProcessorOnLedger(sequence uint32) error
}

var _ ProcessorRunnerInterface = (*ProcessorRunner)(nil)

type ProcessorRunner struct {
	config Config

	ctx            context.Context
	graph          *orderbook.OrderBookGraph
	historyQ       history.IngestionQ
	historyArchive *historyarchive.Archive
	historyAdapter adapters.HistoryArchiveAdapterInterface
	ledgerBackend  *ledgerbackend.DatabaseBackend
}

func (s *ProcessorRunner) buildOrderBookChangeProcessor() horizonChangeProcessor {
	return processors.NewOrderbookProcessor(s.graph)
}

func (s *ProcessorRunner) buildChangeProcessor(source ingestionSource) horizonChangeProcessor {
	useLedgerCache := source == ledgerSource
	return groupChangeProcessors{
		processors.NewOrderbookProcessor(s.graph),
		processors.NewAccountDataProcessor(s.historyQ),
		processors.NewAccountsProcessor(s.historyQ),
		processors.NewOffersProcessor(s.historyQ),
		processors.NewAssetStatsProcessor(s.historyQ, useLedgerCache),
		processors.NewSignersProcessor(s.historyQ, useLedgerCache),
		processors.NewTrustLinesProcessor(s.historyQ),
	}
}

type skipFailedTransactions struct {
	horizonTransactionProcessor
}

func (p skipFailedTransactions) ProcessTransaction(tx io.LedgerTransaction) error {
	if !tx.Successful() {
		return nil
	}
	return p.horizonTransactionProcessor.ProcessTransaction(tx)
}

func (s *ProcessorRunner) buildTransactionProcessor(
	ledger xdr.LedgerHeaderHistoryEntry,
) horizonTransactionProcessor {
	sequence := uint32(ledger.Header.LedgerSeq)
	group := groupTransactionProcessors{
		processors.NewEffectProcessor(s.historyQ, sequence),
		processors.NewLedgerProcessor(s.historyQ, ledger, CurrentVersion),
		processors.NewOperationProcessor(s.historyQ, sequence),
		processors.NewTradeProcessor(s.historyQ, ledger),
		processors.NewParticipantsProcessor(s.historyQ, sequence),
		processors.NewTransactionProcessor(s.historyQ, sequence),
	}

	if s.config.IngestFailedTransactions {
		return group
	}

	return skipFailedTransactions{group}
}

// validateBucketList validates if the bucket list hash in history archive
// matches the one in corresponding ledger header in stellar-core backend.
// This gives you full security if data in stellar-core backend can be trusted
// (ex. you run it in your infrastructure).
// The hashes of actual buckets of this HAS file are checked using
// historyarchive.XdrStream.SetExpectedHash (this is done in
// SingleLedgerStateReader).
func (s *ProcessorRunner) validateBucketList(ledgerSequence uint32) error {
	historyBucketListHash, err := s.historyAdapter.BucketListHash(ledgerSequence)
	if err != nil {
		return errors.Wrap(err, "Error getting bucket list hash")
	}

	ledgerReader, err := io.NewDBLedgerReader(s.ctx, ledgerSequence, s.ledgerBackend)
	if err != nil {
		if err == io.ErrNotFound {
			return fmt.Errorf(
				"cannot validate bucket hash list. Checkpoint ledger (%d) must exist in Stellar-Core database.",
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

func (s *ProcessorRunner) RunHistoryArchiveIngestion(checkpointLedger uint32) error {
	changeProcessor := s.buildChangeProcessor(historyArchiveSource)

	var stateReader io.StateReader
	var err error

	if checkpointLedger == 1 {
		stateReader = &io.GenesisLedgerStateReader{
			NetworkPassphrase: s.config.NetworkPassphrase,
		}
	} else {
		if err = s.validateBucketList(checkpointLedger); err != nil {
			return errors.Wrap(err, "Error validating bucket list from HAS")
		}

		stateReader, err = s.historyAdapter.GetState(
			s.ctx,
			checkpointLedger,
			&io.MemoryTempSet{},
			s.config.MaxStreamRetries,
		)
		if err != nil {
			return errors.Wrap(err, "Error creating HAS reader")
		}
	}
	defer stateReader.Close()

	log.WithField("ledger", checkpointLedger).
		Info("Processing entries from History Archive Snapshot")

	stateReader = newLoggerStateReader(
		stateReader,
		log,
		100000,
	)

	err = io.StreamChanges(changeProcessor, stateReader)
	if err != nil {
		return errors.Wrap(err, "Error streaming changes from HAS")
	}

	err = changeProcessor.Commit()
	if err != nil {
		return errors.Wrap(err, "Error commiting changes from processor")
	}

	return nil
}

func (s *ProcessorRunner) runChangeProcessorOnLedger(
	changeProcessor horizonChangeProcessor, ledger uint32,
) error {
	changeReader, err := io.NewLedgerChangeReader(s.ctx, ledger, s.ledgerBackend)
	if err != nil {
		return errors.Wrap(err, "Error creating ledger change reader")
	}
	if err = io.StreamChanges(changeProcessor, changeReader); err != nil {
		return errors.Wrap(err, "Error streaming changes from ledger")
	}

	err = changeProcessor.Commit()
	if err != nil {
		return errors.Wrap(err, "Error commiting changes from processor")
	}

	return nil
}

func (s *ProcessorRunner) RunTransactionProcessorsOnLedger(ledger uint32) error {
	ledgerReader, err := io.NewDBLedgerReader(s.ctx, ledger, s.ledgerBackend)
	if err != nil {
		return errors.Wrap(err, "Error creating ledger reader")
	}

	txProcessor := s.buildTransactionProcessor(ledgerReader.GetHeader())
	err = io.StreamLedgerTransactions(txProcessor, ledgerReader)
	if err != nil {
		return errors.Wrap(err, "Error streaming changes from ledger")
	}

	err = txProcessor.Commit()
	if err != nil {
		return errors.Wrap(err, "Error commiting changes from processor")
	}

	return nil
}

func (s *ProcessorRunner) RunAllProcessorsOnLedger(ledger uint32) error {
	err := s.runChangeProcessorOnLedger(
		s.buildChangeProcessor(ledgerSource), ledger,
	)
	if err != nil {
		return err
	}

	err = s.RunTransactionProcessorsOnLedger(ledger)
	if err != nil {
		return err
	}

	return nil
}

func (s *ProcessorRunner) RunOrderBookProcessorOnLedger(ledger uint32) error {
	orderBookProcessor := s.buildOrderBookChangeProcessor()
	return s.runChangeProcessorOnLedger(orderBookProcessor, ledger)
}
