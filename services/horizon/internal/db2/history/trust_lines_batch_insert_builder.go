package history

import (
	"context"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// Add adds a new trust line entry to the batch. `lastModifiedLedger` is another
// parameter because `xdr.TrustLineEntry` does not have a field to hold this value.
func (i *trustLinesBatchInsertBuilder) Add(ctx context.Context, entry xdr.LedgerEntry) error {
	m := trustLineToMap(entry)

	// Add lkey only when inserting rows
	key, err := trustLineEntryToLedgerKeyString(entry)
	if err != nil {
		return errors.Wrap(err, "Error running trustLineEntryToLedgerKeyString")
	}
	m["ledger_key"] = key

	return i.builder.Row(ctx, m)
}

func (i *trustLinesBatchInsertBuilder) Exec(ctx context.Context) error {
	return i.builder.Exec(ctx)
}
