package ledgerbackend

import (
	"context"
	"fmt"

	"github.com/stellar/go/metaarchive"
	"github.com/stellar/go/xdr"
)

type HistoryArchiveBackend struct {
	metaArchive *metaarchive.MetaArchive
}

func NewHistoryArchiveBackend(metaArchive *metaarchive.MetaArchive) *HistoryArchiveBackend {
	return &HistoryArchiveBackend{
		metaArchive: metaArchive,
	}
}

func (b *HistoryArchiveBackend) GetLatestLedgerSequence(ctx context.Context) (uint32, error) {
	return b.metaArchive.GetLatestLedgerSequence(ctx)
}

func (b *HistoryArchiveBackend) PrepareRange(ctx context.Context, ledgerRange Range) error {
	// Noop
	return nil
}

func (b *HistoryArchiveBackend) IsPrepared(ctx context.Context, ledgerRange Range) (bool, error) {
	// Noop
	return true, nil
}

func (b *HistoryArchiveBackend) GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error) {
	serializedLedger, err := b.metaArchive.GetLedger(ctx, sequence)
	if err != nil {
		return xdr.LedgerCloseMeta{}, err
	}

	output, isV0 := serializedLedger.GetV0()
	if !isV0 {
		return xdr.LedgerCloseMeta{}, fmt.Errorf("unexpected serialized ledger version number (0x%x)", serializedLedger.V)
	}
	return output, nil
}

func (b *HistoryArchiveBackend) Close() error {
	// Noop
	return nil
}
