package actions

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"

	"github.com/stellar/go/exp/lighthorizon/adapters"
	"github.com/stellar/go/exp/lighthorizon/services"
	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/support/render/hal"
	supportProblem "github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/toid"
)

const (
	urlAccountId = "account_id"
)

func accountRequestParams(w http.ResponseWriter, r *http.Request) (string, pagination, error) {
	var accountId string
	var accountErr bool

	if accountId, accountErr = getURLParam(r, urlAccountId); !accountErr {
		return "", pagination{}, errors.New("unable to find account_id in url path")
	}

	paginate, err := paging(r)
	if err != nil {
		return "", pagination{}, err
	}

	if paginate.Cursor < 1 {
		paginate.Cursor = toid.New(1, 1, 1).ToInt64()
	}

	if paginate.Limit == 0 {
		paginate.Limit = 10
	}

	return accountId, paginate, nil
}

func NewTXByAccountHandler(lightHorizon services.LightHorizon) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var accountId string
		var paginate pagination
		var err error

		if accountId, paginate, err = accountRequestParams(w, r); err != nil {
			errorMsg := supportProblem.MakeInvalidFieldProblem("account_id", err)
			sendErrorResponse(r.Context(), w, *errorMsg)
			return
		}

		page := hal.Page{
			Cursor: strconv.FormatInt(paginate.Cursor, 10),
			Order:  string(paginate.Order),
			Limit:  uint64(paginate.Limit),
		}
		page.Init()
		page.FullURL = r.URL

		txns, err := lightHorizon.Transactions.GetTransactionsByAccount(ctx, paginate.Cursor, paginate.Limit, accountId)
		if err != nil {
			log.Error(err)
			sendErrorResponse(r.Context(), w, supportProblem.ServerError)
			return
		}

		encoder := xdr.NewEncodingBuffer()
		for _, txn := range txns {
			var response hProtocol.Transaction
			response, err = adapters.PopulateTransaction(r.URL, &txn, encoder)
			if err != nil {
				log.Error(err)
				sendErrorResponse(r.Context(), w, supportProblem.ServerError)
				return
			}

			page.Add(response)
		}

		page.PopulateLinks()
		sendPageResponse(r.Context(), w, page)
	}
}

func NewOpsByAccountHandler(lightHorizon services.LightHorizon) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var accountId string
		var paginate pagination
		var err error

		if accountId, paginate, err = accountRequestParams(w, r); err != nil {
			errorMsg := supportProblem.MakeInvalidFieldProblem("account_id", err)
			sendErrorResponse(r.Context(), w, *errorMsg)
			return
		}

		page := hal.Page{
			Cursor: strconv.FormatInt(paginate.Cursor, 10),
			Order:  string(paginate.Order),
			Limit:  uint64(paginate.Limit),
		}
		page.Init()
		page.FullURL = r.URL

		ops, err := lightHorizon.Operations.GetOperationsByAccount(ctx, paginate.Cursor, paginate.Limit, accountId)
		if err != nil {
			log.Error(err)
			sendErrorResponse(r.Context(), w, supportProblem.ServerError)
			return
		}

		for _, op := range ops {
			var response operations.Operation
			response, err = adapters.PopulateOperation(r, &op)
			if err != nil {
				log.Error(err)
				sendErrorResponse(r.Context(), w, supportProblem.ServerError)
				return
			}

			page.Add(response)
		}

		page.PopulateLinks()
		sendPageResponse(r.Context(), w, page)
	}
}
