package expingest

import (
	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	stdio "io"
)

type ChangeProcessor interface {
	Init(sequence uint32) error
	ProcessChange(change io.Change) error
	Commit() error
}

type LedgerTransactionProcessor interface {
	Init(header xdr.LedgerHeader) error
	ProcessTransaction(transaction io.LedgerTransaction) error
	Commit() error
}

type cpGroup []ChangeProcessor

func (g cpGroup) Init(sequence uint32) error {
	for _, p := range g {
		if err := p.Init(sequence); err != nil {
			return err
		}
	}
	return nil
}

func (g cpGroup) ProcessChange(change io.Change) error {
	for _, p := range g {
		if err := p.ProcessChange(change); err != nil {
			return err
		}
	}
	return nil
}

func (g cpGroup) Commit() error {
	for _, p := range g {
		if err := p.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func GroupChangeProcessors(
	processors ...ChangeProcessor,
) ChangeProcessor {
	return cpGroup(processors)
}

type ltGroup []LedgerTransactionProcessor

func (g ltGroup) Init(header xdr.LedgerHeader) error {
	for _, p := range g {
		if err := p.Init(header); err != nil {
			return err
		}
	}
	return nil
}

func (g ltGroup) ProcessTransaction(transaction io.LedgerTransaction) error {
	for _, p := range g {
		if err := p.ProcessTransaction(transaction); err != nil {
			return err
		}
	}
	return nil
}

func (g ltGroup) Commit() error {
	for _, p := range g {
		if err := p.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func GroupLedgerTransactionProcessors(
	processors ...LedgerTransactionProcessor,
) LedgerTransactionProcessor {
	return ltGroup(processors)
}

func RunTransactionProcessorOnLedger(
	ledgerAdapter *adapters.LedgerBackendAdapter,
	ltProcessor LedgerTransactionProcessor,
	ledger uint32,
) error {
	ledgerReader, err := ledgerAdapter.GetLedger(ledger)
	if err != nil {
		return errors.Wrapf(
			err,
			"could not get ledger reader for ledger %v",
			ledger,
		)
	}
	ledgerReader.IgnoreUpgradeChanges()

	if err := ltProcessor.Init(ledgerReader.GetHeader().Header); err != nil {
		return errors.Wrapf(
			err,
			"could not initialize sequence processor for ledger %v",
			ledger,
		)
	}

	for {
		tx, err := ledgerReader.Read()
		if err == stdio.EOF {
			break
		}
		if err != nil {
			return errors.Wrapf(err, "could not read transaction in ledger %v", ledger)
		}
		if err = ltProcessor.ProcessTransaction(tx); err != nil {
			return errors.Wrapf(
				err,
				"could not process transaction %v in ledger %v",
				tx.Index,
				ledger,
			)
		}
	}

	if err := ltProcessor.Commit(); err != nil {
		return errors.Wrapf(err, "could not commit processor for ledger %v", ledger)
	}

	return nil
}

func RunChangeProcessorOnHAS(
	changeProcessor ChangeProcessor,
	reader io.StateReader,
) error {
	ledger := reader.GetSequence()
	if err := changeProcessor.Init(ledger); err != nil {
		return errors.Wrapf(
			err,
			"could not initialize cumulative processor for ledger %v",
			ledger,
		)
	}

	for {
		entryChange, err := reader.Read()
		if err == stdio.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "could not read from state reader")
		}

		if err = changeProcessor.ProcessChange(io.Change{
			Type: entryChange.EntryType(),
			Pre:  nil,
			Post: entryChange.Created,
		}); err != nil {
			return errors.Wrapf(
				err,
				"could not process change in ledger %v",
				ledger,
			)
		}
	}

	if err := changeProcessor.Commit(); err != nil {
		return errors.Wrapf(err, "could not commit processor for sequence %v", ledger)
	}

	return nil
}

func changesFromLedger(
	ledgerAdapter *adapters.LedgerBackendAdapter, ledger uint32,
) (*io.LedgerEntryChangeCache, error) {
	ledgerCache := io.NewLedgerEntryChangeCache()

	r, err := ledgerAdapter.GetLedger(ledger)
	if err != nil {
		return ledgerCache, err
	}

	// Get all transactions
	var transactions []io.LedgerTransaction
	for {
		var transaction io.LedgerTransaction
		transaction, err = r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return ledgerCache, err
			}
		}

		transactions = append(transactions, transaction)
	}

	// Remember that it's possible that transaction can remove a preauth
	// tx signer even when it's a failed transaction so we need to check
	// failed transactions too.

	// Fees are processed before everything else.
	for _, transaction := range transactions {
		for _, change := range transaction.GetFeeChanges() {
			err = ledgerCache.AddChange(change)
			if err != nil {
				return ledgerCache, err
			}
		}
	}

	// Tx meta
	for _, transaction := range transactions {
		var changes []io.Change
		changes, err = transaction.GetChanges()
		if err != nil {
			return ledgerCache, err
		}
		for _, change := range changes {
			err = ledgerCache.AddChange(change)
			if err != nil {
				return ledgerCache, err
			}
		}
	}

	// Process upgrades meta
	for {
		var change io.Change
		change, err = r.ReadUpgradeChange()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return ledgerCache, err
			}
		}

		err = ledgerCache.AddChange(change)
		if err != nil {
			return ledgerCache, err
		}
	}

	return ledgerCache, nil
}

func RunChangeProcessorOnLedger(
	ledgerAdapter *adapters.LedgerBackendAdapter,
	changeProcessor ChangeProcessor,
	ledger uint32,
) error {
	ledgerCache, err := changesFromLedger(ledgerAdapter, ledger)
	if err != nil {
		return err
	}

	if err := changeProcessor.Init(ledger); err != nil {
		return errors.Wrapf(
			err,
			"could not initialize sequence processor for ledger %v",
			ledger,
		)
	}

	for _, change := range ledgerCache.GetChanges() {
		if err = changeProcessor.ProcessChange(change); err != nil {
			return errors.Wrapf(
				err,
				"could not process change in ledger %v",
				ledger,
			)
		}

	}

	if err := changeProcessor.Commit(); err != nil {
		return errors.Wrapf(err, "could not commit processor for ledger %v", ledger)
	}

	return nil
}
