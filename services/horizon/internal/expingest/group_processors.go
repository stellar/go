package expingest

import "github.com/stellar/go/exp/ingest/io"

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
