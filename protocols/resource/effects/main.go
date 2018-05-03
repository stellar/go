package effects

import (
	"time"
	"github.com/stellar/go/protocols/resource/base"
	"github.com/stellar/go/support/render/hal"
)


// BaseEffect provides the common structure for any effect resource effect.
type BaseEffect struct {
	Links struct {
		Operation hal.Link `json:"operation"`
		Succeeds  hal.Link `json:"succeeds"`
		Precedes  hal.Link `json:"precedes"`
	} `json:"_links"`

	ID              string    `json:"id"`
	PT              string    `json:"paging_token"`
	Account         string    `json:"account"`
	Type            string    `json:"type"`
	TypeI           int32     `json:"type_i"`
	LedgerCloseTime time.Time `json:"created_at"`
}

// PagingToken implements `hal.Pageable`
func (this BaseEffect) PagingToken() string {
	return this.PT
}

type AccountCreated struct {
	BaseEffect
	StartingBalance string `json:"starting_balance"`
}

type AccountCredited struct {
	BaseEffect
	base.Asset
	Amount string `json:"amount"`
}

type AccountDebited struct {
	BaseEffect
	base.Asset
	Amount string `json:"amount"`
}

type AccountThresholdsUpdated struct {
	BaseEffect
	LowThreshold  int32 `json:"low_threshold"`
	MedThreshold  int32 `json:"med_threshold"`
	HighThreshold int32 `json:"high_threshold"`
}

type AccountHomeDomainUpdated struct {
	BaseEffect
	HomeDomain string `json:"home_domain"`
}

type AccountFlagsUpdated struct {
	BaseEffect
	AuthRequired  *bool `json:"auth_required_flag,omitempty"`
	AuthRevokable *bool `json:"auth_revokable_flag,omitempty"`
}

type SignerCreated struct {
	BaseEffect
	Weight    int32  `json:"weight"`
	PublicKey string `json:"public_key"`
	Key       string `json:"key"`
}

type SignerRemoved struct {
	BaseEffect
	Weight    int32  `json:"weight"`
	PublicKey string `json:"public_key"`
	Key       string `json:"key"`
}

type SignerUpdated struct {
	BaseEffect
	Weight    int32  `json:"weight"`
	PublicKey string `json:"public_key"`
	Key       string `json:"key"`
}

type TrustlineCreated struct {
	BaseEffect
	base.Asset
	Limit string `json:"limit"`
}

type TrustlineRemoved struct {
	BaseEffect
	base.Asset
	Limit string `json:"limit"`
}

type TrustlineUpdated struct {
	BaseEffect
	base.Asset
	Limit string `json:"limit"`
}

type TrustlineAuthorized struct {
	BaseEffect
	Trustor   string `json:"trustor"`
	AssetType string `json:"asset_type"`
	AssetCode string `json:"asset_code,omitempty"`
}

type TrustlineDeauthorized struct {
	BaseEffect
	Trustor   string `json:"trustor"`
	AssetType string `json:"asset_type"`
	AssetCode string `json:"asset_code,omitempty"`
}

type Trade struct {
	BaseEffect
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

// interface implementations
var _ base.Rehydratable = &SignerCreated{}
var _ base.Rehydratable = &SignerRemoved{}
var _ base.Rehydratable = &SignerUpdated{}

