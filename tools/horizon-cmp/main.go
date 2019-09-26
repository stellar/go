package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	client "github.com/stellar/go/clients/horizonclient"
	protocol "github.com/stellar/go/protocols/horizon"
	cmp "github.com/stellar/go/tools/horizon-cmp/internal"
)

const horizonOld = "http://localhost:8002"
const horizonNew = "http://localhost:8000"

// maxLevels defines the maximum number of levels deep the crawler
// should go. Here's an example crawl stack:
// Level 1 = /ledgers?order=desc (finds a link to tx details)
// Level 2 = /transactions/abcdef (finds a link to a list of operations)
// Level 3 = /transactions/abcdef/operations (will not follow any links - at level 3)
const maxLevels = 3

type pathWithLevel struct {
	Path   string
	Level  int
	Stream bool
}

func (p pathWithLevel) ID() string {
	return fmt.Sprintf("%t%s", p.Stream, p.Path)
}

var initPaths []string = []string{
	"/transactions?order=desc",
	"/transactions?order=desc",
	"/transactions?order=desc&include_failed=false",
	"/transactions?order=desc&include_failed=true",

	"/operations?order=desc",
	"/operations?order=desc&include_failed=false",
	"/operations?order=desc&include_failed=true",

	"/operations?join=transactions&order=desc",
	"/operations?join=transactions&order=desc&include_failed=false",
	"/operations?join=transactions&order=desc&include_failed=true",

	"/payments?order=desc",
	"/payments?order=desc&include_failed=false",
	"/payments?order=desc&include_failed=true",

	"/payments?join=transactions&order=desc",
	"/payments?join=transactions&order=desc&include_failed=false",
	"/payments?join=transactions&order=desc&include_failed=true",

	"/ledgers?order=desc",
	"/effects?order=desc",
	"/trades?order=desc",

	"/accounts/GAKLCFRTFDXKOEEUSBS23FBSUUVJRMDQHGCHNGGGJZQRK7BCPIMHUC4P/transactions?limit=200",
	"/accounts/GAKLCFRTFDXKOEEUSBS23FBSUUVJRMDQHGCHNGGGJZQRK7BCPIMHUC4P/transactions?limit=200&include_failed=false",
	"/accounts/GAKLCFRTFDXKOEEUSBS23FBSUUVJRMDQHGCHNGGGJZQRK7BCPIMHUC4P/transactions?limit=200&include_failed=true",

	"/accounts/GAKLCFRTFDXKOEEUSBS23FBSUUVJRMDQHGCHNGGGJZQRK7BCPIMHUC4P/operations?limit=200",
	"/accounts/GAKLCFRTFDXKOEEUSBS23FBSUUVJRMDQHGCHNGGGJZQRK7BCPIMHUC4P/payments?limit=200",
	"/accounts/GAKLCFRTFDXKOEEUSBS23FBSUUVJRMDQHGCHNGGGJZQRK7BCPIMHUC4P/effects?limit=200",

	"/accounts/GC2ZV6KGGFLQIMDVDWBWCP6LTODUDXYBLUPTUZCFHIMDCWHR43ULZITJ/trades?limit=200",
	"/accounts/GC2ZV6KGGFLQIMDVDWBWCP6LTODUDXYBLUPTUZCFHIMDCWHR43ULZITJ/offers?limit=200",

	// Pubnet markets
	"/order_book?selling_asset_type=native&buying_asset_type=credit_alphanum4&buying_asset_code=LTC&buying_asset_issuer=GCSTRLTC73UVXIYPHYTTQUUSDTQU2KQW5VKCE4YCMEHWF44JKDMQAL23",
	"/order_book?selling_asset_type=native&buying_asset_type=credit_alphanum4&buying_asset_code=XRP&buying_asset_issuer=GCSTRLTC73UVXIYPHYTTQUUSDTQU2KQW5VKCE4YCMEHWF44JKDMQAL23",
	"/order_book?selling_asset_type=native&buying_asset_type=credit_alphanum4&buying_asset_code=BTC&buying_asset_issuer=GCSTRLTC73UVXIYPHYTTQUUSDTQU2KQW5VKCE4YCMEHWF44JKDMQAL23",
	"/order_book?selling_asset_type=native&buying_asset_type=credit_alphanum4&buying_asset_code=USD&buying_asset_issuer=GBSTRUSD7IRX73RQZBL3RQUH6KS3O4NYFY3QCALDLZD77XMZOPWAVTUK",
	"/order_book?selling_asset_type=native&buying_asset_type=credit_alphanum4&buying_asset_code=SLT&buying_asset_issuer=GCKA6K5PCQ6PNF5RQBF7PQDJWRHO6UOGFMRLK3DYHDOI244V47XKQ4GP",

	"/trade_aggregations?base_asset_type=native&counter_asset_code=USD&counter_asset_issuer=GBSTRUSD7IRX73RQZBL3RQUH6KS3O4NYFY3QCALDLZD77XMZOPWAVTUK&counter_asset_type=credit_alphanum4&end_time=1551866400000&limit=200&order=desc&resolution=900000&start_time=1514764800",
}

// Starting corpus of paths to test. You may want to extend this with a list of
// paths that you want to ensure are tested.
var paths []pathWithLevel
var visitedPaths map[string]bool

func main() {
	// Get latest ledger and operate on it's cursor to get responses at a given ledger.
	ledger := getLatestLedger()
	cursor := ledger.PagingToken()

	visitedPaths = make(map[string]bool)
	for _, p := range initPaths {
		paths = append(paths, pathWithLevel{Path: getPathWithCursor(p, cursor), Level: 0, Stream: false})
		paths = append(paths, pathWithLevel{Path: getPathWithCursor(p, cursor), Level: 0, Stream: true})
	}

	// Sleep for a few seconds to make sure the second Horizon is up to speed
	time.Sleep(2 * time.Second)

	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	outputDir := fmt.Sprintf("%s/horizon-cmp-diff/%d", pwd, time.Now().Unix())

	fmt.Println("Comparing:")
	fmt.Printf("%s vs %s\n", horizonOld, horizonNew)
	fmt.Printf("[ledger=%d cursor=%s outputDir=%s]\n\n", ledger.Sequence, cursor, outputDir)

	err = os.MkdirAll(outputDir, 0744)
	if err != nil {
		panic(err)
	}

	for len(paths) > 0 {
		var pl pathWithLevel
		pl, paths = paths[0], paths[1:]

		if pl.Level > maxLevels {
			continue
		}

		if visitedPaths[pl.ID()] {
			continue
		}

		visitedPaths[pl.ID()] = true

		fmt.Printf("[stream=%t] %s ", pl.Stream, pl.Path)

		a := cmp.NewResponse(horizonOld, pl.Path, pl.Stream)
		fmt.Print(".")
		b := cmp.NewResponse(horizonNew, pl.Path, pl.Stream)
		fmt.Print(".")

		status := ""
		if a.Equal(b) {
			status = "ok"
		} else {
			status = "diff"
		}
		fmt.Printf("%s %d %d %d\n", status, a.StatusCode, a.Size(), b.Size())
		if status == "diff" {
			a.SaveDiff(outputDir, b)
		}

		newPaths := a.GetPaths()
		for _, newPath := range newPaths {
			// For all indexes with chronological sort ignore order=asc
			// without cursor. There will always be a diff if Horizon started
			// at a different ledger.
			if strings.Contains(newPath, "/ledgers") ||
				strings.Contains(newPath, "/transactions") ||
				strings.Contains(newPath, "/operations") ||
				strings.Contains(newPath, "/payments") ||
				strings.Contains(newPath, "/effects") ||
				strings.Contains(newPath, "/trades") {
				u, err := url.Parse(newPath)
				if err != nil {
					panic(err)
				}

				if u.Query().Get("cursor") == "" &&
					(u.Query().Get("order") == "" || u.Query().Get("order") == "asc") {
					continue
				}
			}

			if (strings.Contains(newPath, "/transactions") ||
				strings.Contains(newPath, "/operations") ||
				strings.Contains(newPath, "/payments")) && !strings.Contains(newPath, "include_failed") {
				prefix := "?"
				if strings.Contains(newPath, "?") {
					prefix = "&"
				}

				paths = append(paths, pathWithLevel{newPath + prefix + "include_failed=false", pl.Level + 1, false})
				paths = append(paths, pathWithLevel{newPath + prefix + "include_failed=false", pl.Level + 1, true})

				paths = append(paths, pathWithLevel{newPath + prefix + "include_failed=true", pl.Level + 1, false})
				paths = append(paths, pathWithLevel{newPath + prefix + "include_failed=true", pl.Level + 1, true})
				continue
			}

			paths = append(paths, pathWithLevel{newPath, pl.Level + 1, false})
			paths = append(paths, pathWithLevel{newPath, pl.Level + 1, true})
		}
	}
}

func getLatestLedger() protocol.Ledger {
	horizon := client.Client{
		HorizonURL: horizonOld,
		HTTP:       http.DefaultClient,
	}

	ledgers, err := horizon.Ledgers(client.LedgerRequest{
		Order: client.OrderDesc,
		Limit: 1,
	})

	if err != nil {
		panic(err)
	}

	return ledgers.Embedded.Records[0]
}

func getPathWithCursor(path, cursor string) string {
	urlObj, err := url.Parse(path)
	if err != nil {
		panic(err)
	}

	// Add cursor if not present
	q := urlObj.Query()
	if q.Get("cursor") == "" {
		q.Set("cursor", cursor)
	}

	urlObj.RawQuery = q.Encode()
	return urlObj.String()
}
