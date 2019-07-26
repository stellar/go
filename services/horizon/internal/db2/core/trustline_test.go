package core

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func assetToSymbol(asset xdr.Asset) string {
	if alpha12, ok := asset.GetAlphaNum12(); ok {
		return string(alpha12.AssetCode[:len(alpha12.AssetCode)])
	} else if alpha4, ok := asset.GetAlphaNum4(); ok {
		return string(alpha4.AssetCode[:len(alpha4.AssetCode)])
	} else {
		return "XLM"
	}
}

func assetsToSymbols(assets []xdr.Asset) []string {
	symbols := make([]string, len(assets))
	for i, asset := range assets {
		symbols[i] = assetToSymbol(asset)
	}
	return symbols
}

func TestAssetsForAddress(t *testing.T) {
	tt := test.Start(t).Scenario("order_books")
	defer tt.Finish()
	q := &Q{tt.CoreSession()}

	assets, balances, err := q.AssetsForAddress(
		"GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON",
	)
	tt.Assert.NoError(err)
	assetSymbols := assetsToSymbols(assets)

	assetsToBalance := map[string]xdr.Int64{}
	for i, symbol := range assetSymbols {
		assetsToBalance[symbol] = balances[i]
	}

	expected := map[string]xdr.Int64{"BTC\x00": 60000000000, "USD\x00": 50000000000, "XLM": 99999999200}

	tt.Assert.Equal(expected, assetsToBalance)
}
