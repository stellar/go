package txnbuild

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// Asset represents assets on the Stellar network.
type Asset struct {
	Code   string
	Issuer string
}

// NewNativeAsset is syntactic sugar that makes instantiating an XLM *Asset more convenient.
func NewNativeAsset() *Asset {
	a := Asset{}
	return &a
}

// NewAsset is syntactic sugar that makes instantiating *Asset more convenient.
func NewAsset(code, issuer string) *Asset {
	a := Asset{
		Code:   code,
		Issuer: issuer,
	}
	return &a
}

// IsNative for Asset returns true if this is an XLM asset.
func (a *Asset) IsNative() bool {
	// Native (Lumens) has no code or issuer
	return a.Code == "" && a.Issuer == ""
}

// ToXDR for Asset produces a corresponding XDR asset.
func (a *Asset) ToXDR() (xdr.Asset, error) {
	xdrAsset := xdr.Asset{}
	var err error
	if a.IsNative() {
		err = xdrAsset.SetNative()
		if err != nil {
			return xdr.Asset{}, err
		}
		return xdrAsset, nil
	}

	var issuer xdr.AccountId
	err = issuer.SetAddress(a.Issuer)
	if err != nil {
		return xdr.Asset{}, err
	}

	err = xdrAsset.SetCredit(a.Code, issuer)
	if err != nil {
		return xdr.Asset{}, errors.Wrap(err, "Asset code length must be between 1 and 12 characters")
	}

	return xdrAsset, nil
}

// ToXDRAllowTrustOpAsset for Asset produces a corresponding XDR "allow trust" asset, used by the
// XDR allow trust operation.
func (a *Asset) ToXDRAllowTrustOpAsset() (xdr.AllowTrustOpAsset, error) {
	length := len(a.Code)

	// TODO: This code could be moved to XDR library as is done with xdr.Asset()
	switch {
	case length >= 1 && length <= 4:
		var code [4]byte
		byteArray := []byte(a.Code)
		copy(code[:], byteArray[0:length])
		return xdr.NewAllowTrustOpAsset(xdr.AssetTypeAssetTypeCreditAlphanum4, code)
	case length >= 5 && length <= 12:
		var code [12]byte
		byteArray := []byte(a.Code)
		copy(code[:], byteArray[0:length])
		return xdr.NewAllowTrustOpAsset(xdr.AssetTypeAssetTypeCreditAlphanum12, code)
	default:
		return xdr.AllowTrustOpAsset{}, errors.New("Asset code length is invalid")
	}
}
