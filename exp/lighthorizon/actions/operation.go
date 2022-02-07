package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/stellar/go/exp/lighthorizon/adapters"
	"github.com/stellar/go/exp/lighthorizon/archive"
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

		for _, op := range ops {
			resp, err := adapters.PopulateOperation(&op)
			if err != nil {
				fmt.Fprintf(w, "Error: %v", err)
				return
			}

			encoder := json.NewEncoder(w)
			encoder.SetIndent("", "    ")
			err = encoder.Encode(resp)
			if err != nil {
				fmt.Fprintf(w, "Error: %v", err)
				return
			}
		}
	}
}
