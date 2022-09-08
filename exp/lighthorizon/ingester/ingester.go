package ingester

import (
	"context"
	"fmt"
	"net/url"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/metaarchive"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/storage"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/xdr"
)

type IngesterConfig struct {
	SourceUrl         string
	NetworkPassphrase string

	CacheDir  string
	CacheSize int

	ParallelDownloads uint
}

type liteIngester struct {
	metaarchive.MetaArchive
	networkPassphrase string
}

func NewIngester(config IngesterConfig) (Ingester, error) {
	if config.CacheSize <= 0 {
		return nil, fmt.Errorf("invalid cache size: %d", config.CacheSize)
	}

	// Now, set up a simple filesystem-like access to the backend and wrap it in
	// a local on-disk LRU cache if we can.
	source, err := historyarchive.ConnectBackend(
		config.SourceUrl,
		storage.ConnectOptions{Context: context.Background()},
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to %s", config.SourceUrl)
	}

	parsed, err := url.Parse(config.SourceUrl)
	if err != nil {
		return nil, errors.Wrapf(err, "%s is not a valid URL", config.SourceUrl)
	}

	if parsed.Scheme != "file" { // otherwise, already on-disk
		cache, errr := storage.MakeOnDiskCache(source, config.CacheDir, uint(config.CacheSize))

		if errr != nil { // non-fatal: warn but continue w/o cache
			log.WithField("path", config.CacheDir).WithError(errr).
				Warnf("Failed to create cached ledger backend")
		} else {
			log.WithField("path", config.CacheDir).
				Infof("On-disk cache configured")
			source = cache
		}
	}

	return &liteIngester{
		MetaArchive:       metaarchive.NewMetaArchive(source),
		networkPassphrase: config.NetworkPassphrase,
	}, nil
}

func (i *liteIngester) NewLedgerTransactionReader(
	ledgerCloseMeta xdr.SerializedLedgerCloseMeta,
) (LedgerTransactionReader, error) {
	reader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(
		i.networkPassphrase,
		ledgerCloseMeta.MustV0())
	if err != nil {
		return nil, err
	}

	return &liteLedgerTransactionReader{reader}, err
}

type liteLedgerTransactionReader struct {
	*ingest.LedgerTransactionReader
}

func (reader *liteLedgerTransactionReader) Read() (LedgerTransaction, error) {
	ingestedTx, err := reader.LedgerTransactionReader.Read()
	if err != nil {
		return LedgerTransaction{}, err
	}
	return LedgerTransaction{LedgerTransaction: &ingestedTx}, nil
}

var _ Ingester = (*liteIngester)(nil) // ensure conformity to the interface
var _ LedgerTransactionReader = (*liteLedgerTransactionReader)(nil)
