package archive

import (
	"context"
	"fmt"
	"net/url"

	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/metaarchive"
	"github.com/stellar/go/support/collections/set"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/storage"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/xdr"
)

type ArchiveConfig struct {
	SourceUrl         string
	NetworkPassphrase string
	CacheDir          string
	CacheSize         int
}

func NewIngestArchive(config ArchiveConfig) (Archive, error) {
	if config.CacheSize <= 0 {
		return nil, fmt.Errorf("invalid cache size: %d", config.CacheSize)
	}

	parsed, err := url.Parse(config.SourceUrl)
	if err != nil {
		return nil, errors.Wrapf(err, "%s is not a valid URL", config.SourceUrl)
	}

	region := ""
	needsCache := true
	switch parsed.Scheme {
	case "file":
		// We should only avoid a cache if the ledgers are already local.
		needsCache = false

	case "s3":
		// We need to extract the region if it's specified.
		region = parsed.Query().Get("region")
	}

	// Now, set up a simple filesystem-like access to the backend and wrap it in
	// a local on-disk LRU cache if we can.
	source, err := historyarchive.ConnectBackend(
		config.SourceUrl,
		storage.ConnectOptions{
			Context:  context.Background(),
			S3Region: region,
		},
	)
	if err != nil {
		return nil, err
	}

	if needsCache {
		cache, err := storage.MakeOnDiskCache(source,
			config.CacheDir, uint(config.CacheSize))

		if err != nil { // warn but continue w/o cache
			log.WithField("path", config.CacheDir).
				WithError(err).
				Warnf("Failed to create cached ledger backend")
		} else {
			log.WithField("path", config.CacheDir).
				Infof("On-disk cache configured")
			source = cache
		}
	}

	metaArchive := metaarchive.NewMetaArchive(source)

	ledgerBackend := ledgerbackend.NewHistoryArchiveBackend(metaArchive)
	return ingestArchive{ledgerBackend}, nil
}

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

func (a ingestArchive) GetTransactionParticipants(tx LedgerTransaction) (set.Set[string], error) {
	participants, err := index.GetTransactionParticipants(a.ingestTx(tx))
	if err != nil {
		return nil, err
	}

	s := set.NewSet[string](len(participants))
	s.AddSlice(participants)
	return s, nil
}

func (a ingestArchive) GetOperationParticipants(tx LedgerTransaction, op xdr.Operation, opIndex int) (set.Set[string], error) {
	participants, err := index.GetOperationParticipants(a.ingestTx(tx), op, opIndex)
	if err != nil {
		return nil, err
	}

	s := set.NewSet[string](len(participants))
	s.AddSlice(participants)
	return s, nil
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

var _ Archive = (*ingestArchive)(nil) // ensure conformity to the interface
