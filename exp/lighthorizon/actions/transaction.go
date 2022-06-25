package actions

import (
	"encoding/hex"
	"github.com/stellar/go/support/log"
	"io"
	"net/http"
	"strconv"

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
			sendErrorResponse(w, http.StatusMethodNotAllowed, "")
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

		if paginate.Limit < 1 {
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
		txId, err := RequestUnaryParam(r, "id")
		if err != nil {
			log.Error(err)
			sendErrorResponse(w, http.StatusInternalServerError, "")
			return
		}

		if txId != "" {
			// if 'id' is on request, it overrides any paging cursor that may be on request.
			var b []byte
			b, err = hex.DecodeString(txId)
			if err != nil {
				sendErrorResponse(w, http.StatusBadRequest, "Invalid transaction id request parameter, not valid hex encoding")
				return
			}
			if len(b) != 32 {
				sendErrorResponse(w, http.StatusBadRequest, "Invalid transaction id request parameter, the encoded hex value must decode to length of 32 bytes")
				return
			}
			var hash [32]byte
			copy(hash[:], b)

			if paginate.Cursor, err = indexStore.TransactionTOID(hash); err != nil {
				log.Error(err)
				sendErrorResponse(w, http.StatusInternalServerError, "")
			}
			if err == io.EOF {
				page.PopulateLinks()
				sendPageResponse(w, page)
				return
			}
		}

		//TODO - implement paginate.Order(asc/desc)
		txns, err := archiveWrapper.GetTransactions(r.Context(), paginate.Cursor, paginate.Limit)
		if err != nil {
			log.Error(err)
			sendErrorResponse(w, http.StatusInternalServerError, "")
			return
		}

		for _, txn := range txns {
			var response hProtocol.Transaction
			response, err = adapters.PopulateTransaction(r, &txn)
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
