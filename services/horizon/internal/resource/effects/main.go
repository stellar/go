package effects

import (
	"golang.org/x/net/context"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/render/hal"
	"github.com/stellar/go/services/horizon/internal/resource/base"
	"github.com/stellar/go/support/errors"
)

var TypeNames = map[history.EffectType]string{
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

// New creates a new effect resource from the provided database representation
// of the effect.
func New(ctx context.Context, row history.Effect) (result hal.Pageable, err error) {
	basev := Base{}
	basev.Populate(ctx, row)

	switch row.Type {
	case history.EffectAccountCreated:
		e := AccountCreated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectAccountCredited:
		e := AccountCredited{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectAccountDebited:
		e := AccountDebited{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectAccountThresholdsUpdated:
		e := AccountThresholdsUpdated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectAccountHomeDomainUpdated:
		e := AccountHomeDomainUpdated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectAccountFlagsUpdated:
		e := AccountFlagsUpdated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectAccountInflationDestinationUpdated:
		e := AccountInflationDestinationUpdated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectSignerCreated:
		e := SignerCreated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectSignerUpdated:
		e := SignerUpdated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectSignerRemoved:
		e := SignerRemoved{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrustlineCreated:
		e := TrustlineCreated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrustlineUpdated:
		e := TrustlineUpdated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrustlineRemoved:
		e := TrustlineRemoved{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrustlineAuthorized:
		e := TrustlineAuthorized{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrustlineDeauthorized:
		e := TrustlineDeauthorized{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectTrade:
		e := Trade{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectDataCreated:
		e := DataCreated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectDataRemoved:
		e := DataRemoved{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	case history.EffectDataUpdated:
		e := DataUpdated{Base: basev}
		err = row.UnmarshalDetails(&e)
		result = e
	default:
		result = basev
	}
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshaling details into effect")
	}

	if rh, ok := result.(base.Rehydratable); ok {
		if err := rh.Rehydrate(); err != nil {
			return nil, errors.Wrap(err, "error rehydrating effect")
		}
	}

	return result, nil
}

// Base provides the common structure for any effect resource effect.
type Base struct {
	Links struct {
		Operation hal.Link `json:"operation"`
		Succeeds  hal.Link `json:"succeeds"`
		Precedes  hal.Link `json:"precedes"`
	} `json:"_links"`

	ID      string `json:"id"`
	PT      string `json:"paging_token"`
	Account string `json:"account"`
	Type    string `json:"type"`
	TypeI   int32  `json:"type_i"`
}

type AccountCreated struct {
	Base
	StartingBalance string `json:"starting_balance"`
}

type AccountCredited struct {
	Base
	base.Asset
	Amount string `json:"amount"`
}

type AccountDebited struct {
	Base
	base.Asset
	Amount string `json:"amount"`
}

type AccountThresholdsUpdated struct {
	Base
	LowThreshold  int32 `json:"low_threshold"`
	MedThreshold  int32 `json:"med_threshold"`
	HighThreshold int32 `json:"high_threshold"`
}

type AccountHomeDomainUpdated struct {
	Base
	HomeDomain string `json:"home_domain"`
}

type AccountFlagsUpdated struct {
	Base
	AuthRequired  *bool `json:"auth_required_flag,omitempty"`
	AuthRevokable *bool `json:"auth_revokable_flag,omitempty"`
	AuthImmutable *bool `json:"auth_immutable_flag,omitempty"`
}

type AccountInflationDestinationUpdated struct {
	Base
	InflationDestination string `json:"inflation_destination"`
}

type SignerCreated struct {
	Base
	Weight    int32  `json:"weight"`
	PublicKey string `json:"public_key"`
	Key       string `json:"key"`
}

type SignerRemoved struct {
	Base
	PublicKey string `json:"public_key"`
	Key       string `json:"key"`
}

type SignerUpdated struct {
	Base
	Weight    int32  `json:"weight"`
	PublicKey string `json:"public_key"`
	Key       string `json:"key"`
}

type TrustlineCreated struct {
	Base
	base.Asset
	Limit string `json:"limit"`
}

type TrustlineRemoved struct {
	Base
	base.Asset
	Limit string `json:"limit"`
}

type TrustlineUpdated struct {
	Base
	base.Asset
	Limit string `json:"limit"`
}

type TrustlineAuthorized struct {
	Base
	base.Asset
	Trustor string `json:"trustor"`
}

type TrustlineDeauthorized struct {
	Base
	base.Asset
	Trustor string `json:"trustor"`
}

type Trade struct {
	Base
	Seller            string `json:"seller"`
	OfferID           int64  `json:"offer_id"`
	SoldAmount        string `json:"sold_amount"`
	SoldAssetType     string `json:"sold_asset_type"`
	SoldAssetCode     string `json:"sold_asset_code,omitempty"`
	SoldAssetIssuer   string `json:"sold_asset_issuer,omitempty"`
	BoughtAmount      string `json:"bought_amount"`
	BoughtAssetType   string `json:"bought_asset_type"`
	BoughtAssetCode   string `json:"bought_asset_code,omitempty"`
	BoughtAssetIssuer string `json:"bought_asset_issuer,omitempty"`
}

type DataCreated struct {
	Base
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DataRemoved struct {
	Base
	Name string `json:"name"`
}

type DataUpdated struct {
	Base
	Name  string `json:"name"`
	Value string `json:"value"`
}

// interface implementations
var _ base.Rehydratable = &SignerCreated{}
var _ base.Rehydratable = &SignerRemoved{}
var _ base.Rehydratable = &SignerUpdated{}
