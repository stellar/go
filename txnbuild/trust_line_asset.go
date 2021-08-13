package txnbuild

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// TrustLineAsset represents a Stellar trust line asset.
type TrustLineAsset interface {
	BasicAsset
	ToTrustLineXDR() (xdr.TrustLineAsset, error)
}

// LiquidityPoolShareTrustLineAsset represents shares in a liquidity pool on the Stellar network.
type LiquidityPoolShareTrustLineAsset struct {
	LiquidityPoolID LiquidityPoolId
}

// GetType for LiquidityPoolShareAsset returns the enum type of the asset, based on its code length.
func (lpsa LiquidityPoolShareTrustLineAsset) GetType() (AssetType, error) {
	return AssetTypePoolShare, nil
}

// IsNative for LiquidityPoolShareAsset returns false (this is not an XLM asset).
func (lpsa LiquidityPoolShareTrustLineAsset) IsNative() bool { return false }

// GetCode for LiquidityPoolShareAsset returns blank string
func (lpsa LiquidityPoolShareTrustLineAsset) GetCode() string { return "" }

// GetIssuer for LiquidityPoolShareAsset returns blank string
func (lpsa LiquidityPoolShareTrustLineAsset) GetIssuer() string { return "" }

// GetLiquidityPoolID for LiquidityPoolShareTrustLineAsset returns the pool id.
func (lpsa LiquidityPoolShareTrustLineAsset) GetLiquidityPoolID() (LiquidityPoolId, bool) {
	return lpsa.LiquidityPoolID, true
}

// GetLiquidityPoolParameters for LiquidityPoolShareTrustLineAsset returns empty.
func (lpsa LiquidityPoolShareTrustLineAsset) GetLiquidityPoolParameters() (LiquidityPoolParameters, bool) {
	return LiquidityPoolParameters{}, false
}

// ToXDR for LiquidityPoolShareAsset produces a corresponding XDR asset.
func (lpsa LiquidityPoolShareTrustLineAsset) ToXDR() (xdr.TrustLineAsset, error) {
	xdrPoolId, err := lpsa.LiquidityPoolID.ToXDR()
	if err != nil {
		return xdr.TrustLineAsset{}, errors.Wrap(err, "failed to build XDR liquidity pool id")
	}
	return xdr.TrustLineAsset{LiquidityPoolId: &xdrPoolId}, nil
}

func (lpsa LiquidityPoolShareTrustLineAsset) ToTrustLineXDR() (xdr.TrustLineAsset, error) {
	return lpsa.ToXDR()
}

func assetFromTrustLineAssetXDR(xAsset xdr.TrustLineAsset) (TrustLineAsset, error) {
	if xAsset.Type == xdr.AssetTypeAssetTypePoolShare {
		if xAsset.LiquidityPoolId == nil {
			return LiquidityPoolShareTrustLineAsset{}, errors.New("invalid XDR liquidity pool id")
		}
		poolId, err := liquidityPoolIdFromXDR(*xAsset.LiquidityPoolId)
		if err != nil {
			return LiquidityPoolShareTrustLineAsset{}, errors.Wrap(err, "invalid XDR liquidity pool id")
		}
		return LiquidityPoolShareTrustLineAsset{LiquidityPoolID: poolId}, nil
	}
	return assetFromXDR(xAsset.ToAsset())
}
