package asset

import (
	"fmt"
)

// String returns a human-readable representation of the Asset.
func (a *Asset) Strings() string {
	switch assetType := a.AssetType.(type) {
	case *Asset_Native:
		return "Native Asset (XLM)"
	case *Asset_IssuedAsset:
		return fmt.Sprintf("Issued Asset - Code: %s, Issuer: %s", assetType.IssuedAsset.AssetCode, assetType.IssuedAsset.Issuer)
	default:
		return "Unknown Asset Type"
	}
}
