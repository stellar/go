package horizon

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/hchi"
	"github.com/stellar/go/services/horizon/internal/ledger"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/problem"
)

func getCursor(r *http.Request) (string, error) {
	cursor, err := hchi.GetStringFromURL(r, actions.ParamCursor)
	if err != nil {
		return "", errors.Wrap(err, "loading cursor from URL")
	}

	if cursor == "now" {
		cursor = toid.AfterLedger(ledger.CurrentState().HistoryLatest).String()
	}

	if lastEventID := r.Header.Get("Last-Event-ID"); lastEventID != "" {
		cursor = lastEventID
	}

	if cursor == "" {
		return "", nil
	}

	curInt64, err := strconv.ParseInt(cursor, 10, 64)
	if err != nil {
		return "", problem.MakeInvalidFieldProblem(actions.ParamCursor, errors.New("invalid int64 value"))
	}
	if curInt64 < 0 {
		return "", problem.MakeInvalidFieldProblem(actions.ParamCursor, errors.New(fmt.Sprintf("cursor %d is a negative number: ", curInt64)))
	}

	return cursor, nil
}

func getOrder(r *http.Request) (string, error) {
	order, err := hchi.GetStringFromURL(r, actions.ParamOrder)
	if err != nil {
		return "", errors.Wrap(err, "getting param order from URL")
	}

	// Set order
	if order == "" {
		order = db2.OrderAscending
	}
	if order != db2.OrderAscending && order != db2.OrderDescending {
		return "", db2.ErrInvalidOrder
	}

	return order, nil
}

func getLimit(r *http.Request, defaultSize, maxSize uint64) (uint64, error) {
	limit, err := hchi.GetStringFromURL(r, actions.ParamLimit)
	if err != nil {
		return 0, errors.Wrap(err, "loading param limit from URL")
	}
	if limit == "" {
		return defaultSize, nil
	}

	limitInt64, err := strconv.ParseInt(limit, 10, 64)
	if err != nil {
		return 0, problem.MakeInvalidFieldProblem(actions.ParamLimit, errors.New("invalid int64 value"))
	}
	if limitInt64 <= 0 {
		return 0, problem.MakeInvalidFieldProblem(actions.ParamLimit, errors.New(fmt.Sprintf("limit %d is a non-positive number: ", limitInt64)))
	}
	if limitInt64 > int64(maxSize) {
		return 0, problem.MakeInvalidFieldProblem(actions.ParamLimit, errors.New(fmt.Sprintf("limit %d is greater than limit max of %d", limitInt64, maxSize)))
	}

	return uint64(limitInt64), nil
}

func getPageQuery(r *http.Request, disableCursorValidation bool) (db2.PageQuery, error) {
	cursor, err := getCursor(r)
	if err != nil {
		return db2.PageQuery{}, errors.Wrap(err, "getting param cursor")
	}

	order, err := getOrder(r)
	if err != nil {
		return db2.PageQuery{}, errors.Wrap(err, "getting param order")
	}

	limit, err := getLimit(r, db2.DefaultPageSize, db2.MaxPageSize)
	if err != nil {
		return db2.PageQuery{}, errors.Wrap(err, "getting param limit")
	}

	pq := db2.PageQuery{
		Cursor: cursor,
		Order:  order,
		Limit:  limit,
	}
	if !disableCursorValidation {
		_, _, err = pq.CursorInt64Pair(db2.DefaultPairSep)
		if err != nil {
			return db2.PageQuery{}, problem.MakeInvalidFieldProblem(actions.ParamCursor, db2.ErrInvalidCursor)
		}
	}

	return pq, nil
}

func getInt32ParamFromURL(r *http.Request, key string) (int32, error) {
	val, err := hchi.GetStringFromURL(r, key)
	if err != nil {
		return 0, errors.Wrapf(err, "loading %s from URL", key)
	}
	if val == "" {
		return int32(0), nil
	}

	asI64, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		return int32(0), problem.MakeInvalidFieldProblem(key, errors.New("invalid int32 value"))
	}

	return int32(asI64), nil
}

func getBoolParamFromURL(r *http.Request, key string) (bool, error) {
	asStr := r.URL.Query().Get(key)
	if asStr == "true" {
		return true, nil
	}
	if asStr == "false" || asStr == "" {
		return false, nil
	}

	return false, problem.MakeInvalidFieldProblem(key, errors.New("invalid bool value"))
}

func validateCursorWithinHistory(pq db2.PageQuery) error {
	// an ascending query should never return a gone response:  An ascending query
	// prior to known history should return results at the beginning of history,
	// and an ascending query beyond the end of history should not error out but
	// rather return an empty page (allowing code that tracks the procession of
	// some resource more easily).
	if pq.Order != "desc" {
		return nil
	}

	var (
		cursor int64
		err    error
	)
	if strings.Contains(pq.Cursor, "-") {
		cursor, _, err = pq.CursorInt64Pair("-")
	} else {
		cursor, err = pq.CursorInt64()
	}
	if err != nil {
		return problem.MakeInvalidFieldProblem(actions.ParamCursor, errors.New("invalid value"))
	}

	elder := toid.New(ledger.CurrentState().HistoryElder, 0, 0)
	if cursor <= elder.ToInt64() {
		return &hProblem.BeforeHistory
	}

	return nil
}
