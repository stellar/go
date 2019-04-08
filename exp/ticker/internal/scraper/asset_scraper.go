package scraper

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
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

	timeout := time.Duration(10 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get(tomlURL)
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

func domainsMatch(tomlURL *url.URL, orgURL *url.URL) bool {
	tomlDomainParts := strings.Split(tomlURL.Host, ".")
	orgDomainParts := strings.Split(orgURL.Host, ".")

	if len(orgDomainParts) < len(tomlDomainParts) {
		// Org can only be a subdomain if it has more (or equal)
		// pieces than the TOML domain
		return false
	}

	lenDiff := len(orgDomainParts) - len(tomlDomainParts)
	orgDomainParts = orgDomainParts[lenDiff:]
	orgRootDomain := strings.Join(orgDomainParts, ".")
	if tomlURL.Host != orgRootDomain {
		return false
	}

	return true
}

// isDomainVerified performs some checking to ensure we can trust the Asset's domain
func isDomainVerified(orgURL string, tomlURL string, hasCurrency bool) bool {
	if tomlURL == "" {
		return false
	}

	parsedTomlURL, err := url.Parse(tomlURL)
	if err != nil || parsedTomlURL.Scheme != "https" {
		return false
	}

	if !hasCurrency {
		return false
	}

	if orgURL == "" {
		// if no orgURL is provided, we'll simply use tomlURL, so no need
		// for domain verification
		return true
	}

	parsedOrgURL, err := url.Parse(orgURL)
	if err != nil || parsedOrgURL.Scheme != "https" {
		return false
	}

	if !domainsMatch(parsedTomlURL, parsedOrgURL) {
		return false
	}
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
			t.AnchorAssetType = currency.AnchorAssetType
			t.DisplayDecimals = currency.DisplayDecimals
			t.Name = currency.Name
			t.Desc = currency.Desc
			t.Conditions = currency.Conditions
			t.IsAssetAnchored = currency.IsAssetAnchored
			t.FixedNumber = currency.FixedNumber
			t.MaxNumber = currency.MaxNumber
			t.IsUnlimited = currency.IsUnlimited
			t.RedemptionInstructions = currency.RedemptionInstructions
			t.CollateralAddresses = currency.CollateralAddresses
			t.CollateralAddressSignatures = currency.CollateralAddressSignatures
			t.Status = currency.Status
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

// parallelCleanUpAssets filters the assets that don't match the shouldDiscardAsset criteria.
// The TOML validation is performed in parallel to improve performance.
func parallelCleanUpAssets(assets []hProtocol.AssetStat, parallelism int) (cleanAssets []TOMLAsset, numTrash int) {
	queue := make(chan TOMLAsset, parallelism)

	var mutex = &sync.Mutex{}
	var wg sync.WaitGroup
	numAssets := len(assets)
	chunkSize := int(math.Ceil(float64(numAssets) / float64(parallelism)))
	wg.Add(numAssets)

	// The assets are divided into chunks of chunkSize, and each goroutine is responsible
	// for cleaning up one chunk
	for i := 0; i < parallelism; i++ {
		go func(start int) {
			end := start + chunkSize

			if end > numAssets {
				end = numAssets
			}

			for j := start; j < end; j++ {
				if !shouldDiscardAsset(assets[j]) {
					tomlAsset, err := processAsset(assets[j])
					if err != nil {
						mutex.Lock()
						numTrash++
						mutex.Unlock()
						// Invalid assets are also sent to the queue to preserve
						// the WaitGroup count
						queue <- TOMLAsset{IsTrash: true}
						continue
					}
					queue <- tomlAsset
				} else {
					mutex.Lock()
					numTrash++
					mutex.Unlock()
					// Discarded assets are also sent to the queue to preserve
					// the WaitGroup count
					queue <- TOMLAsset{IsTrash: true}
				}
			}
		}(i * chunkSize)
	}

	// Whenever a new asset is sent to the channel, it is appended to the cleanAssets
	// slice. This does not preserve the original order, but shouldn't be an issue
	// in this case.
	go func() {
		count := 0
		for t := range queue {
			count++
			if !t.IsTrash {
				cleanAssets = append(cleanAssets, t)
			}
			fmt.Println("Total assets cleaned up:", count)
			wg.Done()
		}
	}()

	wg.Wait()

	return
}

// retrieveAssets retrieves existing assets from the Horizon API. If limit=0, will fetch all assets.
func retrieveAssets(c *horizonclient.Client, limit int) (assets []hProtocol.AssetStat, err error) {
	r := horizonclient.AssetRequest{Limit: 200}

	assetsPage, err := c.Assets(r)
	if err != nil {
		return
	}

	fmt.Println("Fetching assets from Horizon")

	for assetsPage.Links.Next.Href != assetsPage.Links.Self.Href {
		assetsPage, err = c.Assets(r)
		if err != nil {
			return
		}
		assets = append(assets, assetsPage.Embedded.Records...)

		if limit != 0 { // for performance reasons, only perform these additional checks when limit != 0
			numAssets := len(assets)
			if numAssets >= limit {
				diff := numAssets - limit
				assets = assets[0 : numAssets-diff]
				break
			}
		}

		n, err := nextCursor(assetsPage)
		if err != nil {
			return assets, err
		}
		fmt.Println("Cursor currently at:", n)

		r = horizonclient.AssetRequest{Limit: 200, Cursor: n}
	}

	fmt.Printf("Fetched: %d assets\n", len(assets))
	return
}
