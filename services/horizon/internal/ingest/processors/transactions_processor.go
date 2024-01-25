package processors

import (
	"context"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type TransactionProcessor struct {
	batch       history.TransactionBatchInsertBuilder
	skipSoroban bool
}

func NewTransactionFilteredTmpProcessor(batch history.TransactionBatchInsertBuilder, skipSoroban bool) *TransactionProcessor {
	return &TransactionProcessor{
		batch:       batch,
		skipSoroban: skipSoroban,
	}
}

func NewTransactionProcessor(batch history.TransactionBatchInsertBuilder, skipSoroban bool) *TransactionProcessor {
	return &TransactionProcessor{
		batch:       batch,
		skipSoroban: skipSoroban,
	}
}

func (p *TransactionProcessor) ProcessTransaction(lcm xdr.LedgerCloseMeta, transaction ingest.LedgerTransaction) error {
	elidedTransaction := transaction

	if p.skipSoroban &&
		elidedTransaction.UnsafeMeta.V == 3 &&
		elidedTransaction.UnsafeMeta.MustV3().SorobanMeta != nil {
		elidedTransaction.UnsafeMeta.V3 = &xdr.TransactionMetaV3{
			Ext:             xdr.ExtensionPoint{},
			TxChangesBefore: xdr.LedgerEntryChanges{},
			Operations:      []xdr.OperationMeta{},
			TxChangesAfter:  xdr.LedgerEntryChanges{},
			SorobanMeta:     nil,
		}
	}

	if err := p.batch.Add(elidedTransaction, lcm.LedgerSequence()); err != nil {
		return errors.Wrap(err, "Error batch inserting transaction rows")
	}

	return nil
}

func (p *TransactionProcessor) Flush(ctx context.Context, session db.SessionInterface) error {
	return p.batch.Exec(ctx, session)
}
