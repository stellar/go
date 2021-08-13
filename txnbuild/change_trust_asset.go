package txnbuild

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// ChangeTrustAsset represents a Stellar change trust asset.
type ChangeTrustAsset interface {
	BasicAsset
	ToChangeTrustXDR() (xdr.ChangeTrustAsset, error)
	ToTrustLineXDR() (xdr.TrustLineAsset, error)
}

// LiquidityPoolShareChangeTrustAsset represents non-XLM assets on the Stellar network.
type LiquidityPoolShareChangeTrustAsset struct {
	LiquidityPoolParameters LiquidityPoolParameters
}

// GetType for LiquidityPoolShareChangeTrustAsset returns the enum type of the asset, based on its code length.
func (lpsa LiquidityPoolShareChangeTrustAsset) GetType() (AssetType, error) {
	return AssetTypePoolShare, nil
}

// IsNative for LiquidityPoolShareChangeTrustAsset returns false (this is not an XLM asset).
func (lpsa LiquidityPoolShareChangeTrustAsset) IsNative() bool { return false }

// GetCode for LiquidityPoolShareChangeTrustAsset returns blank string
func (lpsa LiquidityPoolShareChangeTrustAsset) GetCode() string { return "" }

// GetIssuer for LiquidityPoolShareChangeTrustAsset returns blank string
func (lpsa LiquidityPoolShareChangeTrustAsset) GetIssuer() string { return "" }

// GetLiquidityPoolID for LiquidityPoolShareChangeTrustAsset returns the pool id computed from the parameters.
func (lpsa LiquidityPoolShareChangeTrustAsset) GetLiquidityPoolID() (LiquidityPoolId, bool) {
	poolId, err := NewLiquidityPoolId(lpsa.LiquidityPoolParameters.AssetA, lpsa.LiquidityPoolParameters.AssetB)
	return poolId, err == nil
}

// GetLiquidityPoolParameters for LiquidityPoolShareChangeTrustAsset returns the pool parameters.
func (lpsa LiquidityPoolShareChangeTrustAsset) GetLiquidityPoolParameters() (LiquidityPoolParameters, bool) {
	return lpsa.LiquidityPoolParameters, true
}

// ToXDR for LiquidityPoolShareChangeTrustAsset produces a corresponding XDR change trust asset.
func (lpsa LiquidityPoolShareChangeTrustAsset) ToChangeTrustXDR() (xdr.ChangeTrustAsset, error) {
	xdrPoolParams, err := lpsa.LiquidityPoolParameters.ToXDR()
	if err != nil {
		return xdr.ChangeTrustAsset{}, errors.Wrap(err, "failed to build XDR liquidity pool parameters")
	}
	return xdr.ChangeTrustAsset{LiquidityPool: &xdrPoolParams}, nil
}

// ToXDR for LiquidityPoolShareChangeTrustAsset produces a corresponding XDR change trust asset.
func (lpsa LiquidityPoolShareChangeTrustAsset) ToTrustLineXDR() (xdr.TrustLineAsset, error) {
	poolId, ok := lpsa.GetLiquidityPoolID()
	if !ok {
		return xdr.TrustLineAsset{}, errors.New("failed to build XDR liquidity pool id")
	}
	xdrPoolId, err := poolId.ToXDR()
	if err != nil {
		return xdr.TrustLineAsset{}, errors.Wrap(err, "failed to build XDR liquidity pool id")
	}
	return xdr.TrustLineAsset{LiquidityPoolId: &xdrPoolId}, nil
}

func assetFromChangeTrustAssetXDR(xAsset xdr.ChangeTrustAsset) (ChangeTrustAsset, error) {
	if xAsset.Type == xdr.AssetTypeAssetTypePoolShare {
		params, err := liquidityPoolParametersFromXDR(*xAsset.LiquidityPool)
		if err != nil {
			return nil, errors.Wrap(err, "invalid XDR liquidity pool parameters")
		}
		return LiquidityPoolShareChangeTrustAsset{LiquidityPoolParameters: params}, nil
	}
	return assetFromXDR(xAsset.ToAsset())
}
