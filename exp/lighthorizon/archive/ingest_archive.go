package archive

import (
	"context"

	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/xdr"
)

// This is an implementation of LightHorizon Archive that uses the existing horizon ingestion backend.
type ingestArchive struct {
	*ledgerbackend.HistoryArchiveBackend
}

func (a ingestArchive) NewLedgerTransactionReaderFromLedgerCloseMeta(networkPassphrase string, ledgerCloseMeta xdr.LedgerCloseMeta) (LedgerTransactionReader, error) {
	ingestReader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(networkPassphrase, ledgerCloseMeta)

	if err != nil {
		return nil, err
	}

	return &ingestTransactionReaderAdaption{ingestReader}, nil
}

func (a ingestArchive) GetTransactionParticipants(transaction LedgerTransaction) (map[string]struct{}, error) {
	participants, err := index.GetTransactionParticipants(a.ingestTx(transaction))
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{})
	exists := struct{}{}
	for _, participant := range participants {
		set[participant] = exists
	}
	return set, nil
}

func (a ingestArchive) GetOperationParticipants(transaction LedgerTransaction, operation xdr.Operation, opIndex int) (map[string]struct{}, error) {
	participants, err := index.GetOperationParticipants(a.ingestTx(transaction), operation, opIndex)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{})
	exists := struct{}{}
	for _, participant := range participants {
		set[participant] = exists
	}
	return set, nil
}

func (ingestArchive) ingestTx(transaction LedgerTransaction) ingest.LedgerTransaction {
	tx := ingest.LedgerTransaction{}
	tx.Index = transaction.Index
	tx.Envelope = transaction.Envelope
	tx.Result = transaction.Result
	tx.FeeChanges = transaction.FeeChanges
	tx.UnsafeMeta = transaction.UnsafeMeta
	return tx
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
