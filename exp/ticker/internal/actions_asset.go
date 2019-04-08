package ticker

import (
	"encoding/json"
	"fmt"
	"strings"

	horizonclient "github.com/stellar/go/exp/clients/horizon"
	"github.com/stellar/go/exp/ticker/internal/scraper"
	"github.com/stellar/go/exp/ticker/internal/tickerdb"
	"github.com/stellar/go/exp/ticker/internal/utils"
)

// RefreshAssets scrapes the most recent asset list and ingests then into the db.
func RefreshAssets(s *tickerdb.TickerSession) (err error) {
	c := horizonclient.DefaultPublicNetClient
	finalAssetList, err := scraper.FetchAllAssets(c, 0, 500)
	if err != nil {
		return
	}

	err = writeAssetsToFile(finalAssetList, "assets.json")
	if err != nil {
		fmt.Println("Could not write assets to file")
	}

	for _, finalAsset := range finalAssetList {
		dbAsset := finalAssetToDBAsset(finalAsset)
		err = s.InsertOrUpdateAsset(&dbAsset, []string{"code", "public_key"})
		if err != nil {
			fmt.Println("Error inserting asset:", dbAsset, err)
		}
	}

	return
}

// writeAssetsToFile creates a list of assets exported in a JSON file.
func writeAssetsToFile(assets []scraper.FinalAsset, filename string) (err error) {
	jsonAssets, err := json.MarshalIndent(assets, "", "\t")
	if err != nil {
		return
	}

	numBytes, err := utils.WriteJSONToFile(jsonAssets, filename)
	if err != nil {
		return
	}
	fmt.Printf("Wrote %d bytes to %s\n", numBytes, filename)
	return
}

// finalAssetToDBAsset converts a scraper.FinalAsset to a tickerdb.Asset.
func finalAssetToDBAsset(asset scraper.FinalAsset) tickerdb.Asset {
	return tickerdb.Asset{
		Code:                        asset.Code,
		PublicKey:                   asset.Issuer,
		Type:                        asset.Type,
		NumAccounts:                 asset.NumAccounts,
		AuthRequired:                asset.AuthRequired,
		AuthRevocable:               asset.AuthRevocable,
		Amount:                      asset.Amount,
		AssetControlledByDomain:     asset.AssetControlledByDomain,
		AnchorAssetCode:             asset.AnchorAsset,
		AnchorAssetType:             asset.AnchorAssetType,
		IsValid:                     asset.IsValid,
		ValidationError:             asset.Error,
		LastValid:                   asset.LastValid,
		LastChecked:                 asset.LastChecked,
		DisplayDecimals:             asset.DisplayDecimals,
		Name:                        asset.Name,
		Desc:                        asset.Desc,
		Conditions:                  asset.Conditions,
		IsAssetAnchored:             asset.IsAssetAnchored,
		FixedNumber:                 asset.FixedNumber,
		MaxNumber:                   asset.MaxNumber,
		IsUnlimited:                 asset.IsUnlimited,
		RedemptionInstructions:      asset.RedemptionInstructions,
		CollateralAddresses:         strings.Join(asset.CollateralAddresses, ","),
		CollateralAddressSignatures: strings.Join(asset.CollateralAddressSignatures, ","),
		Countries:                   asset.Countries,
		Status:                      asset.Status,
	}
}
