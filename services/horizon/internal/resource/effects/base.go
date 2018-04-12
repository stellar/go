package effects

import (
	"context"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/httpx"
	"github.com/stellar/go/services/horizon/internal/render/hal"
)

// PagingToken implements `hal.Pageable`
func (this Base) PagingToken() string {
	return this.PT
}

// Populate loads this resource from `row`
func (this *Base) Populate(ctx context.Context, row history.Effect, ledger history.Ledger) {
	this.ID = row.ID()
	this.PT = row.PagingToken()
	this.Account = row.Account
	this.populateType(row)
	this.LedgerCloseTime = ledger.ClosedAt

	lb := hal.LinkBuilder{httpx.BaseURL(ctx)}
	this.Links.Operation = lb.Linkf("/operations/%d", row.HistoryOperationID)
	this.Links.Succeeds = lb.Linkf("/effects?order=desc&cursor=%s", this.PT)
	this.Links.Precedes = lb.Linkf("/effects?order=asc&cursor=%s", this.PT)
}

func (this *Base) populateType(row history.Effect) {
	var ok bool
	this.TypeI = int32(row.Type)
	this.Type, ok = TypeNames[row.Type]

	if !ok {
		this.Type = "unknown"
	}
}
