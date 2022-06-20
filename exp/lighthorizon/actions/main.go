package actions

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/stellar/go/support/render/hal"
)

var (
	//go:embed static
	staticFiles embed.FS
)

type Pagination struct {
	Limit  int64
	Cursor int64
}

func sendPageResponse(w http.ResponseWriter, page hal.Page) {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(page)
	if err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func sendErrorResponse(w http.ResponseWriter, errorCode int) {
	w.WriteHeader(errorCode)
}

func RequestUnaryParam(r *http.Request, paramName string) (string, error) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return "", err
	}
	return query.Get(paramName), nil
}

func Paging(r *http.Request) (Pagination, error) {
	var cursorRequested, limitRequested string
	var err error
	paginate := Pagination{}

	if cursorRequested, err = RequestUnaryParam(r, "cursor"); err != nil {
		return Pagination{}, err
	}

	if limitRequested, err = RequestUnaryParam(r, "limit"); err != nil {
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
	return paginate, nil
}
