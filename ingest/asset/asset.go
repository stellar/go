package asset

import "github.com/stellar/go/xdr"

// NewNativeAsset creates an Asset representing the native token (XLM).
func NewNativeAsset() *Asset {
	return &Asset{AssetType: &Asset_Native{Native: true}}
}

// NewIssuedAssetFromCodeAndIssuer creates an Asset with an asset code and issuer.
func NewIssuedAssetFromCodeAndIssuer(assetCode, issuer string) *Asset {
	return &Asset{
		AssetType: &Asset_IssuedAsset{
			IssuedAsset: &IssuedAsset{
				AssetCode: assetCode,
				Issuer:    issuer,
			},
		},
	}
}

func NewIssuedAsset(asset xdr.Asset) *Asset {
	return &Asset{
		AssetType: &Asset_IssuedAsset{
			IssuedAsset: &IssuedAsset{
				AssetCode: asset.GetCode(),
				Issuer:    asset.GetIssuer(),
			},
		},
	}
}
