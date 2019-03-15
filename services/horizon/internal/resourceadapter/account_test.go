package resourceadapter

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestSplitTrustlines(t *testing.T) {
	trustlines := make([]core.Trustline, 2)
	trustlines[0] = core.Trustline{
		Accountid: "testID",
		Assettype: xdr.AssetTypeAssetTypeCreditAlphanum12,
		Issuer:    "",
		Assetcode: "TEST_ASSET_1",
		Tlimit:    100,
		Balance:   10,
		Flags:     1,
	}
	trustlines[1] = core.Trustline{
		Accountid: "testID",
		Assettype: xdr.AssetTypeAssetTypeCreditAlphanum12,
		Issuer:    "",
		Assetcode: "TEST_ASSET_2",
		Tlimit:    100,
		Balance:   10,
		Flags:     2,
	}

	auth, unauth := splitTrustlines(trustlines)
	assert.Equal(t, len(auth), 1)
	assert.Equal(t, len(unauth), 1)
	assert.Equal(t, auth[0].Assetcode, "TEST_ASSET_1")
	assert.Equal(t, unauth[0].Assetcode, "TEST_ASSET_2")
}
