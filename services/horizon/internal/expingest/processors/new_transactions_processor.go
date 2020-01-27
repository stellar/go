package processors

import (
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
)

type NewTransactionProcessor struct {
	TransactionsQ history.QTransactions

	sequence uint32
	batch    history.TransactionBatchInsertBuilder
}

func (p *NewTransactionProcessor) Init(header xdr.LedgerHeader) error {
	p.sequence = uint32(header.LedgerSeq)
	p.batch = p.TransactionsQ.NewTransactionBatchInsertBuilder(maxBatchSize)
	return nil
}

func (p *NewTransactionProcessor) ProcessTransaction(transaction io.LedgerTransaction) error {
	return p.batch.Add(transaction, p.sequence)
}

func (p *NewTransactionProcessor) Commit() error {
	return p.batch.Exec()
}
