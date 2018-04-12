package resource

import (
	"context"
	"fmt"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/httpx"
	"github.com/stellar/go/services/horizon/internal/render/hal"
	"github.com/stellar/go/xdr"
)

func (this *Ledger) Populate(ctx context.Context, row history.Ledger) {
	this.ID = row.LedgerHash
	this.PT = row.PagingToken()
	this.Hash = row.LedgerHash
	this.PrevHash = row.PreviousLedgerHash.String
	this.Sequence = row.Sequence
	this.TransactionCount = row.TransactionCount
	this.OperationCount = row.OperationCount
	this.ClosedAt = row.ClosedAt
	this.TotalCoins = amount.String(xdr.Int64(row.TotalCoins))
	this.FeePool = amount.String(xdr.Int64(row.FeePool))
	this.BaseFee = row.BaseFee
	this.BaseReserve = row.BaseReserve
	this.MaxTxSetSize = row.MaxTxSetSize
	this.ProtocolVersion = row.ProtocolVersion

	if row.LedgerHeaderXDR.Valid {
		this.HeaderXDR = row.LedgerHeaderXDR.String
	} else {
		this.HeaderXDR = ""
	}

	self := fmt.Sprintf("/ledgers/%d", row.Sequence)
	lb := hal.LinkBuilder{httpx.BaseURL(ctx)}
	this.Links.Self = lb.Link(self)
	this.Links.Transactions = lb.PagedLink(self, "transactions")
	this.Links.Operations = lb.PagedLink(self, "operations")
	this.Links.Payments = lb.PagedLink(self, "payments")
	this.Links.Effects = lb.PagedLink(self, "effects")

	return
}

func (this Ledger) PagingToken() string {
	return this.PT
}
