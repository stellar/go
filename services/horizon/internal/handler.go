package horizon

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/hchi"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/render"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
)

// jsonResponderFunc represents the signature of the function that handles
// requests to which the server responds in json format.
type jsonResponderFunc func(context.Context, interface{}) (interface{}, error)

// streamFunc represents the signature of the function that handles requests
// with stream mode turned on using server-sent events.
type streamFunc func(context.Context, *sse.Stream) error

// singleObjectStreamFunc represents the signature of the function that handles
// requests with stream mode turned on using server-sent events. The difference
// between this function and streamFunc is that this one only loads an event. The
// server will not send an event out if the current event is same as the last one.
// Please see the implementation of streamHandler for more details.
type singleObjectStreamFunc func(context.Context, interface{}) (sse.Event, error)

// streamableEndpointHandler handles endpoints that have the stream mode
// available. It inspects the Accept header to determine which function to be
// executed. If it's "application/hal+json" or "application/json", then jfn
// will be executed. If it's "text/event-stream", then either sfn or sosfn will
// be executed with the streamHandler.
func (we *web) streamableEndpointHandler(jfn jsonResponderFunc, sfn streamFunc, sosfn singleObjectStreamFunc, params interface{}) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		contentType := render.Negotiate(r)
		switch contentType {
		case render.MimeHal, render.MimeJSON:
			if jfn == nil {
				problem.Render(ctx, w, hProblem.NotAcceptable)
				return
			}

			hal.Handler(jfn, params).ServeHTTP(w, r)
			return

		case render.MimeEventStream:
			if sfn == nil && sosfn == nil {
				problem.Render(ctx, w, hProblem.NotAcceptable)
				return
			}

			we.streamHandler(sfn, sosfn, params).ServeHTTP(w, r)
			return
		}

		problem.Render(ctx, w, hProblem.NotAcceptable)
	})
}

// streamHandler handles requests with stream mode turned on using server-sent
// events. It will execute one of the provided streaming functions. Note that
// we don't return an error if both sfn and sosfn are not nil. sfn will simply
// take precedence.
func (we *web) streamHandler(sfn streamFunc, sosfn singleObjectStreamFunc, params interface{}) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		stream := sse.NewStream(ctx, w)
		var oldHash [32]byte
		for {
			lastLedgerState := ledger.CurrentState()

			// Rate limit the request if it's a call to stream since it queries the DB every second. See
			// https://github.com/stellar/go/issues/715 for more details.
			rateLimiter := we.rateLimiter
			if rateLimiter != nil {
				limited, _, err := rateLimiter.RateLimiter.RateLimit(rateLimiter.VaryBy.Key(r), 1)
				if err != nil {
					stream.Err(errors.Wrap(err, "RateLimiter error"))
					return
				}
				if limited {
					stream.Err(sse.ErrRateLimited)
					return
				}
			}

			if sfn != nil {
				err := sfn(ctx, stream)
				if err != nil {
					stream.Err(err)
					return
				}
			} else if sosfn != nil {
				newEvent, err := sosfn(ctx, params)
				if err != nil {
					stream.Err(err)
					return
				}
				resource, err := json.Marshal(newEvent.Data)
				if err != nil {
					stream.Err(errors.Wrap(err, "unable to marshal next action resource"))
					return
				}

				nextHash := sha256.Sum256(resource)
				if !bytes.Equal(nextHash[:], oldHash[:]) {
					oldHash = nextHash
					stream.SetLimit(10)
					stream.Send(newEvent)
				}
			}

			// Manually send the preamble in case there are no data events in SSE to trigger a stream.Send call.
			// This method is called every iteration of the loop, but is protected by a sync.Once variable so it's
			// only executed once.
			stream.Init()

			if stream.IsDone() {
				return
			}

			// Make sure this is buffered channel of size 1. Otherwise, the go routine below
			// will never return if `newLedgers` channel is not read. From Effective Go:
			// > If the channel is unbuffered, the sender blocks until the receiver has received the value.
			newLedgers := make(chan bool, 1)
			go func() {
				for {
					time.Sleep(we.sseUpdateFrequency)
					currentLedgerState := ledger.CurrentState()
					if currentLedgerState.HistoryLatest >= lastLedgerState.HistoryLatest+1 {
						newLedgers <- true
						return
					}
				}
			}()

			select {
			case <-newLedgers:
				continue
			case <-ctx.Done():
			case <-we.appCtx.Done():
			}

			stream.Done()
			return
		}
	})
}

// accountHandler gets the account address from the request and pass it on to
// streamableEndpointHandler.
// Note that we cannot put this handler in the middleware stack because of
// Chi's routing mechanism. A request will have to reach the end of the route
// in order to have a valid route pattern in Chi.
func (we *web) accountHandler(jfn jsonResponderFunc, sosfn singleObjectStreamFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		addr, err := getAccountID(r, "account_id", true)
		if err != nil {
			problem.Render(ctx, w, err)
			return
		}

		we.streamableEndpointHandler(jfn, nil, sosfn, addr).ServeHTTP(w, r)
	})
}

func getAccountID(r *http.Request, key string, required bool) (string, error) {
	val, err := hchi.GetStringFromURL(r, key)
	if err != nil {
		return "", err
	}

	if val == "" && !required {
		return val, nil
	}

	_, err = strkey.Decode(strkey.VersionByteAccountID, val)
	if err != nil {
		// TODO: add errInvalidValue
		return "", problem.MakeInvalidFieldProblem(key, errors.New("invalid address"))
	}

	return val, nil
}

func (we *web) transactionHandler(jfn jsonResponderFunc, sfn streamFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		err := errorIfHistoryIsStale(we.isHistoryStale())
		if err != nil {
			problem.Render(ctx, w, err)
			return

		}

		params, err := loadTransactionParams(r, we.ingestFailedTx)
		if err != nil {
			problem.Render(ctx, w, err)
			return
		}

		err = validateCursorWithinHistory(params.PagingParams)
		if err != nil {
			problem.Render(ctx, w, err)
			return
		}

		we.streamableEndpointHandler(jfn, sfn, nil, params).ServeHTTP(w, r)
	})
}

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

func loadTransactionParams(r *http.Request, ingestFailedTransactions bool) (*actions.TransactionParams, error) {
	addr, err := getAccountID(r, "account_id", false)
	if err != nil {
		return nil, errors.Wrap(err, "getting account address")
	}

	lid, err := getInt32ParamFromURL(r, "ledger_id")
	if err != nil {
		return nil, errors.Wrap(err, "getting ledger id")
	}

	if addr != "" && lid != int32(0) {
		return nil, problem.BadRequest
	}

	pq, err := getPageQuery(r, false)
	if err != nil {
		return nil, errors.Wrap(err, "getting page query")
	}

	includeFailedTx, err := getBoolParamFromURL(r, "include_failed")
	if err != nil {
		return nil, errors.Wrap(err, "getting include_failed param")
	}
	if includeFailedTx == true && !ingestFailedTransactions {
		return nil, problem.MakeInvalidFieldProblem("include_failed",
			errors.New("`include_failed` parameter is unavailable when Horizon is not ingesting failed "+
				"transactions. Set `INGEST_FAILED_TRANSACTIONS=true` to start ingesting them."))
	}

	return &actions.TransactionParams{
		AccountFilter: addr,
		LedgerFilter:  lid,
		PagingParams:  pq,
		IncludeFailed: includeFailedTx,
	}, nil
}

func getInt32ParamFromURL(r *http.Request, key string) (int32, error) {
	val, err := hchi.GetStringFromURL(r, key)
	if err != nil {
		return 0, errors.Wrapf(err, "loading %s from URL", key)
	}

	asI64, err := strconv.ParseInt(val, 10, 32)
	// TODO: add errInvalidValue
	return int32(asI64), problem.MakeInvalidFieldProblem(key, errors.New("invalid int32 value"))
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
