package horizon

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"time"

	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/hchi"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/render"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/render/sse"
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
				err := sfn(ctx, stream, params)
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
