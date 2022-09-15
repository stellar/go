package ingester

import (
	"context"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/metaarchive"

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

func (i *liteIngester) PrepareRange(ctx context.Context, r historyarchive.Range) error {
	return nil
}

func (i *liteIngester) NewLedgerTransactionReader(
	ledgerCloseMeta xdr.SerializedLedgerCloseMeta,
) (LedgerTransactionReader, error) {
	reader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(
		i.networkPassphrase,
		ledgerCloseMeta.MustV0())

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
