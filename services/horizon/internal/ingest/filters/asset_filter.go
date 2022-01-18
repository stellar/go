package filters

import (
	"context"
	"fmt"

	"github.com/stellar/go/ingest"
)

type AssetFilterParms struct {
	// list of fully qualified asset canonical id <issuer:code>
	canonicalAssetList []string

	// list of just the account id as the asset issuer value,
	// filter will match on any asset reference with same issuer
	assetIssuerList []string

	// list of just asset codes, filter will match on any asset reference with same code
	assetCodeList []string

	// if a liquidity pool references a filtered asset, then include all operation
	// types within 'traverseOperationsList' that are related to the pool
	resolveLiquidityPoolAsAsset bool

	// true means generate effects for any operations referencing a filtered asset
	traverseEffects bool

	// list of 'Offers, Trades, Payments, TrustLine, Claimable Balance', include any
	// of these operation types when they reference a filtered asset
	traverseOperationsList []string
}

type AssetFilter struct {
	filterParams *AssetFilterParms
}

func NewAssetFilter(filterParams *AssetFilterParms) *AssetFilter {
	filter := &AssetFilter{
		filterParams: filterParams,
	}
	return filter
}

func (f *AssetFilter) FilterTransaction(ctx context.Context, transaction ingest.LedgerTransaction) (bool, error) {
	//TODO implement the asset filter based on filter params and intropsectin ops for matching references to asset in tx

	for _, asset := range f.filterParams.canonicalAssetList {
		fmt.Printf("asset %v", asset)
	}

	for _, asset := range f.filterParams.assetIssuerList {
		fmt.Printf("asset issuer %v", asset)
	}

	for _, asset := range f.filterParams.assetCodeList {
		fmt.Printf("asset code %v", asset)
	}

	for _, operation := range f.filterParams.traverseOperationsList {
		fmt.Printf("include operation %v", operation)
	}

	fmt.Printf("resolve pools %v", f.filterParams.resolveLiquidityPoolAsAsset)
	fmt.Printf("include effects %v", f.filterParams.traverseEffects)

	return true, nil
}
