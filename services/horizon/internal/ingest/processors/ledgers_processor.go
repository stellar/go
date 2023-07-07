package processors

import (
	"context"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type LedgersProcessor struct {
	session        db.SessionInterface
	ledgersQ       history.QLedgers
	ledger         xdr.LedgerHeaderHistoryEntry
	ingestVersion  int
	successTxCount int
	failedTxCount  int
	opCount        int
	txSetOpCount   int
}

func NewLedgerProcessor(
	session db.SessionInterface,
	ledgerQ history.QLedgers,
	ledger xdr.LedgerHeaderHistoryEntry,
	ingestVersion int,
) *LedgersProcessor {
	return &LedgersProcessor{
		session:       session,
		ledger:        ledger,
		ledgersQ:      ledgerQ,
		ingestVersion: ingestVersion,
	}
}

func (p *LedgersProcessor) ProcessTransaction(ctx context.Context, transaction ingest.LedgerTransaction) (err error) {
	opCount := len(transaction.Envelope.Operations())
	p.txSetOpCount += opCount
	if transaction.Result.Successful() {
		p.successTxCount++
		p.opCount += opCount
	} else {
		p.failedTxCount++
	}

	return nil
}

func (p *LedgersProcessor) Commit(ctx context.Context) error {
	batch := p.ledgersQ.NewLedgerBatchInsertBuilder()
	err := batch.Add(p.ledger, p.successTxCount, p.failedTxCount, p.opCount, p.txSetOpCount, p.ingestVersion)
	if err != nil {
		return errors.Wrap(err, "Could not insert ledger")
	}

	if err = batch.Exec(ctx, p.session); err != nil {
		return errors.Wrap(err, "Could not commit ledger")
	}

	return nil
}
