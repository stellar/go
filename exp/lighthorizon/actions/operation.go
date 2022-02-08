package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/stellar/go/exp/lighthorizon/adapters"
	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/toid"
)

func Operations(archiveWrapper archive.Wrapper) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			return
		}

		query, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			fmt.Fprintf(w, "Error: %v", err)
			return
		}

		var cursor int64
		if query.Get("cursor") == "" {
			cursor = toid.New(1, 1, 1).ToInt64()
		} else {
			cursor, err = strconv.ParseInt(query.Get("cursor"), 10, 64)
			if err != nil {
				fmt.Fprintf(w, "Error: %v", err)
				return
			}
		}

		var limit int64
		if query.Get("limit") == "" {
			limit = 10
		} else {
			limit, err = strconv.ParseInt(query.Get("limit"), 10, 64)
			if err != nil {
				fmt.Fprintf(w, "Error: %v", err)
				return
			}
		}

		if limit == 0 || limit > 200 {
			limit = 10
		}

		ops, err := archiveWrapper.GetOperations(cursor, limit)
		if err != nil {
			fmt.Fprintf(w, "Error: %v", err)
			return
		}

		// For _links rendering, imitate horizon.stellar.org links for horizon-cmp
		r.URL.Scheme = "http"
		r.URL.Host = "localhost:8080"

		page := hal.Page{
			Cursor: query.Get("cursor"),
			Order:  "asc",
			Limit:  uint64(limit),
		}
		page.Init()

		for _, op := range ops {
			var response operations.Operation
			response, err = adapters.PopulateOperation(r, &op)
			if err != nil {
				fmt.Fprintf(w, "Error: %v", err)
				return
			}

			page.Add(response)
		}

		page.FullURL = r.URL
		page.PopulateLinks()

		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(page)
		if err != nil {
			fmt.Fprintf(w, "Error: %v", err)
			return
		}
	}
}
