package resource

import (
	"context"

	"github.com/stellar/go/services/horizon/internal/httpx"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/render/hal"
)

// Populate fills in the details
func (res *Root) Populate(
	ctx context.Context,
	ledgerState ledger.State,
	hVersion, cVersion string,
	passphrase string,
	pVersion int32,
) {
	res.HorizonSequence = ledgerState.HistoryLatest
	res.HistoryElderSequence = ledgerState.HistoryElder
	res.CoreSequence = ledgerState.CoreLatest
	res.HorizonVersion = hVersion
	res.StellarCoreVersion = cVersion
	res.NetworkPassphrase = passphrase
	res.ProtocolVersion = pVersion

	lb := hal.LinkBuilder{httpx.BaseURL(ctx)}
	res.Links.Account = lb.Link("/accounts/{account_id}")
	res.Links.AccountTransactions = lb.PagedLink("/accounts/{account_id}/transactions")
	res.Links.Assets = lb.Link("/assets{?asset_code,asset_issuer,cursor,limit,order}")
	res.Links.Friendbot = lb.Link("/friendbot{?addr}")
	res.Links.Metrics = lb.Link("/metrics")
	res.Links.OrderBook = lb.Link("/order_book{?selling_asset_type,selling_asset_code,selling_issuer,buying_asset_type,buying_asset_code,buying_issuer,limit}")
	res.Links.Self = lb.Link("/")
	res.Links.Transaction = lb.Link("/transactions/{hash}")
	res.Links.Transactions = lb.PagedLink("/transactions")
}
