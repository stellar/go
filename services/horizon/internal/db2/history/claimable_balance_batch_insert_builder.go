package history

import (
	"context"

	"github.com/stellar/go/support/db"
	"github.com/stellar/go/xdr"
)

// ClaimableBalanceBatchInsertBuilder is used to insert claimable balance into the
// claimable_balances table
type ClaimableBalanceBatchInsertBuilder interface {
	Add(claimableBalance ClaimableBalance) error
	Exec(ctx context.Context, session db.SessionInterface) error
	Reset()
}

// ClaimableBalanceBatchInsertBuilder is a simple wrapper around db.FastBatchInsertBuilder
type claimableBalanceBatchInsertBuilder struct {
	encodingBuffer *xdr.EncodingBuffer
	builder        db.FastBatchInsertBuilder
	table          string
}

// NewClaimableBalanceBatchInsertBuilder constructs a new ClaimableBalanceBatchInsertBuilder instance
func (q *Q) NewClaimableBalanceBatchInsertBuilder() ClaimableBalanceBatchInsertBuilder {
	return &claimableBalanceBatchInsertBuilder{
		encodingBuffer: xdr.NewEncodingBuffer(),
		builder:        db.FastBatchInsertBuilder{},
		table:          "claimable_balances",
	}
}

// Add adds a new claimable balance to the batch
func (i *claimableBalanceBatchInsertBuilder) Add(claimableBalance ClaimableBalance) error {
	return i.builder.RowStruct(claimableBalance)
}

// Exec writes the batch of claimable balances to the database.
func (i *claimableBalanceBatchInsertBuilder) Exec(ctx context.Context, session db.SessionInterface) error {
	return i.builder.Exec(ctx, session, i.table)
}

// Reset clears out the current batch of claimable balances
func (i *claimableBalanceBatchInsertBuilder) Reset() {
	i.builder.Reset()
}
