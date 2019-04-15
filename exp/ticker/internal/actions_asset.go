package ticker

import (
	"encoding/json"
	"strings"
	"time"

	horizonclient "github.com/stellar/go/exp/clients/horizon"
	"github.com/stellar/go/exp/ticker/internal/scraper"
	"github.com/stellar/go/exp/ticker/internal/tickerdb"
	"github.com/stellar/go/exp/ticker/internal/utils"
	hlog "github.com/stellar/go/support/log"
)

// RefreshAssets scrapes the most recent asset list and ingests then into the db.
func RefreshAssets(s *tickerdb.TickerSession, c *horizonclient.Client, l *hlog.Entry) (err error) {
	sc := scraper.ScraperConfig{
		Client: c,
		Logger: l,
	}
	finalAssetList, err := sc.FetchAllAssets(0, 500)
	if err != nil {
		return
	}

	for _, finalAsset := range finalAssetList {
		dbIssuer := tomlIssuerToDBIssuer(finalAsset.IssuerDetails)
		if dbIssuer.PublicKey == "" {
			dbIssuer.PublicKey = finalAsset.Issuer
		}
		issuerID, err := s.InsertOrUpdateIssuer(&dbIssuer, []string{"public_key"})
		if err != nil {
			l.Error("Error inserting issuer:", dbIssuer, err)
			continue
		}

		dbAsset := finalAssetToDBAsset(finalAsset, issuerID)
		err = s.InsertOrUpdateAsset(&dbAsset, []string{"code", "issuer_account", "issuer_id"})
		if err != nil {
			l.Error("Error inserting asset:", dbAsset, err)
		}
	}

	return
}

// GenerateAssetsFile generates a file with the info about all valid scraped Assets
func GenerateAssetsFile(s *tickerdb.TickerSession, l *hlog.Entry, filename string) error {
	l.Infoln("Retrieving asset data from db...")
	var finalAssets []scraper.FinalAsset
	validAssets, err := s.GetAllValidAssets()
	if err != nil {
		return err
	}

	for _, dbAsset := range validAssets {
		finalAsset := dbAssetToFinalAsset(dbAsset)
		finalAssets = append(finalAssets, finalAsset)
	}
	l.Infoln("Asset data successfully retrieved! Writing to: ", filename)
	assetSummary := AssetSummary{
		GeneratedAt: time.Now().UnixNano() / 1000000,
		Assets:      finalAssets,
	}
	numBytes, err := writeAssetSummaryToFile(assetSummary, filename)
	if err != nil {
		return err
	}
	l.Infof("Wrote %d bytes to %s\n", numBytes, filename)
	return nil
}

// writeAssetSummaryToFile creates a list of assets exported in a JSON file.
func writeAssetSummaryToFile(assetSummary AssetSummary, filename string) (numBytes int, err error) {
	jsonAssets, err := json.MarshalIndent(assetSummary, "", "\t")
	if err != nil {
		return
	}

	numBytes, err = utils.WriteJSONToFile(jsonAssets, filename)
	if err != nil {
		return
	}
	return
}

// finalAssetToDBAsset converts a scraper.TOMLAsset to a tickerdb.Asset.
func finalAssetToDBAsset(asset scraper.FinalAsset, issuerID int32) tickerdb.Asset {
	return tickerdb.Asset{
		Code:                        asset.Code,
		IssuerID:                    issuerID,
		IssuerAccount:               asset.Issuer,
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

// finalAssetToDBAsset converts a tickerdb.Asset to a scraper.TOMLAsset.
func dbAssetToFinalAsset(asset tickerdb.Asset) scraper.FinalAsset {
	return scraper.FinalAsset{
		Code:                        asset.Code,
		Issuer:                      asset.IssuerAccount,
		Type:                        asset.Type,
		NumAccounts:                 asset.NumAccounts,
		AuthRequired:                asset.AuthRequired,
		AuthRevocable:               asset.AuthRevocable,
		Amount:                      asset.Amount,
		AssetControlledByDomain:     asset.AssetControlledByDomain,
		AnchorAsset:                 asset.AnchorAssetCode,
		AnchorAssetType:             asset.AnchorAssetType,
		IsValid:                     asset.IsValid,
		Error:                       asset.ValidationError,
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
		CollateralAddresses:         strings.Split(asset.CollateralAddresses, ","),
		CollateralAddressSignatures: strings.Split(asset.CollateralAddressSignatures, ","),
		Countries:                   asset.Countries,
		Status:                      asset.Status,
	}
}
