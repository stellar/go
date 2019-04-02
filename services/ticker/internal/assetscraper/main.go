package assetscraper

import (
	"fmt"
	"net/url"

	horizonclient "github.com/stellar/go/exp/clients/horizon"
	hProtocol "github.com/stellar/go/protocols/horizon"
)

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

// FetchAllAssets fetches all assets from the Horizon public net
func FetchAllAssets() (assets []hProtocol.AssetStat, err error) {
	c := horizonclient.DefaultPublicNetClient
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
