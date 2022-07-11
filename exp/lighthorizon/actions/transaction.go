package actions

import (
	"net/http"
	"strconv"

	"github.com/stellar/go/support/log"

	"github.com/stellar/go/exp/lighthorizon/adapters"
	"github.com/stellar/go/exp/lighthorizon/services"
	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/toid"
)

func Transactions(lh services.LightHorizon) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		paginate, err := paging(r)
		if err != nil {
			sendErrorResponse(w, http.StatusBadRequest, string(invalidPagingParameters))
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

		//TODO - implement paginate.Order(asc/desc)
		txns, err := lh.Transactions.GetTransactions(r.Context(), paginate.Cursor, paginate.Limit)
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
