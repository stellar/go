package asset

import (
	"github.com/stellar/go/xdr"
	"strings"
)

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
				// Need to trim the extra null characters from showing in the code when saving to assetCode
				AssetCode: strings.TrimRight(asset.GetCode(), "\x00"),
				Issuer:    asset.GetIssuer(),
			},
		},
	}
}
