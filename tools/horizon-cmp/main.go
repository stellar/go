package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	protocol "github.com/stellar/go/protocols/horizon"
	cmp "github.com/stellar/go/tools/horizon-cmp/internal"
)

const HorizonOld = "http://localhost:8000"
const HorizonNew = "http://localhost:8001"

const MaxLevels = 3

type PathWithLevel struct {
	Path  string
	Level int
}

// Starting corpus of paths to test
var paths []PathWithLevel = []PathWithLevel{
	PathWithLevel{"/transactions?order=desc", 0},
	PathWithLevel{"/transactions?order=desc&include_failed=true", 0},

	PathWithLevel{"/operations?order=desc", 0},
	PathWithLevel{"/operations?order=desc&include_failed=true", 0},

	PathWithLevel{"/payments?order=desc", 0},
	PathWithLevel{"/payments?order=desc&include_failed=true", 0},

	PathWithLevel{"/ledgers?order=desc", 0},
	PathWithLevel{"/effects?order=desc", 0},
	PathWithLevel{"/trades?order=desc", 0},
	// PathWithLevel{"/assets", 0},
	PathWithLevel{"/fee_stats", 0},

	// Pubnet markets
	PathWithLevel{"/order_book?selling_asset_type=native&buying_asset_type=credit_alphanum4&buying_asset_code=LTC&buying_asset_issuer=GCSTRLTC73UVXIYPHYTTQUUSDTQU2KQW5VKCE4YCMEHWF44JKDMQAL23", 0},
	PathWithLevel{"/order_book?selling_asset_type=native&buying_asset_type=credit_alphanum4&buying_asset_code=XRP&buying_asset_issuer=GCSTRLTC73UVXIYPHYTTQUUSDTQU2KQW5VKCE4YCMEHWF44JKDMQAL23", 0},
	PathWithLevel{"/order_book?selling_asset_type=native&buying_asset_type=credit_alphanum4&buying_asset_code=BTC&buying_asset_issuer=GCSTRLTC73UVXIYPHYTTQUUSDTQU2KQW5VKCE4YCMEHWF44JKDMQAL23", 0},
	PathWithLevel{"/order_book?selling_asset_type=native&buying_asset_type=credit_alphanum4&buying_asset_code=USD&buying_asset_issuer=GBSTRUSD7IRX73RQZBL3RQUH6KS3O4NYFY3QCALDLZD77XMZOPWAVTUK", 0},
	PathWithLevel{"/order_book?selling_asset_type=native&buying_asset_type=credit_alphanum4&buying_asset_code=SLT&buying_asset_issuer=GCKA6K5PCQ6PNF5RQBF7PQDJWRHO6UOGFMRLK3DYHDOI244V47XKQ4GP", 0},

	PathWithLevel{"/trade_aggregations?base_asset_type=native&counter_asset_code=USD&counter_asset_issuer=GBSTRUSD7IRX73RQZBL3RQUH6KS3O4NYFY3QCALDLZD77XMZOPWAVTUK&counter_asset_type=credit_alphanum4&end_time=1551866400000&limit=200&order=desc&resolution=900000&start_time=1514764800", 0},
}

var visitedPaths map[string]bool

func init() {
	visitedPaths = make(map[string]bool)
}

func main() {
	// Get latest ledger and operate on it's cursor to get responses at a given ledger.
	ledger := getLatestLedger()
	cursor := ledger.PagingToken()

	// Sleep for a few sec to make sure second Horizon is up to speed
	time.Sleep(2 * time.Second)

	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	outputDir := fmt.Sprintf("%s/horizon-cmp-diff/%d", pwd, time.Now().Unix())

	fmt.Println("Comparing:")
	fmt.Printf("%s vs %s\n", HorizonOld, HorizonNew)
	fmt.Printf("[ledger=%d cursor=%s outputDir=%s]\n\n", ledger.Sequence, cursor, outputDir)

	err = os.MkdirAll(outputDir, 0744)
	if err != nil {
		panic(err)
	}

	for {
		if len(paths) == 0 {
			return
		}

		var pl PathWithLevel
		pl, paths = paths[0], paths[1:]

		p := pl.Path
		level := pl.Level

		if level > MaxLevels {
			continue
		}

		if visitedPaths[p] {
			continue
		}

		visitedPaths[p] = true

		fmt.Printf("%s ", p)

		p = getPathWithCursor(p, cursor)

		a := cmp.NewResponse(HorizonOld, p)
		fmt.Print(".")
		b := cmp.NewResponse(HorizonNew, p)
		fmt.Print(".")

		status := ""
		if a.Equal(b) {
			status = "ok"
		} else {
			status = "diff"
		}
		fmt.Println(status)
		if status == "diff" {
			a.SaveDiff(outputDir, b)
		}

		newPaths := a.GetPaths()
		for _, newPath := range newPaths {
			// Such links can get recent ledgers data that may be different
			// if Horizon nodes are not at the same ledger.
			if strings.Contains(newPath, "order=asc") {
				continue
			}

			paths = append(paths, PathWithLevel{newPath, level + 1})
		}
	}
}

func getLatestLedger() protocol.Ledger {
	ledgersResponse := struct {
		Embedded struct {
			Records []protocol.Ledger `json:"records"`
		} `json:"_embedded"`
	}{}

	resp, err := http.Get(HorizonOld + "/ledgers?order=desc&limit=1")
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		panic(resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(body, &ledgersResponse)
	if err != nil {
		panic(err)
	}

	return ledgersResponse.Embedded.Records[0]
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
