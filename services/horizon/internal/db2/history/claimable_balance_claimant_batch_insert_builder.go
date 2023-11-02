package history

import (
	"context"

	"github.com/stellar/go/support/db"
	"github.com/stellar/go/xdr"
)

// ClaimableBalanceClaimantBatchInsertBuilder is used to insert claimants into the
// claimable_balance_claimants table
type ClaimableBalanceClaimantBatchInsertBuilder interface {
	Add(claimableBalanceClaimant ClaimableBalanceClaimant) error
	Exec(ctx context.Context, session db.SessionInterface) error
	Reset()
}

// ClaimableBalanceClaimantBatchInsertBuilder is a simple wrapper around db.FastBatchInsertBuilder
type claimableBalanceClaimantBatchInsertBuilder struct {
	encodingBuffer *xdr.EncodingBuffer
	builder        db.FastBatchInsertBuilder
	table          string
}

// NewClaimableBalanceClaimantBatchInsertBuilder constructs a new ClaimableBalanceClaimantBatchInsertBuilder instance
func (q *Q) NewClaimableBalanceClaimantBatchInsertBuilder() ClaimableBalanceClaimantBatchInsertBuilder {
	return &claimableBalanceClaimantBatchInsertBuilder{
		encodingBuffer: xdr.NewEncodingBuffer(),
		builder:        db.FastBatchInsertBuilder{},
		table:          "claimable_balance_claimants",
	}
}

// Add adds a new claimant to the batch
func (i *claimableBalanceClaimantBatchInsertBuilder) Add(claimableBalanceClaimant ClaimableBalanceClaimant) error {
	return i.builder.RowStruct(claimableBalanceClaimant)
}

// Exec writes the batch of claimants to the database.
func (i *claimableBalanceClaimantBatchInsertBuilder) Exec(ctx context.Context, session db.SessionInterface) error {
	return i.builder.Exec(ctx, session, i.table)
}

// Reset clears out the current batch of claimants
func (i *claimableBalanceClaimantBatchInsertBuilder) Reset() {
	i.builder.Reset()
}
