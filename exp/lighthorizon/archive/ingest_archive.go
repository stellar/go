package archive

import (
	"context"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/xdr"
)

type ingestArchive struct {
	*ledgerbackend.HistoryArchiveBackend
}

func (ingestArchive) NewLedgerTransactionReaderFromLedgerCloseMeta(networkPassphrase string, ledgerCloseMeta xdr.LedgerCloseMeta) (LedgerTransactionReader, error) {
	ingestReader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(networkPassphrase, ledgerCloseMeta)

	if err != nil {
		return nil, err
	}

	return &ingestTransactionReaderAdaption{ingestReader}, nil
}

type ingestTransactionReaderAdaption struct {
	*ingest.LedgerTransactionReader
}

func (adaptation *ingestTransactionReaderAdaption) Read() (LedgerTransaction, error) {
	tx := LedgerTransaction{}
	ingestLedgerTransaction, err := adaptation.LedgerTransactionReader.Read()
	if err != nil {
		return tx, err
	}

	tx.Index = ingestLedgerTransaction.Index
	tx.Envelope = ingestLedgerTransaction.Envelope
	tx.Result = ingestLedgerTransaction.Result
	tx.FeeChanges = ingestLedgerTransaction.FeeChanges
	tx.UnsafeMeta = ingestLedgerTransaction.UnsafeMeta

	return tx, nil
}

// LightHorizon Archive adaptation based on existing horizon ingest package
func NewIngestArchive(sourceUrl string, networkPassphrase string) (Archive, error) {
	// Simple file os access
	source, err := historyarchive.ConnectBackend(
		sourceUrl,
		historyarchive.ConnectOptions{
			Context:           context.Background(),
			NetworkPassphrase: networkPassphrase,
		},
	)
	if err != nil {
		return nil, err
	}
	ledgerBackend := ledgerbackend.NewHistoryArchiveBackend(source)
	return ingestArchive{ledgerBackend}, nil
}
