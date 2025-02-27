package asset

import "github.com/stellar/go/xdr"

// NewNativeAsset creates an Asset representing the native token (XLM).
func NewNativeAsset() *Asset {
	return &Asset{AssetType: &Asset_Native{Native: true}}
}

func NewProtoAsset(asset xdr.Asset) *Asset {
	if asset.IsNative() {
		return NewNativeAsset()
	}
	return &Asset{
		AssetType: &Asset_IssuedAsset{
			IssuedAsset: &IssuedAsset{
				AssetCode: asset.GetCode(),
				Issuer:    asset.GetIssuer(),
			},
		},
	}
}
