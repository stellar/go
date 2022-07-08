package actions

import (
	"embed"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/gorilla/schema"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/hal"
)

var (
	//go:embed static
	staticFiles embed.FS
)

type order string
type errorMessage string

const (
	orderAsc  order = "asc"
	orderDesc order = "desc"
)

const (
	serverError             errorMessage = "Error: A problem occurred on the server while processing request"
	invalidPagingParameters errorMessage = "Error: Invalid paging parameters"
)

type pagination struct {
	Limit  int64
	Cursor int64
	Order  order
}

func sendPageResponse(w http.ResponseWriter, page hal.Page) {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(page)
	if err != nil {
		log.Error(err)
		sendErrorResponse(w, http.StatusInternalServerError, "")
	}
}

func sendErrorResponse(w http.ResponseWriter, errorCode int, errorMsg string) {
	if errorMsg != "" {
		http.Error(w, errorMsg, errorCode)
	} else {
		http.Error(w, string(serverError), errorCode)
	}
}

func requestUnaryParam(r *http.Request, paramName string) (string, error) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return "", err
	}
	return query.Get(paramName), nil
}

func paging(r *http.Request) (pagination, error) {
	paginate := pagination{
		Order: orderAsc,
	}

	if cursorRequested, err := requestUnaryParam(r, "cursor"); err != nil {
		return pagination{}, err
	} else if cursorRequested != "" {
		paginate.Cursor, err = strconv.ParseInt(cursorRequested, 10, 64)
		if err != nil {
			return pagination{}, err
		}
	}

	if limitRequested, err := requestUnaryParam(r, "limit"); err != nil {
		return pagination{}, err
	} else if limitRequested != "" {
		paginate.Limit, err = strconv.ParseInt(limitRequested, 10, 64)
		if err != nil {
			return pagination{}, err
		}
	}

	if orderRequested, err := requestUnaryParam(r, "order"); err != nil {
		return pagination{}, err
	} else if orderRequested != "" && orderRequested == string(orderDesc) {
		paginate.Order = orderDesc
	}

	return paginate, nil
}

var decoder = schema.NewDecoder()

func getURLParam(r *http.Request, key string) (string, bool) {
	rctx := chi.RouteContext(r.Context())

	if rctx == nil {
		return "", false
	}

	if len(rctx.URLParams.Keys) != len(rctx.URLParams.Values) {
		return "", false
	}

	for k := len(rctx.URLParams.Keys) - 1; k >= 0; k-- {
		if rctx.URLParams.Keys[k] == key {
			return rctx.URLParams.Values[k], true
		}
	}

	return "", false
}
