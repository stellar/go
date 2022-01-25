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

	for _, operation := range tx.Tx.Operations {
		var allowed = false
		switch operation.Body.Type {
		case xdr.OperationTypeChangeTrust:
			if pool, ok := operation.Body.ChangeTrustOp.Line.GetLiquidityPool(); ok {
				if f.assetMatchedFilter(&pool.ConstantProduct.AssetA) || f.assetMatchedFilter(&pool.ConstantProduct.AssetB) {
					allowed = true
				}
			} else {
				asset := operation.Body.ChangeTrustOp.Line.ToAsset()
				allowed = f.assetMatchedFilter(&asset)
			}
		case xdr.OperationTypeClaimClaimableBalance:
			// TODO, try to get asset for claimable balance id
		case xdr.OperationTypeClawbackClaimableBalance:
			// TODO, try to get asset for claimable balance id
		case xdr.OperationTypeManageSellOffer:
			if f.assetMatchedFilter(&operation.Body.ManageSellOfferOp.Buying) || f.assetMatchedFilter(&operation.Body.ManageSellOfferOp.Selling) {
				allowed = true
			}
		case xdr.OperationTypeManageBuyOffer:
			if f.assetMatchedFilter(&operation.Body.ManageBuyOfferOp.Buying) || f.assetMatchedFilter(&operation.Body.ManageBuyOfferOp.Selling) {
				allowed = true
			}
		case xdr.OperationTypeCreateClaimableBalance:
			if f.assetMatchedFilter(&operation.Body.CreateClaimableBalanceOp.Asset) {
				allowed = true
			}
		case xdr.OperationTypeCreatePassiveSellOffer:
			if f.assetMatchedFilter(&operation.Body.CreatePassiveSellOfferOp.Buying) || f.assetMatchedFilter(&operation.Body.CreatePassiveSellOfferOp.Selling) {
				allowed = true
			}
		case xdr.OperationTypeClawback:
			if f.assetMatchedFilter(&operation.Body.ClawbackOp.Asset) {
				allowed = true
			}
		case xdr.OperationTypePayment:
			if f.assetMatchedFilter(&operation.Body.PaymentOp.Asset) {
				allowed = true
			}
		case xdr.OperationTypePathPaymentStrictReceive:
			if f.assetMatchedFilter(&operation.Body.PathPaymentStrictReceiveOp.DestAsset) || f.assetMatchedFilter(&operation.Body.PathPaymentStrictReceiveOp.SendAsset) {
				allowed = true
			}
		case xdr.OperationTypePathPaymentStrictSend:
			if f.assetMatchedFilter(&operation.Body.PathPaymentStrictSendOp.DestAsset) || f.assetMatchedFilter(&operation.Body.PathPaymentStrictSendOp.SendAsset) {
				allowed = true
			}
		case xdr.OperationTypeLiquidityPoolDeposit:
			if f.includeLiquidityPool(operation.Body.LiquidityPoolDepositOp.LiquidityPoolId) {
				allowed = true
			}
		case xdr.OperationTypeLiquidityPoolWithdraw:
			if f.includeLiquidityPool(operation.Body.LiquidityPoolWithdrawOp.LiquidityPoolId) {
				allowed = true
			}
		}

		if allowed && f.operationMatchedFilterDepth(operation.Body.Type) {
			return true, nil
		}
	}

	logger.Debugf("No match, dropped tx with seq %v ", transaction.Envelope.SeqNum())
	return false, nil
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
			if f.assetMatchedFilter(asset) {
				return true
			}
		}
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
