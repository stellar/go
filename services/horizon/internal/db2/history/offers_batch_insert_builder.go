package history

import (
	"context"
)

// Add adds a new offer entry to the batch.
func (i *offersBatchInsertBuilder) Add(ctx context.Context, offer Offer) error {
	return i.builder.RowStruct(ctx, offer)
}

func (i *offersBatchInsertBuilder) Exec(ctx context.Context) error {
	return i.builder.Exec(ctx)
}
