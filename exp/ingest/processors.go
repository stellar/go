package ingest

import (
	"context"
	stdio "io"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
	"github.com/stellar/go/xdr"
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

func ProcessLedgerTransactions(
	ctx context.Context,
	txProcessor LedgerTransactionProcessor,
	ledgerBackend ledgerbackend.LedgerBackend,
	ledger uint32,
) error {
	ledgerReader, err := io.NewDBLedgerReader(ledger, ledgerBackend)
	if err != nil {
		return errors.Wrapf(
			err,
			"could not get ledger reader for ledger %v",
			ledger,
		)
	}

	if err = txProcessor.Init(ledgerReader.GetHeader().Header); err != nil {
		return errors.Wrapf(
			err,
			"could not initialize sequence processor for ledger %v",
			ledger,
		)
	}

	for {
		var tx io.LedgerTransaction
		tx, err = ledgerReader.Read()
		if err == stdio.EOF {
			break
		}
		if err != nil {
			return errors.Wrapf(err, "could not read transaction in ledger %v", ledger)
		}
		if err = txProcessor.ProcessTransaction(tx); err != nil {
			return errors.Wrapf(
				err,
				"could not process transaction %v in ledger %v",
				tx.Index,
				ledger,
			)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			continue
		}
	}

	if err = txProcessor.Commit(); err != nil {
		return errors.Wrapf(err, "could not commit processor for ledger %v", ledger)
	}

	return nil
}

func ProcessHAS(
	ctx context.Context,
	changeProcessor ChangeProcessor,
	archive historyarchive.ArchiveInterface,
	tempSet io.TempSet,
	maxStreamRetries int,
	ledger uint32,
) error {
	reader, err := io.MakeSingleLedgerStateReader(
		archive,
		tempSet,
		ledger,
		maxStreamRetries,
	)
	if err != nil {
		return errors.Wrap(err, "could not create state reader")
	}

	if err = changeProcessor.Init(ledger); err != nil {
		return errors.Wrapf(
			err,
			"could not initialize cumulative processor for ledger %v",
			ledger,
		)
	}

	for {
		var entryChange xdr.LedgerEntryChange
		entryChange, err = reader.Read()
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

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			continue
		}
	}

	if err = changeProcessor.Commit(); err != nil {
		return errors.Wrapf(err, "could not commit processor for sequence %v", ledger)
	}

	return nil
}

func ProcessLedgerChanges(
	ctx context.Context,
	changeProcessor ChangeProcessor,
	ledgerBackend ledgerbackend.LedgerBackend,
	ledger uint32,
) error {
	changeReader, err := io.NewChangeReader(ledger, ledgerBackend)
	if err != nil {
		return errors.Wrapf(
			err,
			"could not get ledger reader for ledger %v",
			ledger,
		)
	}

	if err = changeProcessor.Init(ledger); err != nil {
		return errors.Wrapf(
			err,
			"could not initialize sequence processor for ledger %v",
			ledger,
		)
	}

	for {
		var change io.Change
		change, err = changeReader.Read()
		if err == stdio.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "could not read from change reader")
		}

		if err = changeProcessor.ProcessChange(change); err != nil {
			return errors.Wrapf(
				err,
				"could not process change in ledger %v",
				ledger,
			)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			continue
		}
	}

	if err = changeProcessor.Commit(); err != nil {
		return errors.Wrapf(err, "could not commit processor for ledger %v", ledger)
	}

	return nil
}
