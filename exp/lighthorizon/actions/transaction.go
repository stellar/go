package actions

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/stellar/go/exp/lighthorizon/adapters"
	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/index"
	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/render/hal"
)

func Transactions(archiveWrapper archive.Wrapper, indexStore index.Store) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// For _links rendering, imitate horizon.stellar.org links for horizon-cmp
		r.URL.Scheme = "http"
		r.URL.Host = "localhost:8080"

		if r.Method != "GET" {
			return
		}

		query, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			fmt.Fprintf(w, "Error: %v", err)
			return
		}

		page := hal.Page{}
		page.Init()
		page.FullURL = r.URL

		// For now, use a query param for now to avoid dragging in chi-router. Not
		// really the point of the experiment yet.
		id := query.Get("id")
		var cursor int64
		if id != "" {
			b, err := hex.DecodeString(id)
			if err != nil {
				fmt.Fprintf(w, "Error: %v", err)
				return
			}
			if len(b) != 32 {
				fmt.Fprintf(w, "Error: invalid hash")
				return
			}
			var hash [32]byte
			copy(hash[:], b)
			// Skip the cursor ahead to the next active checkpoint for this account
			txnToid, err := indexStore.TransactionTOID(hash)
			if err == io.EOF {
				// never active. No results.
				page.PopulateLinks()

				encoder := json.NewEncoder(w)
				encoder.SetIndent("", "  ")
				err = encoder.Encode(page)
				if err != nil {
					fmt.Fprintf(w, "Error: %v", err)
					return
				}
				return
			} else if err != nil {
				fmt.Fprintf(w, "Error: %v", err)
				return
			}
			cursor = txnToid
		}

		txns, err := archiveWrapper.GetTransactions(cursor, 1)
		if err != nil {
			fmt.Fprintf(w, "Error: %v", err)
			return
		}

		for _, txn := range txns {
			hash, err := txn.TransactionHash()
			if err != nil {
				fmt.Fprintf(w, "Error: %v", err)
				return
			}
			if id != "" && hash != id {
				continue
			}
			var response hProtocol.Transaction
			response, err = adapters.PopulateTransaction(r, &txn)
			if err != nil {
				fmt.Fprintf(w, "Error: %v", err)
				return
			}

			page.Add(response)
		}

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
