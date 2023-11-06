package history

import (
	"context"

	"github.com/stellar/go/support/db"
)

// ClaimableBalanceClaimantBatchInsertBuilder is used to insert claimants into the
// claimable_balance_claimants table
type ClaimableBalanceClaimantBatchInsertBuilder interface {
	Add(claimableBalanceClaimant ClaimableBalanceClaimant) error
	Exec(ctx context.Context) error
}

// ClaimableBalanceClaimantBatchInsertBuilder is a simple wrapper around db.FastBatchInsertBuilder
type claimableBalanceClaimantBatchInsertBuilder struct {
	session db.SessionInterface
	builder db.FastBatchInsertBuilder
	table   string
}

// NewClaimableBalanceClaimantBatchInsertBuilder constructs a new ClaimableBalanceClaimantBatchInsertBuilder instance
func (q *Q) NewClaimableBalanceClaimantBatchInsertBuilder(session db.SessionInterface) ClaimableBalanceClaimantBatchInsertBuilder {
	return &claimableBalanceClaimantBatchInsertBuilder{
		session: session,
		builder: db.FastBatchInsertBuilder{},
		table:   "claimable_balance_claimants",
	}
}

// Add adds a new claimant to the batch
func (i *claimableBalanceClaimantBatchInsertBuilder) Add(claimableBalanceClaimant ClaimableBalanceClaimant) error {
	return i.builder.RowStruct(claimableBalanceClaimant)
}

// Exec writes the batch of claimants to the database.
func (i *claimableBalanceClaimantBatchInsertBuilder) Exec(ctx context.Context) error {
	return i.builder.Exec(ctx, i.session, i.table)
}
