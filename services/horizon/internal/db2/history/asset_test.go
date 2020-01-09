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
	rows, err := q.CreateExpAssets(assets)
	tt.Assert.NoError(err)
	tt.Assert.Len(rows, len(assets))

	set := map[int64]bool{}
	for i, asset := range assets {
		row := rows[i]

		tt.Assert.False(set[row.ID])
		set[row.ID] = true

		var assetType, assetCode, assetIssuer string
		asset.MustExtract(&assetType, &assetCode, &assetIssuer)

		tt.Assert.Equal(row.Type, assetType)
		tt.Assert.Equal(row.Code, assetCode)
		tt.Assert.Equal(row.Issuer, assetIssuer)
	}

	// CreateExpAssets handles duplicates
	rows, err = q.CreateExpAssets([]xdr.Asset{
		nativeAsset, nativeAsset, eurAsset, eurAsset,
		nativeAsset, nativeAsset, eurAsset, eurAsset,
	})
	tt.Assert.NoError(err)
	tt.Assert.Len(rows, 8)

	for i, row := range rows {
		asset := assets[(i/2)%2]

		tt.Assert.True(set[row.ID])

		var assetType, assetCode, assetIssuer string
		asset.MustExtract(&assetType, &assetCode, &assetIssuer)

		tt.Assert.Equal(row.Type, assetType)
		tt.Assert.Equal(row.Code, assetCode)
		tt.Assert.Equal(row.Issuer, assetIssuer)
	}

	// CreateExpAssets handles duplicates and new rows
	assets = append(assets, usdAsset)
	rows, err = q.CreateExpAssets(assets)
	tt.Assert.NoError(err)
	tt.Assert.Len(rows, 3)

	for i, row := range rows {
		asset := assets[i]

		// only the last asset is new
		tt.Assert.Equal(set[row.ID], i < 2)

		var assetType, assetCode, assetIssuer string
		asset.MustExtract(&assetType, &assetCode, &assetIssuer)

		tt.Assert.Equal(row.Type, assetType)
		tt.Assert.Equal(row.Code, assetCode)
		tt.Assert.Equal(row.Issuer, assetIssuer)
	}
}
