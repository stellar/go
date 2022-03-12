package filters

import (
	"context"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/processors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"

	"github.com/stellar/go/ingest"
)

var (
	logger = log.WithFields(log.F{
		"ingest filter": "asset",
	})
)

type assetFilter struct {
	canonicalAssetsLookup map[string]struct{}
	lastModified          int64
	enabled               bool
}

type AssetFilter interface {
	processors.LedgerTransactionFilterer
	RefreshAssetFilter(filterConfig *history.AssetFilterConfig) error
}

func NewAssetFilter() AssetFilter {
	return &assetFilter{
		canonicalAssetsLookup: map[string]struct{}{},
	}
}

func (filter *assetFilter) RefreshAssetFilter(filterConfig *history.AssetFilterConfig) error {
	// only need to re-initialize the filter config state(rules) if it's cached version(in  memory)
	// is older than the incoming config version based on lastModified epoch timestamp
	if filterConfig.LastModified > filter.lastModified {
		logger.Infof("New Asset Filter config detected, reloading new config %v ", *filterConfig)
		filter.enabled = filterConfig.Enabled
		filter.canonicalAssetsLookup = listToMap(filterConfig.Whitelist)
		filter.lastModified = filterConfig.LastModified
	}

	return nil
}

func (f *assetFilter) FilterTransaction(ctx context.Context, transaction ingest.LedgerTransaction) (bool, error) {
	// filtering is disabled if the whitelist is empty for now as that is the only filter rule
	if len(f.canonicalAssetsLookup) < 1 || !f.enabled {
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
		}

		if allowed {
			return true, nil
		}
	}

	logger.Debugf("No match, dropped tx with seq %v ", transaction.Envelope.SeqNum())
	return false, nil
}

func (f *assetFilter) assetMatchedFilter(asset *xdr.Asset) bool {
	var matched = false
	if _, found := f.canonicalAssetsLookup[asset.StringCanonical()]; found {
		matched = true
	}
	return matched
}

func listToMap(list []string) map[string]struct{} {
	set := make(map[string]struct{})
	for i := 0; i < len(list); i++ {
		set[list[i]] = struct{}{}
	}
	return set
}
