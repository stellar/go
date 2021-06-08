package history

import (
	"context"

	"github.com/stellar/go/support/db"
	"github.com/stellar/go/xdr"
)

// accountsBatchInsertBuilder is a simple wrapper around db.BatchInsertBuilder
type accountsBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

func (i *accountsBatchInsertBuilder) Add(ctx context.Context, entry xdr.LedgerEntry) error {
	return i.builder.Row(ctx, accountToMap(entry))
}

func (i *accountsBatchInsertBuilder) Exec(ctx context.Context) error {
	return i.builder.Exec(ctx)
}
