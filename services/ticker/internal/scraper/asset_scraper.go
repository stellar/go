package scraper

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	horizonclient "github.com/stellar/go/exp/clients/horizon"
	hProtocol "github.com/stellar/go/protocols/horizon"
)

// nextCursor finds the cursor parameter on the "next" link of an AssetPage
func nextCursor(assetsPage hProtocol.AssetsPage) (cursor string, err error) {
	nexturl := assetsPage.Links.Next.Href
	u, err := url.Parse(nexturl)
	if err != nil {
		return
	}

	m, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return
	}
	cursor = m["cursor"][0]

	return
}

// shouldDiscardAsset maps the criteria for discarding an asset from the asset index
func shouldDiscardAsset(asset hProtocol.AssetStat) bool {
	if asset.Amount == "" {
		return true
	}
	f, _ := strconv.ParseFloat(asset.Amount, 64)
	if f == 0.0 {
		return true
	}
	// [StellarX Ticker]: assets need at least some adoption to show up
	if asset.NumAccounts < 10 {
		return true
	}
	if asset.Code == "REMOVE" {
		return true
	}
	// [StellarX Ticker]: assets with at least 100 accounts get a pass,
	// even with toml issues
	if asset.NumAccounts >= 100 {
		return false
	}
	if asset.Links.Toml.Href == "" {
		return true
	}
	// [StellarX Ticker]: TOML files should be hosted on HTTPS
	if !strings.HasPrefix(asset.Links.Toml.Href, "https://") {
		return true
	}
	return false
}

// decodeTOMLIssuer decodes retrieved TOML issuer data into a TOMLIssuer struct
func decodeTOMLIssuer(tomlData string) (issuer TOMLIssuer, err error) {
	_, err = toml.Decode(tomlData, &issuer)
	return
}

// fetchTOMLData fetches the TOML data for a given hProtocol.AssetStat
func fetchTOMLData(asset hProtocol.AssetStat) (data string, err error) {
	tomlURL := asset.Links.Toml.Href

	if tomlURL == "" {
		err = errors.New("Asset does not have a TOML URL")
		return
	}

	resp, err := http.Get(tomlURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	data = string(body)
	return
}

// isDomainVerified performs some checking to ensure we can trust the Asset's domain
func isDomainVerified(orgURL string, tomlURL string, hasCurrency bool) bool {
	// TODO
	return true
}

// makeTomlAsset aggregates Horizon Data with TOML Data
func makeTOMLAsset(
	asset hProtocol.AssetStat,
	issuer TOMLIssuer,
	errors []error,
) (t TOMLAsset, err error) {
	amount, err := strconv.ParseFloat(asset.Amount, 64)
	if err != nil {
		return
	}

	t = TOMLAsset{
		Type:          asset.Type,
		Code:          asset.Code,
		Issuer:        asset.Issuer,
		NumAccounts:   asset.NumAccounts,
		AuthRequired:  asset.Flags.AuthRequired,
		AuthRevocable: asset.Flags.AuthRevocable,
		Amount:        amount,
		IssuerDetails: issuer,
	}

	hasCurrency := false
	for _, currency := range t.IssuerDetails.Currencies {
		if currency.Code == asset.Code && currency.Issuer == asset.Issuer {
			hasCurrency = true
			t.AnchorAsset = currency.AnchorAsset
			t.AnchorAsset = currency.AnchorAsset
		}
	}
	t.AssetControlledByDomain = isDomainVerified(
		t.IssuerDetails.Documentation.OrgURL,
		asset.Links.Toml.Href,
		hasCurrency,
	)

	// TODO: determine if the asset is valid even if the issuer doesn't have
	// it listed in its currencies

	now := time.Now()
	if len(errors) > 0 {
		t.Error = fmt.Sprintf("%v", errors)
		t.IsValid = false
	} else {
		t.LastValid = now
		t.IsValid = true
	}
	t.LastChecked = now
	t.AnchorAssetType = strings.ToLower(t.AnchorAssetType)

	return
}

// processAsset merges data from an AssetStat with data retrieved from its corresponding TOML file
func processAsset(asset hProtocol.AssetStat) (processedAsset TOMLAsset, err error) {
	var errors []error

	tomlData, err := fetchTOMLData(asset)
	if err != nil {
		errors = append(errors, err)
	}

	issuer, err := decodeTOMLIssuer(tomlData)
	if err != nil {
		errors = append(errors, err)
	}

	processedAsset, err = makeTOMLAsset(asset, issuer, errors)
	if err != nil {
		return
	}

	return
}

// cleanUpAssets filters the assets that don't match the shouldDiscardAsset criteria
func cleanUpAssets(assets []hProtocol.AssetStat) (cleanAssets []TOMLAsset, numTrash int) {
	fmt.Println("Cleaning up assets")
	// TODO: use some paralellization here to improve speed?
	for _, asset := range assets {
		if !shouldDiscardAsset(asset) {
			tomlAsset, err := processAsset(asset)
			if err != nil {
				fmt.Println("Error processing asset:", err)
				continue
			}
			fmt.Println(tomlAsset)
			cleanAssets = append(cleanAssets, tomlAsset)
		} else {
			// TODO: define if we should start storing the "Trash" assets as well
			numTrash++
		}
	}

	return
}

// retrieveAssets retrieves all existing assets from the Horizon API
func retrieveAssets(c *horizonclient.Client) (assets []hProtocol.AssetStat, err error) {
	r := horizonclient.AssetRequest{Limit: 200}

	assetsPage, err := c.Assets(r)
	if err != nil {
		return
	}

	for assetsPage.Links.Next.Href != assetsPage.Links.Self.Href {
		assetsPage, err = c.Assets(r)
		if err != nil {
			return
		}
		assets = append(assets, assetsPage.Embedded.Records...)

		n, err := nextCursor(assetsPage)
		if err != nil {
			return assets, err
		}
		fmt.Println("Fetching all assets with cursor at:", n)

		r = horizonclient.AssetRequest{Limit: 200, Cursor: n}
	}

	return
}
