package protocols

import (
	"fmt"

	"github.com/stellar/go/build"
)

// ToBaseAsset transforms Asset to github.com/stellar/go-stellar-base/build.Asset
func (a Asset) ToBaseAsset() build.Asset {
	if a.Type == "native" {
		return build.NativeAsset()
	}
	return build.CreditAsset(a.Code, a.Issuer)
}

// String returns string representation of this asset
func (a Asset) String() string {
	return fmt.Sprintf("Code: %s, Issuer: %s", a.Code, a.Issuer)
}

// Validate checks if asset params are correct.
func (a Asset) Validate() bool {
	panic("TODO")
	// if a.Code != "" && a.Issuer != "" {
	// 	// Credit
	// 	return IsValidAssetCode(a.Code) && IsValidAccountID(a.Issuer)
	// } else if a.Code == "" && a.Issuer == "" {
	// 	// Native
	// 	return true
	// } else {
	// 	return false
	// }
}
