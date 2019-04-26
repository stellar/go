package horizon

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
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
type streamFunc func(context.Context, *sse.Stream, interface{}) error

// streamableEndpointHandler handles endpoints that have the stream mode
// available. It inspects the Accept header to determine which function to be
// executed. If it's "application/hal+json" or "application/json", then jfn
// will be executed with params. If it's "text/event-stream", then either sfn
// or jfn will be executed with the streamHandler with params.
func (we *web) streamableEndpointHandler(jfn jsonResponderFunc, streamSingleObjectEnabled bool, sfn streamFunc, params interface{}) http.HandlerFunc {
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
			if sfn == nil && !streamSingleObjectEnabled {
				problem.Render(ctx, w, hProblem.NotAcceptable)
				return
			}

			we.streamHandler(jfn, sfn, params).ServeHTTP(w, r)
			return
		}

		problem.Render(ctx, w, hProblem.NotAcceptable)
	})
}

// streamHandler handles requests with stream mode turned on using server-sent
// events. It will execute one of the provided streaming functions with the
// provided params.
// Note that we don't return an error if both jfn and sfn are not nil. sfn will
// simply take precedence.
func (we *web) streamHandler(jfn jsonResponderFunc, sfn streamFunc, params interface{}) http.HandlerFunc {
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
				err := sfn(ctx, stream, params)
				if err != nil {
					stream.Err(err)
					return
				}
			} else if jfn != nil {
				data, err := jfn(ctx, params)
				if err != nil {
					stream.Err(err)
					return
				}
				resource, err := json.Marshal(data)
				if err != nil {
					stream.Err(errors.Wrap(err, "unable to marshal next action resource"))
					return
				}

				nextHash := sha256.Sum256(resource)
				if !bytes.Equal(nextHash[:], oldHash[:]) {
					oldHash = nextHash
					stream.SetLimit(10)
					stream.Send(sse.Event{Data: data})
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
func (we *web) accountHandler(jfn jsonResponderFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		addr, err := getAccountID(r, "account_id", true)
		if err != nil {
			problem.Render(ctx, w, err)
			return
		}

		we.streamableEndpointHandler(jfn, true, nil, addr).ServeHTTP(w, r)
	})
}

// getAccountID retrieves the account id by the provided key. The key is
// usually "account_id", "source_account", and "destination_account". The
// function would return an error if the account id is empty and the required
// flag is true.
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

// transactionHandler checks whether the history is stale, gets the required
// params for transaction endpoints from the URL, validates the cursor is within
// history, and finally pass the transaction params to the more general purpose
// streamableEndpointHandler.
func (we *web) transactionHandler(jfn jsonResponderFunc, sfn streamFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		params, err := getTransactionQueryParams(r, we.ingestFailedTx)
		if err != nil {
			problem.Render(ctx, w, err)
			return
		}

		err = validateCursorWithinHistory(params.PagingParams)
		if err != nil {
			problem.Render(ctx, w, err)
			return
		}

		we.streamableEndpointHandler(jfn, false, sfn, params).ServeHTTP(w, r)
	})
}

// getTransactionQueryParams gets the available query params for transaction endpoints.
func getTransactionQueryParams(r *http.Request, ingestFailedTransactions bool) (*actions.TransactionParams, error) {
	addr, err := getAccountID(r, "account_id", false)
	if err != nil {
		return nil, errors.Wrap(err, "getting account address")
	}

	lid, err := getInt32ParamFromURL(r, "ledger_id")
	if err != nil {
		return nil, errors.Wrap(err, "getting ledger id")
	}

	// account_id and ledger_id are mutually excludesive.
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

// validateCursorWithinHistory first checks whether the cursor in the page
// param is valid basesd on the order then verifies whether the cursor is
// within history.
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
	// cursor from effect streaming endpoint may contain the DefaultPairSep.
	if strings.Contains(pq.Cursor, db2.DefaultPairSep) {
		cursor, _, err = pq.CursorInt64Pair(db2.DefaultPairSep)
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
