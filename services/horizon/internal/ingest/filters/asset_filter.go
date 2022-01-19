package filters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/stellar/go/ingest"
)

type AssetFilterParms struct {
	// list of fully qualified asset canonical id <issuer:code>
	CanonicalAssetList []string

	// list of just the account id as the asset issuer value,
	// filter will match on any asset reference with same issuer
	AssetIssuerList []string

	// list of just asset codes, filter will match on any asset reference with same code
	AssetCodeList []string

	// if a liquidity pool references a filtered asset, then include all operation
	// types within 'traverseOperationsList' that are related to the pool
	ResolveLiquidityPoolAsAsset bool

	// true means generate effects for any operations referencing a filtered asset
	TraverseEffects bool

	// list of 'Offers, Trades, Payments, TrustLine, Claimable Balance', include any
	// of these operation types when they reference a filtered asset
	TraverseOperationsList []string
}

type AssetFilter struct {
	filterParams *AssetFilterParms
}

func NewAssetFilterFromParamsFile(filterParamsFilePath string) (*AssetFilter, error) {
	data, err := os.ReadFile(filterParamsFilePath)
	if err != nil {
		return nil, err
	}

	filterParams := &AssetFilterParms{}
	if err = json.Unmarshal(data, filterParams); err != nil {
		return nil, err
	}

	filter := &AssetFilter{
		filterParams: filterParams,
	}
	return filter, nil
}

func NewAssetFilterFromParams(filterParams *AssetFilterParms) *AssetFilter {
	filter := &AssetFilter{
		filterParams: filterParams,
	}
	return filter
}

func (f *AssetFilter) CurrentFilterParameters() AssetFilterParms {
	return *f.filterParams
}

func (f *AssetFilter) FilterTransaction(ctx context.Context, transaction ingest.LedgerTransaction) (bool, error) {
	//TODO implement the asset filter based on filter params and intropsectin ops for matching references to asset in tx

	for _, asset := range f.filterParams.CanonicalAssetList {
		fmt.Printf("asset %v", asset)
	}

	for _, asset := range f.filterParams.AssetIssuerList {
		fmt.Printf("asset issuer %v", asset)
	}

	for _, asset := range f.filterParams.AssetCodeList {
		fmt.Printf("asset code %v", asset)
	}

	for _, operation := range f.filterParams.TraverseOperationsList {
		fmt.Printf("include operation %v", operation)
	}

	fmt.Printf("resolve pools %v", f.filterParams.ResolveLiquidityPoolAsAsset)
	fmt.Printf("include effects %v", f.filterParams.TraverseEffects)

	return true, nil
}
