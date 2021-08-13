package txnbuild

import (
	"bytes"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// AssetType represents the type of a Stellar asset.
type AssetType xdr.AssetType

// AssetTypeNative, AssetTypeCreditAlphanum4, AssetTypeCreditAlphanum12 enumerate the different
// types of asset on the Stellar network.
const (
	AssetTypeNative           AssetType = AssetType(xdr.AssetTypeAssetTypeNative)
	AssetTypeCreditAlphanum4  AssetType = AssetType(xdr.AssetTypeAssetTypeCreditAlphanum4)
	AssetTypeCreditAlphanum12 AssetType = AssetType(xdr.AssetTypeAssetTypeCreditAlphanum12)
	AssetTypePoolShare        AssetType = AssetType(xdr.AssetTypeAssetTypePoolShare)
)

// Breaks out some stuff common to all assets
type BasicAsset interface {
	GetType() (AssetType, error)
	IsNative() bool
	GetCode() string
	GetIssuer() string
	GetLiquidityPoolID() (LiquidityPoolId, bool)
	GetLiquidityPoolParameters() (LiquidityPoolParameters, bool)
}

// Asset represents a Stellar asset.
type Asset interface {
	BasicAsset
	ToXDR() (xdr.Asset, error)
	ToChangeTrustXDR() (xdr.ChangeTrustAsset, error)
	ToTrustLineXDR() (xdr.TrustLineAsset, error)
}

// NativeAsset represents the native XLM asset.
type NativeAsset struct{}

// GetType for NativeAsset returns the enum type of the asset.
func (na NativeAsset) GetType() (AssetType, error) {
	return AssetTypeNative, nil
}

// IsNative for NativeAsset returns true (this is an XLM asset).
func (na NativeAsset) IsNative() bool { return true }

// GetCode for NativeAsset returns an empty string (XLM doesn't have a code).
func (na NativeAsset) GetCode() string { return "" }

// GetIssuer for NativeAsset returns an empty string (XLM doesn't have an issuer).
func (na NativeAsset) GetIssuer() string { return "" }

// GetLiquidityPoolID for NativeAsset returns empty.
func (na NativeAsset) GetLiquidityPoolID() (LiquidityPoolId, bool) {
	return LiquidityPoolId{}, false
}

// GetLiquidityPoolParameters for NativeAsset returns empty.
func (na NativeAsset) GetLiquidityPoolParameters() (LiquidityPoolParameters, bool) {
	return LiquidityPoolParameters{}, false
}

// ToXDR for NativeAsset produces a corresponding XDR asset.
func (na NativeAsset) ToXDR() (xdr.Asset, error) {
	xdrAsset := xdr.Asset{}
	err := xdrAsset.SetNative()
	if err != nil {
		return xdr.Asset{}, err
	}
	return xdrAsset, nil
}

// ToChangeTrustXDR for NativeAsset produces a corresponding XDR asset.
func (na NativeAsset) ToChangeTrustXDR() (xdr.ChangeTrustAsset, error) {
	xdrAsset := xdr.Asset{}
	err := xdrAsset.SetNative()
	if err != nil {
		return xdr.ChangeTrustAsset{}, err
	}
	return xdrAsset.ToChangeTrustAsset(), nil
}

// ToTrustLineXDR for NativeAsset produces a corresponding XDR asset.
func (na NativeAsset) ToTrustLineXDR() (xdr.TrustLineAsset, error) {
	xdrAsset := xdr.Asset{}
	err := xdrAsset.SetNative()
	if err != nil {
		return xdr.TrustLineAsset{}, err
	}
	return xdrAsset.ToTrustLineAsset(), nil
}

// CreditAsset represents non-XLM assets on the Stellar network.
type CreditAsset struct {
	Code   string
	Issuer string
}

// GetType for CreditAsset returns the enum type of the asset, based on its code length.
func (ca CreditAsset) GetType() (AssetType, error) {
	switch {
	case len(ca.Code) >= 1 && len(ca.Code) <= 4:
		return AssetTypeCreditAlphanum4, nil
	case len(ca.Code) >= 5 && len(ca.Code) <= 12:
		return AssetTypeCreditAlphanum12, nil
	default:
		return AssetTypeCreditAlphanum4, errors.New("asset code length must be between 1 and 12 characters")
	}
}

// IsNative for CreditAsset returns false (this is not an XLM asset).
func (ca CreditAsset) IsNative() bool { return false }

// GetCode for CreditAsset returns the asset code.
func (ca CreditAsset) GetCode() string { return ca.Code }

// GetIssuer for CreditAsset returns the address of the issuing account.
func (ca CreditAsset) GetIssuer() string { return ca.Issuer }

// GetLiquidityPoolID for CreditAsset returns empty.
func (na CreditAsset) GetLiquidityPoolID() (LiquidityPoolId, bool) {
	return LiquidityPoolId{}, false
}

// GetLiquidityPoolParameters for CreditAsset returns empty.
func (na CreditAsset) GetLiquidityPoolParameters() (LiquidityPoolParameters, bool) {
	return LiquidityPoolParameters{}, false
}

// ToXDR for CreditAsset produces a corresponding XDR asset.
func (ca CreditAsset) ToXDR() (xdr.Asset, error) {
	xdrAsset := xdr.Asset{}
	var issuer xdr.AccountId

	err := issuer.SetAddress(ca.Issuer)
	if err != nil {
		return xdr.Asset{}, err
	}

	err = xdrAsset.SetCredit(ca.Code, issuer)
	if err != nil {
		return xdr.Asset{}, errors.Wrap(err, "asset code length must be between 1 and 12 characters")
	}

	return xdrAsset, nil
}

// ToChangeTrustXDR for CreditAsset produces a corresponding XDR asset.
func (ca CreditAsset) ToChangeTrustXDR() (xdr.ChangeTrustAsset, error) {
	xdrAsset, err := ca.ToXDR()
	return xdrAsset.ToChangeTrustAsset(), err
}

// ToTrustLineXDR for CreditAsset produces a corresponding XDR asset.
func (ca CreditAsset) ToTrustLineXDR() (xdr.TrustLineAsset, error) {
	xdrAsset, err := ca.ToXDR()
	return xdrAsset.ToTrustLineAsset(), err
}

// to do: consider exposing function or adding it to asset interface
func assetFromXDR(xAsset xdr.Asset) (Asset, error) {
	switch xAsset.Type {
	case xdr.AssetTypeAssetTypeNative:
		return NativeAsset{}, nil
	case xdr.AssetTypeAssetTypeCreditAlphanum4:
		code := bytes.Trim(xAsset.AlphaNum4.AssetCode[:], "\x00")
		return CreditAsset{
			Code:   string(code[:]),
			Issuer: xAsset.AlphaNum4.Issuer.Address(),
		}, nil
	case xdr.AssetTypeAssetTypeCreditAlphanum12:
		code := bytes.Trim(xAsset.AlphaNum12.AssetCode[:], "\x00")
		return CreditAsset{
			Code:   string(code[:]),
			Issuer: xAsset.AlphaNum12.Issuer.Address(),
		}, nil
	}

	return nil, errors.New("invalid asset")
}
