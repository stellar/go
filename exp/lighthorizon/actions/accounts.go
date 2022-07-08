package actions

import (
	"net/http"
	"strconv"

	"github.com/stellar/go/support/log"

	"github.com/stellar/go/exp/lighthorizon/adapters"
	"github.com/stellar/go/exp/lighthorizon/services"
	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/toid"
)

const (
	urlAccountId = "account_id"
)

func accountRequestParams(w http.ResponseWriter, r *http.Request) (string, pagination) {
	var accountId string
	var accountErr bool

	if accountId, accountErr = getURLParam(r, urlAccountId); accountErr != true {
		sendErrorResponse(w, http.StatusBadRequest, "")
		return "", pagination{}
	}

	paginate, err := paging(r)
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, string(invalidPagingParameters))
		return "", pagination{}
	}

	if paginate.Cursor < 1 {
		paginate.Cursor = toid.New(1, 1, 1).ToInt64()
	}

	if paginate.Limit < 1 {
		paginate.Limit = 10
	}

	return accountId, paginate
}

func NewTXByAccountHandler(lightHorizon services.LightHorizon) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var accountId string
		var paginate pagination

		if accountId, paginate = accountRequestParams(w, r); accountId == "" {
			return
		}

		page := hal.Page{
			Cursor: strconv.FormatInt(paginate.Cursor, 10),
			Order:  string(paginate.Order),
			Limit:  uint64(paginate.Limit),
		}
		page.Init()
		page.FullURL = r.URL

		txns, err := lightHorizon.GetTransactionsByAccount(ctx, paginate.Cursor, paginate.Limit, accountId)
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

func NewOpsByAccountHandler(lightHorizon services.LightHorizon) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var accountId string
		var paginate pagination

		if accountId, paginate = accountRequestParams(w, r); accountId == "" {
			return
		}

		page := hal.Page{
			Cursor: strconv.FormatInt(paginate.Cursor, 10),
			Order:  string(paginate.Order),
			Limit:  uint64(paginate.Limit),
		}
		page.Init()
		page.FullURL = r.URL

		ops, err := lightHorizon.GetOperationsByAccount(ctx, paginate.Cursor, paginate.Limit, accountId)
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
