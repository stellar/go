package resourceadapter

import (
	"context"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/httpx"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/protocols/resource/base"
	. "github.com/stellar/go/protocols/resource/effects"
)


var EffectTypeNames = map[history.EffectType]string{
	history.EffectAccountCreated:                     "account_created",
	history.EffectAccountRemoved:                     "account_removed",
	history.EffectAccountCredited:                    "account_credited",
	history.EffectAccountDebited:                     "account_debited",
	history.EffectAccountThresholdsUpdated:           "account_thresholds_updated",
	history.EffectAccountHomeDomainUpdated:           "account_home_domain_updated",
	history.EffectAccountFlagsUpdated:                "account_flags_updated",
	history.EffectAccountInflationDestinationUpdated: "account_inflation_destination_updated",
	history.EffectSignerCreated:                      "signer_created",
	history.EffectSignerRemoved:                      "signer_removed",
	history.EffectSignerUpdated:                      "signer_updated",
	history.EffectTrustlineCreated:                   "trustline_created",
	history.EffectTrustlineRemoved:                   "trustline_removed",
	history.EffectTrustlineUpdated:                   "trustline_updated",
	history.EffectTrustlineAuthorized:                "trustline_authorized",
	history.EffectTrustlineDeauthorized:              "trustline_deauthorized",
	history.EffectOfferCreated:                       "offer_created",
	history.EffectOfferRemoved:                       "offer_removed",
	history.EffectOfferUpdated:                       "offer_updated",
	history.EffectTrade:                              "trade",
	history.EffectDataCreated:                        "data_created",
	history.EffectDataRemoved:                        "data_removed",
	history.EffectDataUpdated:                        "data_updated",
}

// NewEffect creates a new effect resource from the provided database representation
// of the effect.
func NewEffect(
	ctx context.Context,
	row history.Effect,
	ledger history.Ledger,
) (result hal.Pageable, err error) {

	basev := BaseEffect{}
	PopulateBaseEffect(ctx, &basev, row, ledger)

	switch row.Type {
	case history.EffectAccountCreated:
		e := AccountCreated{BaseEffect: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectAccountCredited:
		e := AccountCredited{BaseEffect: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectAccountDebited:
		e := AccountDebited{BaseEffect: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectAccountThresholdsUpdated:
		e := AccountThresholdsUpdated{BaseEffect: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectAccountHomeDomainUpdated:
		e := AccountHomeDomainUpdated{BaseEffect: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectAccountFlagsUpdated:
		e := AccountFlagsUpdated{BaseEffect: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectSignerCreated:
		e := SignerCreated{BaseEffect: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectSignerUpdated:
		e := SignerUpdated{BaseEffect: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectSignerRemoved:
		e := SignerRemoved{BaseEffect: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrustlineCreated:
		e := TrustlineCreated{BaseEffect: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrustlineUpdated:
		e := TrustlineUpdated{BaseEffect: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrustlineRemoved:
		e := TrustlineRemoved{BaseEffect: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrustlineAuthorized:
		e := TrustlineAuthorized{BaseEffect: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrustlineDeauthorized:
		e := TrustlineDeauthorized{BaseEffect: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrade:
		e := Trade{BaseEffect: basev}
		err = row.UnmarshalDetails(&e)
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
func PopulateBaseEffect(ctx context.Context, this *BaseEffect, row history.Effect, ledger history.Ledger) {
	this.ID = row.ID()
	this.PT = row.PagingToken()
	this.Account = row.Account
	populateEffectType(this, row)
	this.LedgerCloseTime = ledger.ClosedAt

	lb := hal.LinkBuilder{httpx.BaseURL(ctx)}
	this.Links.Operation = lb.Linkf("/operations/%d", row.HistoryOperationID)
	this.Links.Succeeds = lb.Linkf("/effects?order=desc&cursor=%s", this.PT)
	this.Links.Precedes = lb.Linkf("/effects?order=asc&cursor=%s", this.PT)
}

func populateEffectType(this *BaseEffect, row history.Effect) {
	var ok bool
	this.TypeI = int32(row.Type)
	this.Type, ok = EffectTypeNames[row.Type]

	if !ok {
		this.Type = "unknown"
	}
}
