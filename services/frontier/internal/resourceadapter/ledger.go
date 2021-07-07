package resourceadapter

import (
	"context"
	"fmt"

	"github.com/xdbfoundation/go/amount"
	protocol "github.com/xdbfoundation/go/protocols/frontier"
	frontierContext "github.com/xdbfoundation/go/services/frontier/internal/context"
	"github.com/xdbfoundation/go/services/frontier/internal/db2/history"
	"github.com/xdbfoundation/go/support/render/hal"
	"github.com/xdbfoundation/go/xdr"
)

func PopulateLedger(ctx context.Context, dest *protocol.Ledger, row history.Ledger) {
	dest.ID = row.LedgerHash
	dest.PT = row.PagingToken()
	dest.Hash = row.LedgerHash
	dest.PrevHash = row.PreviousLedgerHash.String
	dest.Sequence = row.Sequence
	// Default to `transaction_count`
	dest.SuccessfulTransactionCount = row.TransactionCount
	if row.SuccessfulTransactionCount != nil {
		dest.SuccessfulTransactionCount = *row.SuccessfulTransactionCount
	}
	dest.FailedTransactionCount = row.FailedTransactionCount
	dest.OperationCount = row.OperationCount
	dest.TxSetOperationCount = row.TxSetOperationCount
	dest.ClosedAt = row.ClosedAt
	dest.TotalCoins = amount.String(xdr.Int64(row.TotalCoins))
	dest.FeePool = amount.String(xdr.Int64(row.FeePool))
	dest.BaseFee = row.BaseFee
	dest.BaseReserve = row.BaseReserve
	dest.MaxTxSetSize = row.MaxTxSetSize
	dest.ProtocolVersion = row.ProtocolVersion

	if row.LedgerHeaderXDR.Valid {
		dest.HeaderXDR = row.LedgerHeaderXDR.String
	} else {
		dest.HeaderXDR = ""
	}

	self := fmt.Sprintf("/ledgers/%d", row.Sequence)
	lb := hal.LinkBuilder{frontierContext.BaseURL(ctx)}
	dest.Links.Self = lb.Link(self)
	dest.Links.Transactions = lb.PagedLink(self, "transactions")
	dest.Links.Operations = lb.PagedLink(self, "operations")
	dest.Links.Payments = lb.PagedLink(self, "payments")
	dest.Links.Effects = lb.PagedLink(self, "effects")
}
