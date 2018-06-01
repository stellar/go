package operations

import (
	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/support/render/hal"
	"time"
)

// Base represents the common attributes of an operation resource
type Base struct {
	Links struct {
		Self        hal.Link `json:"self"`
		Transaction hal.Link `json:"transaction"`
		Effects     hal.Link `json:"effects"`
		Succeeds    hal.Link `json:"succeeds"`
		Precedes    hal.Link `json:"precedes"`
	} `json:"_links"`

	ID              string    `json:"id"`
	PT              string    `json:"paging_token"`
	SourceAccount   string    `json:"source_account"`
	Type            string    `json:"type"`
	TypeI           int32     `json:"type_i"`
	LedgerCloseTime time.Time `json:"created_at"`
	TransactionHash string    `json:"transaction_hash"`
}

// PagingToken implements hal.Pageable
func (this Base) PagingToken() string {
	return this.PT
}


// CreateAccount is the json resource representing a single operation whose type
// is CreateAccount.
type CreateAccount struct {
	Base
	StartingBalance string `json:"starting_balance"`
	Funder          string `json:"funder"`
	Account         string `json:"account"`
}

// Payment is the json resource representing a single operation whose type is
// Payment.
type Payment struct {
	Base
	base.Asset
	From   string `json:"from"`
	To     string `json:"to"`
	Amount string `json:"amount"`
}

// PathPayment is the json resource representing a single operation whose type
// is PathPayment.
type PathPayment struct {
	Payment
	Path              []base.Asset `json:"path"`
	SourceAmount      string       `json:"source_amount"`
	SourceMax         string       `json:"source_max"`
	SourceAssetType   string       `json:"source_asset_type"`
	SourceAssetCode   string       `json:"source_asset_code,omitempty"`
	SourceAssetIssuer string       `json:"source_asset_issuer,omitempty"`
}

// ManageData represents a ManageData operation as it is serialized into json
// for the horizon API.
type ManageData struct {
	Base
	Name  string `json:"name"`
	Value string `json:"value"`
}

// CreatePassiveOffer is the json resource representing a single operation whose
// type is CreatePassiveOffer.
type CreatePassiveOffer struct {
	Base
	Amount             string     `json:"amount"`
	Price              string     `json:"price"`
	PriceR             base.Price `json:"price_r"`
	BuyingAssetType    string     `json:"buying_asset_type"`
	BuyingAssetCode    string     `json:"buying_asset_code,omitempty"`
	BuyingAssetIssuer  string     `json:"buying_asset_issuer,omitempty"`
	SellingAssetType   string     `json:"selling_asset_type"`
	SellingAssetCode   string     `json:"selling_asset_code,omitempty"`
	SellingAssetIssuer string     `json:"selling_asset_issuer,omitempty"`
}

// ManageOffer is the json resource representing a single operation whose type
// is ManageOffer.
type ManageOffer struct {
	CreatePassiveOffer
	OfferID int64 `json:"offer_id"`
}

// SetOptions is the json resource representing a single operation whose type is
// SetOptions.
type SetOptions struct {
	Base
	HomeDomain    string `json:"home_domain,omitempty"`
	InflationDest string `json:"inflation_dest,omitempty"`

	MasterKeyWeight *int   `json:"master_key_weight,omitempty"`
	SignerKey       string `json:"signer_key,omitempty"`
	SignerWeight    *int   `json:"signer_weight,omitempty"`

	SetFlags    []int    `json:"set_flags,omitempty"`
	SetFlagsS   []string `json:"set_flags_s,omitempty"`
	ClearFlags  []int    `json:"clear_flags,omitempty"`
	ClearFlagsS []string `json:"clear_flags_s,omitempty"`

	LowThreshold  *int `json:"low_threshold,omitempty"`
	MedThreshold  *int `json:"med_threshold,omitempty"`
	HighThreshold *int `json:"high_threshold,omitempty"`
}

// ChangeTrust is the json resource representing a single operation whose type
// is ChangeTrust.
type ChangeTrust struct {
	Base
	base.Asset
	Limit   string `json:"limit"`
	Trustee string `json:"trustee"`
	Trustor string `json:"trustor"`
}

// AllowTrust is the json resource representing a single operation whose type is
// AllowTrust.
type AllowTrust struct {
	Base
	base.Asset
	Trustee   string `json:"trustee"`
	Trustor   string `json:"trustor"`
	Authorize bool   `json:"authorize"`
}

// AccountMerge is the json resource representing a single operation whose type
// is AccountMerge.
type AccountMerge struct {
	Base
	Account string `json:"account"`
	Into    string `json:"into"`
}

// Inflation is the json resource representing a single operation whose type is
// Inflation.
type Inflation struct {
	Base
}