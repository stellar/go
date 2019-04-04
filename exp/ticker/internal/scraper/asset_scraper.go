package scraper

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

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

// cleanUpAssets filters the assets that don't match the shouldDiscardAsset criteria
func cleanUpAssets(assets []hProtocol.AssetStat) (cleanAssets []hProtocol.AssetStat, numTrash int) {
	for _, asset := range assets {
		if !shouldDiscardAsset(asset) {
			cleanAssets = append(cleanAssets, asset)
		} else {
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
