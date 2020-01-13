package history

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func TestCreateExpAssetIDs(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &Q{tt.HorizonSession()}

	// CreateExpAssets creates new rows
	assets := []xdr.Asset{
		nativeAsset, eurAsset,
	}
	assetMap, err := q.CreateExpAssets(assets)
	tt.Assert.NoError(err)
	tt.Assert.Len(assetMap, len(assets))

	set := map[int64]bool{}
	for _, asset := range assets {
		row := assetMap[asset.String()]

		tt.Assert.False(set[row.ID])
		set[row.ID] = true

		var assetType, assetCode, assetIssuer string
		asset.MustExtract(&assetType, &assetCode, &assetIssuer)

		tt.Assert.Equal(row.Type, assetType)
		tt.Assert.Equal(row.Code, assetCode)
		tt.Assert.Equal(row.Issuer, assetIssuer)
	}

	// CreateExpAssets handles duplicates
	assetMap, err = q.CreateExpAssets([]xdr.Asset{
		nativeAsset, nativeAsset, eurAsset, eurAsset,
		nativeAsset, nativeAsset, eurAsset, eurAsset,
	})
	tt.Assert.NoError(err)
	tt.Assert.Len(assetMap, len(assets))

	for _, asset := range assets {
		row := assetMap[asset.String()]

		tt.Assert.True(set[row.ID])

		var assetType, assetCode, assetIssuer string
		asset.MustExtract(&assetType, &assetCode, &assetIssuer)

		tt.Assert.Equal(row.Type, assetType)
		tt.Assert.Equal(row.Code, assetCode)
		tt.Assert.Equal(row.Issuer, assetIssuer)
	}

	// CreateExpAssets handles duplicates and new rows
	assets = append(assets, usdAsset)
	assetMap, err = q.CreateExpAssets(assets)
	tt.Assert.NoError(err)
	tt.Assert.Len(assetMap, len(assets))

	for _, asset := range assets {
		row := assetMap[asset.String()]

		inSet := !asset.Equals(usdAsset)
		tt.Assert.Equal(inSet, set[row.ID])

		var assetType, assetCode, assetIssuer string
		asset.MustExtract(&assetType, &assetCode, &assetIssuer)

		tt.Assert.Equal(row.Type, assetType)
		tt.Assert.Equal(row.Code, assetCode)
		tt.Assert.Equal(row.Issuer, assetIssuer)
	}
}
