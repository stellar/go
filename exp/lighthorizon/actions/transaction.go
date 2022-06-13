package actions

import (
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"github.com/stellar/go/exp/lighthorizon/adapters"
	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/index"
	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/toid"
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

		paginate, err := Paging(r)
		if err != nil {
			fmt.Fprintf(w, "Error: %v", err)
			sendErrorResponse(w, http.StatusBadRequest)
			return
		}

		if paginate.Cursor < 1 {
			paginate.Cursor = toid.New(1, 1, 1).ToInt64()
		}

		if paginate.Limit < 1 {
			paginate.Limit = 10
		}

		page := hal.Page{}
		page.Init()
		page.FullURL = r.URL

		// For now, use a query param for now to avoid dragging in chi-router. Not
		// really the point of the experiment yet.
		txId, err := RequestUnaryParam(r, "id")
		if err != nil {
			fmt.Fprintf(w, "Error: %v", err)
			sendErrorResponse(w, http.StatusBadRequest)
			return
		}

		if txId != "" {
			// if 'id' is on request, it overrides any paging cursor that may be on request.
			var b []byte
			b, err = hex.DecodeString(txId)
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

			if paginate.Cursor, err = indexStore.TransactionTOID(hash); err != nil {
				fmt.Fprintf(w, "Error: %v", err)
				sendErrorResponse(w, http.StatusInternalServerError)
			}
			if err == io.EOF {
				page.PopulateLinks()
				sendPageResponse(w, page)
				return
			}
		}

		txns, err := archiveWrapper.GetTransactions(r.Context(), paginate.Cursor, paginate.Limit)
		if err != nil {
			fmt.Fprintf(w, "Error: %v", err)
			sendErrorResponse(w, http.StatusInternalServerError)
			return
		}

		for _, txn := range txns {
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
