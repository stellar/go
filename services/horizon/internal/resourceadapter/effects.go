package resourceadapter

import (
	"context"

	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/protocols/horizon/effects"
	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/render/hal"
)

var EffectTypeNames = map[history.EffectType]string{
	history.EffectAccountCreated:                           "account_created",
	history.EffectAccountRemoved:                           "account_removed",
	history.EffectAccountCredited:                          "account_credited",
	history.EffectAccountDebited:                           "account_debited",
	history.EffectAccountThresholdsUpdated:                 "account_thresholds_updated",
	history.EffectAccountHomeDomainUpdated:                 "account_home_domain_updated",
	history.EffectAccountFlagsUpdated:                      "account_flags_updated",
	history.EffectAccountInflationDestinationUpdated:       "account_inflation_destination_updated",
	history.EffectSignerCreated:                            "signer_created",
	history.EffectSignerRemoved:                            "signer_removed",
	history.EffectSignerUpdated:                            "signer_updated",
	history.EffectTrustlineCreated:                         "trustline_created",
	history.EffectTrustlineRemoved:                         "trustline_removed",
	history.EffectTrustlineUpdated:                         "trustline_updated",
	history.EffectTrustlineAuthorized:                      "trustline_authorized",
	history.EffectTrustlineAuthorizedToMaintainLiabilities: "trustline_authorized_to_maintain_liabilities",
	history.EffectTrustlineDeauthorized:                    "trustline_deauthorized",
	history.EffectOfferCreated:                             "offer_created",
	history.EffectOfferRemoved:                             "offer_removed",
	history.EffectOfferUpdated:                             "offer_updated",
	history.EffectTrade:                                    "trade",
	history.EffectDataCreated:                              "data_created",
	history.EffectDataRemoved:                              "data_removed",
	history.EffectDataUpdated:                              "data_updated",
	history.EffectSequenceBumped:                           "sequence_bumped",
}

// NewEffect creates a new effect resource from the provided database representation
// of the effect.
func NewEffect(
	ctx context.Context,
	row history.Effect,
	ledger history.Ledger,
) (result hal.Pageable, err error) {

	basev := effects.Base{}
	PopulateBaseEffect(ctx, &basev, row, ledger)

	switch row.Type {
	case history.EffectAccountCreated:
		e := effects.AccountCreated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectAccountCredited:
		e := effects.AccountCredited{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectAccountDebited:
		e := effects.AccountDebited{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectAccountThresholdsUpdated:
		e := effects.AccountThresholdsUpdated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectAccountHomeDomainUpdated:
		e := effects.AccountHomeDomainUpdated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectAccountFlagsUpdated:
		e := effects.AccountFlagsUpdated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectSignerCreated:
		e := effects.SignerCreated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectSignerUpdated:
		e := effects.SignerUpdated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectSignerRemoved:
		e := effects.SignerRemoved{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrustlineCreated:
		e := effects.TrustlineCreated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrustlineUpdated:
		e := effects.TrustlineUpdated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrustlineRemoved:
		e := effects.TrustlineRemoved{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrustlineAuthorized:
		e := effects.TrustlineAuthorized{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrustlineAuthorizedToMaintainLiabilities:
		e := effects.TrustlineAuthorizedToMaintainLiabilities{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrustlineDeauthorized:
		e := effects.TrustlineDeauthorized{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrade:
		e := effects.Trade{Base: basev}
		tradeDetails := history.TradeEffectDetails{}
		err = row.UnmarshalDetails(&tradeDetails)
		if err == nil {
			e.Seller = tradeDetails.Seller
			e.OfferID = tradeDetails.OfferID
			e.SoldAmount = tradeDetails.SoldAmount
			e.SoldAssetType = tradeDetails.SoldAssetType
			e.SoldAssetCode = tradeDetails.SoldAssetCode
			e.SoldAssetIssuer = tradeDetails.SoldAssetIssuer
			e.BoughtAmount = tradeDetails.BoughtAmount
			e.BoughtAssetType = tradeDetails.BoughtAssetType
			e.BoughtAssetCode = tradeDetails.BoughtAssetCode
			e.BoughtAssetIssuer = tradeDetails.BoughtAssetIssuer
		}
		result = e
	case history.EffectSequenceBumped:
		e := effects.SequenceBumped{Base: basev}
		hsb := history.SequenceBumped{}
		err = row.UnmarshalDetails(&hsb)
		if err == nil {
			e.NewSeq = hsb.NewSeq
		}
		result = e
	default:
		result = basev
	}

	if err != nil {
		return
	}

	rh, ok := result.(base.Rehydratable)

	if ok {
		err = rh.Rehydrate()
	}

	return
}

// Populate loads this resource from `row`
func PopulateBaseEffect(ctx context.Context, this *effects.Base, row history.Effect, ledger history.Ledger) {
	this.ID = row.ID()
	this.PT = row.PagingToken()
	this.Account = row.Account
	populateEffectType(this, row)
	this.LedgerCloseTime = ledger.ClosedAt

	lb := hal.LinkBuilder{horizonContext.BaseURL(ctx)}
	this.Links.Operation = lb.Linkf("/operations/%d", row.HistoryOperationID)
	this.Links.Succeeds = lb.Linkf("/effects?order=desc&cursor=%s", this.PT)
	this.Links.Precedes = lb.Linkf("/effects?order=asc&cursor=%s", this.PT)
}

func populateEffectType(this *effects.Base, row history.Effect) {
	var ok bool
	this.TypeI = int32(row.Type)
	this.Type, ok = EffectTypeNames[row.Type]

	if !ok {
		this.Type = "unknown"
	}
}
