package ingester

import (
	"context"
	"fmt"
	"net/url"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/metaarchive"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/storage"
	"github.com/stellar/go/xdr"
)

//
// LightHorizon data model
//

// Ingester combines a source of unpacked ledger metadata and a way to create a
// ingestion reader interface on top of it.
type Ingester interface {
	metaarchive.MetaArchive

	PrepareRange(ctx context.Context, r historyarchive.Range) error
	NewLedgerTransactionReader(
		ledgerCloseMeta xdr.SerializedLedgerCloseMeta,
	) (LedgerTransactionReader, error)
}

// For now, this mirrors the `ingest` library exactly, but it's replicated so
// that we can diverge in the future if necessary.
type LedgerTransaction struct {
	*ingest.LedgerTransaction
}

type LedgerTransactionReader interface {
	Read() (LedgerTransaction, error)
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

	if config.ParallelDownloads > 1 {
		log.Infof("Enabling parallel ledger fetches with %d workers", config.ParallelDownloads)
		return NewParallelIngester(
			metaarchive.NewMetaArchive(source),
			config.NetworkPassphrase,
			config.ParallelDownloads), nil
	}

	return &liteIngester{
		MetaArchive:       metaarchive.NewMetaArchive(source),
		networkPassphrase: config.NetworkPassphrase,
	}, nil
}
