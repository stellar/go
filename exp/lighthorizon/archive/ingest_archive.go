package archive

import (
	"context"
	"net/url"

	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/xdr"
)

const (
	maxLedgersToCache = (60 * 60 * 24) / 6 // 1 day of ledgers @ 6s each
)

type ArchiveConfig struct {
	SourceUrl         string
	NetworkPassphrase string
	CacheDir          string
}

func NewIngestArchive(config ArchiveConfig) (Archive, error) {
	// If the source URL is an S3 url and it has a region specified, we should
	// try to extract it.
	parsed, err := url.Parse(config.SourceUrl)
	if err != nil {
		return nil, errors.Wrapf(err, "%s is not a valid URL", config.SourceUrl)
	}
	region := ""
	if parsed.Scheme == "s3" {
		region = parsed.Query().Get("region")
	}

	// Now, set up a simple filesystem-like access to the backend and wrap it in
	// a local on-disk LRU cache if we can.
	source, err := historyarchive.ConnectBackend(
		config.SourceUrl,
		historyarchive.ConnectOptions{
			Context:           context.Background(),
			NetworkPassphrase: config.NetworkPassphrase,
			S3Region:          region,
		},
	)
	if err != nil {
		return nil, err
	}

	cache, err := historyarchive.MakeFsCacheBackend(source, config.CacheDir, maxLedgersToCache)
	if err != nil { // warn but continue w/o cache
		log.WithField("path", config.CacheDir).
			WithError(err).
			Warnf("Failed to create cached ledger backend")
		cache = source
	} else {
		log.WithField("path", config.CacheDir).Infof("On-disk cache configured")
	}

	ledgerBackend := ledgerbackend.NewHistoryArchiveBackend(cache)
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

var _ Archive = (*ingestArchive)(nil) // ensure conformity to the interface
