package expingest

import (
	"bytes"
	"fmt"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/expingest/processors"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type ingestionSource int

const (
	_                    = iota
	historyArchiveSource = ingestionSource(iota)
	ledgerSource         = ingestionSource(iota)
)

type horizonChangeProcessor interface {
	io.ChangeProcessor
	// TODO maybe rename to Flush()
	Commit() error
}

type groupChangeProcessors []horizonChangeProcessor

func (g groupChangeProcessors) ProcessChange(change io.Change) error {
	for _, p := range g {
		if err := p.ProcessChange(change); err != nil {
			return err
		}
	}
	return nil
}

func (g groupChangeProcessors) Commit() error {
	for _, p := range g {
		if err := p.Commit(); err != nil {
			return err
		}
	}
	return nil
}

type horizonTransactionProcessor interface {
	io.LedgerTransactionProcessor
	// TODO maybe rename to Flush()
	Commit() error
}

type groupTransactionProcessors []horizonTransactionProcessor

func (g groupTransactionProcessors) ProcessTransaction(tx io.LedgerTransaction) error {
	for _, p := range g {
		if err := p.ProcessTransaction(tx); err != nil {
			return err
		}
	}
	return nil
}

func (g groupTransactionProcessors) Commit() error {
	for _, p := range g {
		if err := p.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (s *System) buildOrderBookChangeProcessor() horizonChangeProcessor {
	return processors.NewOrderbookProcessor(s.graph)
}

func (s *System) buildChangeProcessor(source ingestionSource) horizonChangeProcessor {
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

func (s *System) buildTransactionProcessor(
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
func (s *System) validateBucketList(ledgerSequence uint32) error {
	historyBucketListHash, err := s.historyAdapter.BucketListHash(ledgerSequence)
	if err != nil {
		return errors.Wrap(err, "Error getting bucket list hash")
	}

	ledgerReader, err := io.NewDBLedgerReader(ledgerSequence, s.ledgerBackend)
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

func (s *System) runHistoryArchiveIngestion(
	checkpointLedger uint32,
) error {
	changeProcessor := s.buildChangeProcessor(historyArchiveSource)

	var stateReader io.ChangeReader
	var err error

	if checkpointLedger == 1 {
		stateReader = &io.GenesisLedgerStateReader{
			NetworkPassphrase: s.config.NetworkPassphrase,
		}
	} else {
		if err = s.validateBucketList(checkpointLedger); err != nil {
			return errors.Wrap(err, "Error validating bucket list from HAS")
		}

		stateReader, err = io.MakeSingleLedgerStateReader(
			s.historyArchive,
			&io.MemoryTempSet{},
			checkpointLedger,
			s.maxStreamRetries,
		)
	}
	if err != nil {
		return errors.Wrap(err, "Error creating HAS reader")
	}

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

func (s *System) runChangeProcessorOnLedger(
	changeProcessor horizonChangeProcessor, ledger uint32,
) error {
	changeReader, err := io.NewLedgerChangeReader(ledger, s.ledgerBackend)
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

func (s *System) runTransactionProcessorsOnLedger(ledger uint32) error {
	ledgerReader, err := io.NewDBLedgerReader(ledger, s.ledgerBackend)
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

func (s *System) runAllProcessorsOnLedger(ledger uint32) error {
	err := s.runChangeProcessorOnLedger(
		s.buildChangeProcessor(ledgerSource), ledger,
	)
	if err != nil {
		return err
	}

	err = s.runTransactionProcessorsOnLedger(ledger)
	if err != nil {
		return err
	}

	return nil
}
