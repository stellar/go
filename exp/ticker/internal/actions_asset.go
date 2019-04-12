package ticker

import (
	"encoding/json"
	"strings"

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

	// TODO: move this to a separate function:
	filename := "assets.json"
	numBytes, err := writeAssetsToFile(finalAssetList, filename)
	if err != nil {
		l.Error("Could not write assets to file.")
	}
	l.Infof("Wrote %d bytes to %s\n", numBytes, filename)
	// TODO END

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

// writeAssetsToFile creates a list of assets exported in a JSON file.
func writeAssetsToFile(assets []scraper.FinalAsset, filename string) (numBytes int, err error) {
	jsonAssets, err := json.MarshalIndent(assets, "", "\t")
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
