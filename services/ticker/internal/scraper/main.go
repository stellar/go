package scraper

import (
	"fmt"

	horizonclient "github.com/stellar/go/exp/clients/horizon"
	hProtocol "github.com/stellar/go/protocols/horizon"
)

// FetchAllAssets fetches all assets from the Horizon public net
func FetchAllAssets(c *horizonclient.Client) (assets []hProtocol.AssetStat, err error) {
	dirtyAssets, err := retrieveAssets(c)
	if err != nil {
		return
	}

	assets, numTrash := cleanUpAssets(dirtyAssets)

	fmt.Printf(
		"Scanned %d entries. Trash: %d. Non-trash: %d\n",
		len(dirtyAssets),
		numTrash,
		len(assets),
	)
	return
}
