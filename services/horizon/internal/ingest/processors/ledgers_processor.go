package processors

import (
	"context"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/xdr"
)

type ledgerInfo struct {
	header         xdr.LedgerHeaderHistoryEntry
	successTxCount int
	failedTxCount  int
	opCount        int
	txSetOpCount   int
}

type LedgersProcessor struct {
	batch         history.LedgerBatchInsertBuilder
	ledgers       map[uint32]*ledgerInfo
	ingestVersion int
}

func NewLedgerProcessor(batch history.LedgerBatchInsertBuilder, ingestVersion int) *LedgersProcessor {
	return &LedgersProcessor{
		batch:         batch,
		ledgers:       map[uint32]*ledgerInfo{},
		ingestVersion: ingestVersion,
	}
}

func (p *LedgersProcessor) ProcessTransaction(lcm xdr.LedgerCloseMeta, transaction ingest.LedgerTransaction) error {
	sequence := lcm.LedgerSequence()
	entry, ok := p.ledgers[sequence]
	if !ok {
		entry = &ledgerInfo{header: lcm.LedgerHeaderHistoryEntry()}
		p.ledgers[sequence] = entry
	}

	opCount := len(transaction.Envelope.Operations())
	entry.txSetOpCount += opCount
	if transaction.Result.Successful() {
		entry.successTxCount++
		entry.opCount += opCount
	} else {
		entry.failedTxCount++
	}

	return nil
}

func (p *LedgersProcessor) Commit(ctx context.Context, session db.SessionInterface) error {
	for _, entry := range p.ledgers {
		err := p.batch.Add(
			entry.header,
			entry.successTxCount,
			entry.failedTxCount,
			entry.opCount,
			entry.txSetOpCount,
			p.ingestVersion,
		)
		if err != nil {
			return err
		}
	}

	if err := p.batch.Exec(ctx, session); err != nil {
		return err
	}

	return nil
}
