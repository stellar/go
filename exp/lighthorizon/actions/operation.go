package actions

import (
	"github.com/stellar/go/support/log"
	"io"
	"net/http"
	"strconv"

	"github.com/stellar/go/exp/lighthorizon/adapters"
	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/toid"
)

func Operations(archiveWrapper archive.Wrapper, indexStore index.Store) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// For _links rendering, imitate horizon.stellar.org links for horizon-cmp
		r.URL.Scheme = "http"
		r.URL.Host = "localhost:8080"

		if r.Method != "GET" {
			return
		}

		paginate, err := Paging(r)
		if err != nil {
			sendErrorResponse(w, http.StatusBadRequest, string(InvalidPagingParameters))
			return
		}

		if paginate.Cursor < 1 {
			paginate.Cursor = toid.New(1, 1, 1).ToInt64()
		}

		if paginate.Limit < 1 || paginate.Limit > 200 {
			paginate.Limit = 10
		}

		page := hal.Page{
			Cursor: strconv.FormatInt(paginate.Cursor, 10),
			Order:  string(paginate.Order),
			Limit:  uint64(paginate.Limit),
		}
		page.Init()
		page.FullURL = r.URL

		// For now, use a query param for now to avoid dragging in chi-router. Not
		// really the point of the experiment yet.
		account, err := RequestUnaryParam(r, "account")
		if err != nil {
			log.Error(err)
			sendErrorResponse(w, http.StatusInternalServerError, "")
			return
		}

		if account != "" {
			// Skip the cursor ahead to the next active checkpoint for this account
			var checkpoint uint32
			checkpoint, err = indexStore.NextActive(account, "all/all", uint32(toid.Parse(paginate.Cursor).LedgerSequence/64))
			if err == io.EOF {
				// never active. No results.
				page.PopulateLinks()
				sendPageResponse(w, page)
				return
			} else if err != nil {
				log.Error(err)
				sendErrorResponse(w, http.StatusInternalServerError, "")
				return
			}
			ledger := int32(checkpoint * 64)
			if ledger < 0 {
				// Check we don't overflow going from uint32 -> int32
				log.Error(err)
				sendErrorResponse(w, http.StatusInternalServerError, "")
				return
			}
			paginate.Cursor = toid.New(ledger, 1, 1).ToInt64()
		}

		//TODO - implement paginate.Order(asc/desc)
		ops, err := archiveWrapper.GetOperations(r.Context(), paginate.Cursor, paginate.Limit)
		if err != nil {
			log.Error(err)
			sendErrorResponse(w, http.StatusInternalServerError, "")
			return
		}

		for _, op := range ops {
			var response operations.Operation
			response, err = adapters.PopulateOperation(r, &op)
			if err != nil {
				log.Error(err)
				sendErrorResponse(w, http.StatusInternalServerError, "")
				return
			}

			page.Add(response)
		}

		page.PopulateLinks()
		sendPageResponse(w, page)
	}
}
