package filters

import (
	"context"
	"encoding/json"
	"os"

	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"

	"github.com/stellar/go/ingest"
)

var logger *log.Entry

func init() {
	logger = log.WithFields(log.F{
		"ingest filter": "asset",
	})
}

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

	// true means the filter will be executed during ingestion, false
	// means the filter is disabled, it will have no effect.
	Activated bool

	// list of 'Offers, Trades, Payments, TrustLine, Claimable Balance', include any
	// of these operation types when they reference a filtered asset. If empty, means
	// include all operations.
	TraverseOperationsList []string
}

type AssetFilter struct {
	filterParams             *AssetFilterParms
	canonicalAssetsLookup    map[string]bool
	assetIssuersLookup       map[string]bool
	assetCodesLookup         map[string]bool
	traverseOperationsLookup map[string]bool
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

	return NewAssetFilterFromParams(filterParams), nil
}

func NewAssetFilterFromParams(filterParams *AssetFilterParms) *AssetFilter {
	filter := &AssetFilter{
		filterParams:             filterParams,
		canonicalAssetsLookup:    listToMap(filterParams.CanonicalAssetList),
		assetIssuersLookup:       listToMap(filterParams.AssetIssuerList),
		assetCodesLookup:         listToMap(filterParams.AssetCodeList),
		traverseOperationsLookup: listToMap(filterParams.TraverseOperationsList),
	}
	return filter
}

func (f *AssetFilter) CurrentFilterParameters() AssetFilterParms {
	return *f.filterParams
}

func (f *AssetFilter) FilterTransaction(ctx context.Context, transaction ingest.LedgerTransaction) (bool, error) {
	if !f.filterParams.Activated {
		return true, nil
	}

	tx, v1Exists := transaction.Envelope.GetV1()
	if !v1Exists {
		return true, nil
	}

	var allowedOperations []*xdr.Operation

	for _, operation := range tx.Tx.Operations {
		var allowed = true
		switch operation.Body.Type {
		case xdr.OperationTypePayment:
			if !f.assetMatchedFilter(&operation.Body.PaymentOp.Asset) {
				allowed = false
			}
		case xdr.OperationTypePathPaymentStrictReceive:
			if !f.assetMatchedFilter(&operation.Body.PathPaymentStrictReceiveOp.DestAsset) && !f.assetMatchedFilter(&operation.Body.PathPaymentStrictReceiveOp.SendAsset) {
				allowed = false
			}
		case xdr.OperationTypeLiquidityPoolDeposit:
			if !f.includeLiquidityPool(operation.Body.LiquidityPoolDepositOp.LiquidityPoolId) {
				allowed = false
			}

		case xdr.OperationTypeLiquidityPoolWithdraw:
			if !f.includeLiquidityPool(operation.Body.LiquidityPoolWithdrawOp.LiquidityPoolId) {
				allowed = false
			}
		}

		if allowed && f.operationMatchedFilterDepth(operation.Body.Type) {
			allowedOperations = append(allowedOperations, &operation)
		}
	}

	logger.Debugf("filter status %v for tx seq %v ", len(allowedOperations), transaction.Envelope.SeqNum())
	return len(allowedOperations) > 0, nil
}

func (f *AssetFilter) operationMatchedFilterDepth(operationType xdr.OperationType) bool {
	if len(f.traverseOperationsLookup) < 1 {
		return true
	}
	_, found := f.traverseOperationsLookup[operationType.String()]
	return found
}

func (f *AssetFilter) assetMatchedFilter(asset *xdr.Asset) bool {

	var matched = false

	if _, found := f.canonicalAssetsLookup[asset.StringCanonical()]; found {
		matched = true
	} else if _, found := f.assetCodesLookup[asset.GetCode()]; found {
		matched = true
	} else if _, found := f.assetIssuersLookup[asset.GetIssuer()]; found {
		matched = true
	}

	return matched
}

func (f *AssetFilter) includeLiquidityPool(poolId xdr.PoolId) bool {
	if f.filterParams.ResolveLiquidityPoolAsAsset {
		assets := f.resolveLiquidityPoolToAssetPair(poolId)
		for _, asset := range assets {
			if !f.assetMatchedFilter(asset) {
				return false
			}
		}
		return true
	}
	return false
}

func (f *AssetFilter) resolveLiquidityPoolToAssetPair(poolId xdr.PoolId) []*xdr.Asset {
	//TODO - implement the resolutiuon to asset pair
	return []*xdr.Asset{}
}

func listToMap(list []string) map[string]bool {
	set := make(map[string]bool)
	for i := 0; i < len(list); i++ {
		set[list[i]] = true
	}
	return set
}
