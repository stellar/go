package operations

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/xdr"
)

// OperationTypeNames maps from operation type to the string used to represent that type
// in horizon's JSON responses
var TypeNames = map[xdr.OperationType]string{
	xdr.OperationTypeCreateAccount:      "create_account",
	xdr.OperationTypePayment:            "payment",
	xdr.OperationTypePathPayment:        "path_payment",
	xdr.OperationTypeManageOffer:        "manage_offer",
	xdr.OperationTypeCreatePassiveOffer: "create_passive_offer",
	xdr.OperationTypeSetOptions:         "set_options",
	xdr.OperationTypeChangeTrust:        "change_trust",
	xdr.OperationTypeAllowTrust:         "allow_trust",
	xdr.OperationTypeAccountMerge:       "account_merge",
	xdr.OperationTypeInflation:          "inflation",
	xdr.OperationTypeManageData:         "manage_data",
	xdr.OperationTypeBumpSequence:       "bump_sequence",
}

// Base represents the common attributes of an operation resource
type Base struct {
	Links struct {
		Self        hal.Link `json:"self"`
		Transaction hal.Link `json:"transaction"`
		Effects     hal.Link `json:"effects"`
		Succeeds    hal.Link `json:"succeeds"`
		Precedes    hal.Link `json:"precedes"`
	} `json:"_links"`

	ID string `json:"id"`
	PT string `json:"paging_token"`
	// TransactionSuccessful defines if this operation is part of
	// successful transaction.
	TransactionSuccessful bool      `json:"transaction_successful"`
	SourceAccount         string    `json:"source_account"`
	Type                  string    `json:"type"`
	TypeI                 int32     `json:"type_i"`
	LedgerCloseTime       time.Time `json:"created_at"`
	TransactionHash       string    `json:"transaction_hash"`
}

// PagingToken implements hal.Pageable
func (this Base) PagingToken() string {
	return this.PT
}

// BumpSequence is the json resource representing a single operation whose type is
// BumpSequence.
type BumpSequence struct {
	Base
	BumpTo string `json:"bump_to"`
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

// Operation interface contains methods implemented by the operation types
type Operation interface {
	PagingToken() string
	GetType() string
	GetOperation() Base
	GetCreateAccount(object interface{}) (CreateAccount, error)
	GetPayment(object interface{}) (Payment, error)
	GetManageOffer(object interface{}) (ManageOffer, error)
	GetChangeTrust(object interface{}) (ChangeTrust, error)
}

// GetType returns the type of operation
func (this Base) GetType() string {
	return this.Type
}

// GetOperation returns details that is generic for all operation type. This is
// represented as the Base operation struct
func (this Base) GetOperation() Base {
	return this
}

// GetCreateAccount returns a json resource for create account operation type
func (this Base) GetCreateAccount(object interface{}) (CreateAccount, error) {
	c, ok := object.(CreateAccount)
	if !ok {
		return CreateAccount{}, errors.New("operation is not of type create_account")
	}
	return c, nil
}

// GetPayment returns a json resource for payment operation type
func (this Base) GetPayment(object interface{}) (Payment, error) {
	c, ok := object.(Payment)
	if !ok {
		return Payment{}, errors.New("operation is not of type payment")
	}
	return c, nil
}

// GetChangeTrust returns a json resource for change trust operation type
func (this Base) GetChangeTrust(object interface{}) (ChangeTrust, error) {
	c, ok := object.(ChangeTrust)
	if !ok {
		return ChangeTrust{}, errors.New("operation is not of type change_trust")
	}
	return c, nil
}

// GetManageOffer returns a json resource for manage offer operation type
func (this Base) GetManageOffer(object interface{}) (ManageOffer, error) {
	c, ok := object.(ManageOffer)
	if !ok {
		return ManageOffer{}, errors.New("operation is not of type manage_offer")
	}
	return c, nil
}

// OperationsPage is the json resource representing a page of operations.
// OperationsPage.Record can contain various operation types.
type OperationsPage struct {
	Links    hal.Links `json:"_links"`
	Embedded struct {
		Records []Operation
	} `json:"_embedded"`
}

func (ops *OperationsPage) UnmarshalJSON(data []byte) error {
	var opsPage struct {
		Links    hal.Links `json:"_links"`
		Embedded struct {
			Records []interface{}
		} `json:"_embedded"`
	}

	if err := json.Unmarshal(data, &opsPage); err != nil {
		return err
	}

	// to do: add more operation types
	for _, j := range opsPage.Embedded.Records {
		var b Base
		dataString, err := json.Marshal(j)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(dataString, &b); err != nil {
			return err
		}

		switch b.Type {
		case TypeNames[xdr.OperationTypeCreateAccount]:
			var op CreateAccount
			if err := json.Unmarshal(dataString, &op); err != nil {
				return err
			}
			ops.Embedded.Records = append(ops.Embedded.Records, op)
		case TypeNames[xdr.OperationTypePayment]:
			var op Payment
			if err := json.Unmarshal(dataString, &op); err != nil {
				return err
			}
			ops.Embedded.Records = append(ops.Embedded.Records, op)
		case TypeNames[xdr.OperationTypeManageOffer]:
			var op ManageOffer
			if err := json.Unmarshal(dataString, &op); err != nil {
				return err
			}
			ops.Embedded.Records = append(ops.Embedded.Records, op)
		default:
		}

	}

	return nil
}
