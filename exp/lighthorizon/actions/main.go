package actions

import (
	"embed"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/hal"
)

var (
	//go:embed static
	staticFiles embed.FS
)

type Order string
type ErrorMessage string

const (
	OrderAsc  Order = "asc"
	OrderDesc Order = "desc"
)

const (
	ServerError             ErrorMessage = "Error: A problem occurred on the server while processing request"
	InvalidPagingParameters ErrorMessage = "Error: Invalid paging parameters"
)

type Pagination struct {
	Limit  int64
	Cursor int64
	Order
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
		http.Error(w, string(ServerError), errorCode)
	}
}

func RequestUnaryParam(r *http.Request, paramName string) (string, error) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return "", err
	}
	return query.Get(paramName), nil
}

func Paging(r *http.Request) (Pagination, error) {
	var cursorRequested, limitRequested, orderRequested string
	var err error
	paginate := Pagination{
		Order: OrderAsc,
	}

	if cursorRequested, err = RequestUnaryParam(r, "cursor"); err != nil {
		return Pagination{}, err
	}

	if limitRequested, err = RequestUnaryParam(r, "limit"); err != nil {
		return Pagination{}, err
	}

	if orderRequested, err = RequestUnaryParam(r, "order"); err != nil {
		return Pagination{}, err
	}

	if cursorRequested != "" {
		paginate.Cursor, err = strconv.ParseInt(cursorRequested, 10, 64)
		if err != nil {
			return Pagination{}, err
		}
	}

	if limitRequested != "" {
		paginate.Limit, err = strconv.ParseInt(limitRequested, 10, 64)
		if err != nil {
			return Pagination{}, err
		}
	}

	if orderRequested != "" && orderRequested == string(OrderDesc) {
		paginate.Order = OrderDesc
	}

	return paginate, nil
}
