package horizon

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/stellar/go/services/horizon/internal/actions"
	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/hchi"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/render"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/httpjson"
	"github.com/stellar/go/support/render/problem"
)

// Fields of this struct are exported for json marshaling/unmarshaling in
// support/render/hal package.
type indexActionQueryParams struct {
	AccountID        string
	LedgerID         int32
	PagingParams     db2.PageQuery
	IncludeFailedTxs bool
	Signer           string
}

// Fields of this struct are exported for json marshaling/unmarshaling in
// support/render/hal package.
type showActionQueryParams struct {
	AccountID string
	TxHash    string
}

// streamFunc represents the signature of the function that handles requests
// with stream mode turned on using server-sent events.
type streamFunc func(context.Context, *sse.Stream, *indexActionQueryParams) error

// streamableEndpointHandler handles endpoints that have the stream mode
// available. It inspects the Accept header to determine which function to be
// executed. If it's "application/hal+json" or "application/json", then jfn
// will be executed with params. If it's "text/event-stream", then either sfn
// or jfn will be executed with the streamHandler with params.
func (we *web) streamableEndpointHandler(jfn interface{}, streamSingleObjectEnabled bool, sfn streamFunc, params interface{}) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		switch render.Negotiate(r) {
		case render.MimeHal, render.MimeJSON:
			if jfn == nil {
				problem.Render(ctx, w, hProblem.NotAcceptable)
				return
			}
			h, err := hal.Handler(jfn, params)
			if err != nil {
				panic(err)
			}

			h.ServeHTTP(w, r)
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
func (we *web) streamHandler(jfn interface{}, sfn streamFunc, params interface{}) http.HandlerFunc {
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
				err := sfn(ctx, stream, params.(*indexActionQueryParams))
				if err != nil {
					stream.Err(err)
					return
				}
			} else if jfn != nil {
				data, ok, err := hal.ExecuteFunc(ctx, jfn, params)
				if err != nil {
					if !ok {
						panic(err)
					}
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
			var closedLock sync.Mutex
			var closed bool
			go func() {
				for {
					time.Sleep(we.sseUpdateFrequency)

					closedLock.Lock()
					tmpClosed := closed
					closedLock.Unlock()
					if tmpClosed {
						return
					}

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
				closedLock.Lock()
				closed = true
				closedLock.Unlock()
			case <-we.appCtx.Done():
			}

			stream.Done()
			return
		}
	})
}

// streamShowActionHandler gets the showAction query params from the request
// and pass it on to streamableEndpointHandler.
func (we *web) streamShowActionHandler(jfn interface{}, requireAccountID bool) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		param, err := getShowActionQueryParams(r, requireAccountID)
		if err != nil {
			problem.Render(ctx, w, err)
			return
		}

		we.streamableEndpointHandler(jfn, true, nil, param).ServeHTTP(w, r)
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

	if _, err = strkey.Decode(strkey.VersionByteAccountID, val); err != nil {
		// TODO: add errInvalidValue
		return "", problem.MakeInvalidFieldProblem(key, errors.New("invalid address"))
	}

	return val, nil
}

// getShowActionQueryParams gets the available query params for all non-indexable endpoints.
func getShowActionQueryParams(r *http.Request, requireAccountID bool) (*showActionQueryParams, error) {
	txHash, err := actions.GetTransactionID(r, "tx_id")
	if err != nil {
		return nil, errors.Wrap(err, "getting tx id")
	}

	addr, err := getAccountID(r, "account_id", requireAccountID)
	if err != nil {
		return nil, errors.Wrap(err, "getting account id")
	}

	return &showActionQueryParams{
		AccountID: addr,
		TxHash:    txHash,
	}, nil
}

type objectAction interface {
	GetResource(
		w actions.HeaderWriter,
		r *http.Request,
	) (interface{}, error)
}

type objectActionHandler struct {
	action objectAction
}

func (handler objectActionHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	switch render.Negotiate(r) {
	case render.MimeHal, render.MimeJSON:
		response, err := handler.action.GetResource(w, r)
		if err != nil {
			problem.Render(r.Context(), w, err)
			return
		}

		httpjson.Render(
			w,
			response,
			httpjson.HALJSON,
		)
		return
	}

	problem.Render(r.Context(), w, hProblem.NotAcceptable)
}

const defaultObjectStreamLimit = 10

type streamableObjectAction interface {
	GetResource(
		w actions.HeaderWriter,
		r *http.Request,
	) (actions.StreamableObjectResponse, error)
}

type streamableObjectActionHandler struct {
	action        streamableObjectAction
	streamHandler sse.StreamHandler
	limit         int
}

func (handler streamableObjectActionHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	switch render.Negotiate(r) {
	case render.MimeHal, render.MimeJSON:
		response, err := handler.action.GetResource(w, r)
		if err != nil {
			problem.Render(r.Context(), w, err)
			return
		}

		httpjson.Render(
			w,
			response,
			httpjson.HALJSON,
		)
		return
	case render.MimeEventStream:
		handler.renderStream(w, r)
		return
	}

	problem.Render(r.Context(), w, hProblem.NotAcceptable)
}

func repeatableReadStream(
	r *http.Request,
	generateEvents sse.GenerateEventsFunc,
) sse.GenerateEventsFunc {
	var session db.SessionInterface
	if val := r.Context().Value(&horizonContext.SessionContextKey); val != nil {
		session = val.(db.SessionInterface)
	}

	return func() ([]sse.Event, error) {
		if session != nil {
			err := session.BeginTx(&sql.TxOptions{
				Isolation: sql.LevelRepeatableRead,
				ReadOnly:  true,
			})
			if err != nil {
				return nil, errors.Wrap(err, "Error starting repeatable read transaction")
			}
			defer session.Rollback()
		}

		return generateEvents()
	}
}

func (handler streamableObjectActionHandler) renderStream(
	w http.ResponseWriter,
	r *http.Request,
) {
	var lastResponse actions.StreamableObjectResponse
	limit := handler.limit
	if limit == 0 {
		limit = defaultObjectStreamLimit
	}

	handler.streamHandler.ServeStream(
		w,
		r,
		limit,
		repeatableReadStream(r, func() ([]sse.Event, error) {
			response, err := handler.action.GetResource(w, r)
			if err != nil {
				return nil, err
			}

			if lastResponse == nil || !lastResponse.Equals(response) {
				lastResponse = response
				return []sse.Event{{Data: response}}, nil
			}
			return []sse.Event{}, nil
		}),
	)
}

type pageAction interface {
	GetResourcePage(w actions.HeaderWriter, r *http.Request) ([]hal.Pageable, error)
}

type pageBuilder interface {
	BuildPage(r *http.Request, records []hal.Pageable) (interface{}, error)
}

type pageActionHandler struct {
	action            pageAction
	streamable        bool
	streamHandler     sse.StreamHandler
	repeatableRead    bool
	customPageBuilder pageBuilder
}

func restPageHandler(action pageAction) pageActionHandler {
	return pageActionHandler{action: action}
}

type customBuiltPageAction interface {
	pageAction
	pageBuilder
}

func restCustomBuiltPageHandler(action customBuiltPageAction) pageActionHandler {
	return pageActionHandler{
		action:            action,
		customPageBuilder: action,
	}
}

// streamableStatePageHandler creates a streamable page handler than generates
// events within a REPEATABLE READ transaction.
func streamableStatePageHandler(
	action pageAction,
	streamHandler sse.StreamHandler,
) pageActionHandler {
	return pageActionHandler{
		action:         action,
		streamable:     true,
		streamHandler:  streamHandler,
		repeatableRead: true,
	}
}

// streamableStatePageHandler creates a streamable page handler than generates
// events without starting a REPEATABLE READ transaction.
func streamableHistoryPageHandler(
	action pageAction,
	streamHandler sse.StreamHandler,
) pageActionHandler {
	return pageActionHandler{
		action:         action,
		streamable:     true,
		streamHandler:  streamHandler,
		repeatableRead: false,
	}
}

func (handler pageActionHandler) renderPage(w http.ResponseWriter, r *http.Request) {
	records, err := handler.action.GetResourcePage(w, r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}

	var page interface{}

	if handler.customPageBuilder != nil {
		page, err = handler.customPageBuilder.BuildPage(r, records)
	} else {
		page, err = buildPage(r, records)
	}

	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}

	httpjson.Render(
		w,
		page,
		httpjson.HALJSON,
	)
}

func (handler pageActionHandler) renderStream(w http.ResponseWriter, r *http.Request) {
	// Use pq to Get SSE limit.
	pq, err := actions.GetPageQuery(r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}

	var generateEvents sse.GenerateEventsFunc = func() ([]sse.Event, error) {
		records, err := handler.action.GetResourcePage(w, r)
		if err != nil {
			return nil, err
		}

		events := make([]sse.Event, 0, len(records))
		for _, record := range records {
			events = append(events, sse.Event{ID: record.PagingToken(), Data: record})
		}

		if len(events) > 0 {
			// Update the cursor for the next call to GetObject, getCursor
			// will use Last-Event-ID if present. This feels kind of hacky,
			// but otherwise, we'll have to edit r.URL, which is also a
			// hack.
			r.Header.Set("Last-Event-ID", events[len(events)-1].ID)
		} else if len(r.Header.Get("Last-Event-ID")) == 0 {
			// If there are no records and Last-Event-ID has not been set,
			// use the cursor from pq as the Last-Event-ID, otherwise, we'll
			// keep using `now` which will always resolve to the next
			// ledger.
			r.Header.Set("Last-Event-ID", pq.Cursor)
		}

		return events, nil
	}

	if handler.repeatableRead {
		generateEvents = repeatableReadStream(r, generateEvents)
	}

	handler.streamHandler.ServeStream(
		w,
		r,
		int(pq.Limit),
		generateEvents,
	)
}

func (handler pageActionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch render.Negotiate(r) {
	case render.MimeHal, render.MimeJSON:
		handler.renderPage(w, r)
		return
	case render.MimeEventStream:
		if handler.streamable {
			handler.renderStream(w, r)
			return
		}
	}

	problem.Render(r.Context(), w, hProblem.NotAcceptable)
}

func buildPage(r *http.Request, records []hal.Pageable) (hal.Page, error) {
	// Always DisableCursorValidation - we can assume it's valid since the
	// validation is done in GetResourcePage.
	pageQuery, err := actions.GetPageQuery(r, actions.DisableCursorValidation)
	if err != nil {
		return hal.Page{}, err
	}

	ctx := r.Context()

	page := hal.Page{
		Cursor: pageQuery.Cursor,
		Order:  pageQuery.Order,
		Limit:  pageQuery.Limit,
	}
	page.Init()

	for _, record := range records {
		page.Add(record)
	}

	page.FullURL = actions.FullURL(ctx)
	page.PopulateLinks()

	return page, nil
}

type rawAction interface {
	WriteRawResponse(w io.Writer, r *http.Request) error
}

func HandleRaw(action rawAction) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := action.WriteRawResponse(w, r); err != nil {
			problem.Render(r.Context(), w, err)
		}
	}
}

func WrapRaw(next http.Handler, action rawAction) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch render.Negotiate(r) {
		case render.MimeRaw:
			HandleRaw(action).ServeHTTP(w, r)
		default:
			next.ServeHTTP(w, r)
		}
	})
}
