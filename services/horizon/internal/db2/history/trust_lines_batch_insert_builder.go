package history

import (
	"context"
)

// Add adds a new trust line to the batch
func (i *trustLinesBatchInsertBuilder) Add(ctx context.Context, trustline TrustLine) error {
	return i.builder.RowStruct(ctx, trustline)
}

func (i *trustLinesBatchInsertBuilder) Exec(ctx context.Context) error {
	return i.builder.Exec(ctx)
}
