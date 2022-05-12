package processors

import (
	"context"
	"time"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
)

type TxSubmissionResultProcessor struct {
	txSubmissionResultQ history.QTxSubmissionResult
	ledger              xdr.LedgerHeaderHistoryEntry
	txs                 []ingest.LedgerTransaction
}

func NewTxSubmissionResultProcessor(
	txSubmissionResultQ history.QTxSubmissionResult,
	ledger xdr.LedgerHeaderHistoryEntry,
) *TxSubmissionResultProcessor {
	return &TxSubmissionResultProcessor{
		ledger:              ledger,
		txSubmissionResultQ: txSubmissionResultQ,
	}
}

func (p *TxSubmissionResultProcessor) ProcessTransaction(ctx context.Context, transaction ingest.LedgerTransaction) (err error) {
	p.txs = append(p.txs, transaction)

	return nil
}

func (p *TxSubmissionResultProcessor) Commit(ctx context.Context) error {
	seq := uint32(p.ledger.Header.LedgerSeq)
	closeTime := time.Unix(int64(p.ledger.Header.ScpValue.CloseTime), 0).UTC()
	_, err := p.txSubmissionResultQ.SetTxSubmissionResults(ctx, p.txs, seq, closeTime)
	return err
}
