package operations

import (
	"context"
	"fmt"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/httpx"
	"github.com/stellar/go/services/horizon/internal/render/hal"
)

// PagingToken implements hal.Pageable
func (this Base) PagingToken() string {
	return this.PT
}

// Populate fills out this resource using `row` as the source.
func (this *Base) Populate(
	ctx context.Context,
	row history.Operation,
	ledger history.Ledger,
) {
	this.ID = fmt.Sprintf("%d", row.ID)
	this.PT = row.PagingToken()
	this.SourceAccount = row.SourceAccount
	this.populateType(row)
	this.LedgerCloseTime = ledger.ClosedAt
	this.TransactionHash = row.TransactionHash

	lb := hal.LinkBuilder{Base: httpx.BaseURL(ctx)}
	self := fmt.Sprintf("/operations/%d", row.ID)
	this.Links.Self = lb.Link(self)
	this.Links.Succeeds = lb.Linkf("/effects?order=desc&cursor=%s", this.PT)
	this.Links.Precedes = lb.Linkf("/effects?order=asc&cursor=%s", this.PT)
	this.Links.Transaction = lb.Linkf("/transactions/%s", row.TransactionHash)
	this.Links.Effects = lb.Link(self, "effects")
}

func (this *Base) populateType(row history.Operation) {
	var ok bool
	this.TypeI = int32(row.Type)
	this.Type, ok = TypeNames[row.Type]

	if !ok {
		this.Type = "unknown"
	}
}
