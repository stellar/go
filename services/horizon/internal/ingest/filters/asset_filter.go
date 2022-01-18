package filters

import (
	"context"

	"github.com/stellar/go/ingest"
)

type AssetFilterParms struct {
	canonicalAssetList               []string // list of fully qualified asset canonical id <issuer:code>
	assetIssuerList                  []string // list of just the account id as the asset issuer value, 
	                                          // filter will match on any asset reference with same issuer
	assetCodeList                    []string // list of just asset codes, filter will match on any asset reference with same code
	resolveLiquidityPoolAsAsset      bool     // if a liquidity pool references a filtered asset, then include all operation 
	                                          // types within 'traverseOperationsList' that are related to the pool
	traverseEffects                  bool     // true means generate effects for any operations referencing a filtered asset
	traverseOperationsList           []string // list of 'Offers, Trades, Payments, TrustLine, Claimable Balance', include any 
	                                          // of these operation types when they reference a filtered asset 
}

type AssetFilter struct {
	filterParams               *AssetFilterParms
}

func NewAssetFilter(filterParams *AssetFilterParms) *AssetFilter {
	filter := &AssetFilter{
		filterParams:         filterParams,
	}
	return filter
}



func (p *AssetFilter) FilterTransaction(ctx context.Context, transaction ingest.LedgerTransaction) (bool, error) {
	//TODO implement the asset filter based on filter params and intropsectin ops for matching references to asset in tx
    return true, nil
}

