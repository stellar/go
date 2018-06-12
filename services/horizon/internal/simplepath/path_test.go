package simplepath

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func TestCachedCost(t *testing.T) {
	tt := test.Start(t).Scenario("paths")
	defer tt.Finish()

	eur := makeAsset(
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		"GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)

	p := pathNode{
		Asset: eur,
		Tail:  nil,
		Q:     &core.Q{Session: tt.CoreSession()},
	}
	p.init()

	// when we first run there should be no value
	if !tt.Assert.Nil(p.CachedCost(xdr.Int64(100))) {
		return
	}

	// run the cost function
	cost, err := p.Cost(xdr.Int64(100))
	if !tt.Assert.Nil(err) {
		return
	}
	tt.Assert.Equal(xdr.Int64(100), cost)

	// now the same CachedCost call should yield a result
	cachedCost := p.CachedCost(xdr.Int64(100))
	if tt.Assert.NotNil(cachedCost) {
		tt.Assert.Equal(xdr.Int64(100), *cachedCost)
	}
}
