package actions

import (
	"encoding/hex"
	"fmt"
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
			sendErrorResponse(w, http.StatusMethodNotAllowed)
			return
		}

		query, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			fmt.Fprintf(w, "Error: %v", err)
			sendErrorResponse(w, http.StatusBadRequest)
			return
		}

		page := hal.Page{}
		page.Init()
		page.FullURL = r.URL

		// For now, use a query param for now to avoid dragging in chi-router. Not
		// really the point of the experiment yet.
		id := query.Get("id")
		var cursor int64
		var eof bool

		if id != "" {
			var b []byte
			b, err = hex.DecodeString(id)
			if err != nil {
				fmt.Fprintf(w, "Error: %v", err)
				sendErrorResponse(w, http.StatusBadRequest)
				return
			}
			if len(b) != 32 {
				sendErrorResponse(w, http.StatusBadRequest)
				return
			}
			var hash [32]byte
			copy(hash[:], b)

			if cursor, eof, err = indexedCursorFromHash(hash, indexStore); err != nil {
				fmt.Fprintf(w, "Error: %v", err)
				sendErrorResponse(w, http.StatusInternalServerError)
			}
			if eof {
				page.PopulateLinks()
				sendPageResponse(w, page)
				return
			}
		}

		txns, err := archiveWrapper.GetTransactions(cursor, 1)
		if err != nil {
			fmt.Fprintf(w, "Error: %v", err)
			sendErrorResponse(w, http.StatusInternalServerError)
			return
		}

		for _, txn := range txns {
			hash, err := txn.TransactionHash()
			if err != nil {
				fmt.Fprintf(w, "Error: %v", err)
				sendErrorResponse(w, http.StatusInternalServerError)
				return
			}
			if id != "" && hash != id {
				continue
			}
			var response hProtocol.Transaction
			response, err = adapters.PopulateTransaction(r, &txn)
			if err != nil {
				fmt.Fprintf(w, "Error: %v", err)
				sendErrorResponse(w, http.StatusInternalServerError)
				return
			}

			page.Add(response)
		}

		page.PopulateLinks()
		sendPageResponse(w, page)
	}
}
