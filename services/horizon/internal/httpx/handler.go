package httpx

import (
	"database/sql"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stellar/go/services/horizon/internal/actions"
	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/render"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/httpjson"
	"github.com/stellar/go/support/render/problem"
)

type objectAction interface {
	GetResource(
		w actions.HeaderWriter,
		r *http.Request,
	) (interface{}, error)
}

type ObjectActionHandler struct {
	Action objectAction
}

func (handler ObjectActionHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	switch render.Negotiate(r) {
	case render.MimeHal, render.MimeJSON:
		response, err := handler.Action.GetResource(w, r)
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

type pageActionHandler struct {
	action              pageAction
	streamable          bool
	streamHandler       sse.StreamHandler
	repeatableRead      bool
	ledgerState         *ledger.State
	responseAgeObserver *responseAgeMetric
}

func restPageHandler(ledgerState *ledger.State, action pageAction) pageActionHandler {
	return pageActionHandler{action: action, ledgerState: ledgerState}
}

// streamableStatePageHandler creates a streamable page handler than generates
// events within a REPEATABLE READ transaction.
func streamableStatePageHandler(
	ledgerState *ledger.State,
	action pageAction,
	streamHandler sse.StreamHandler,
) pageActionHandler {
	return pageActionHandler{
		action:         action,
		ledgerState:    ledgerState,
		streamable:     true,
		streamHandler:  streamHandler,
		repeatableRead: true,
	}
}

// streamableStatePageHandler creates a streamable page handler than generates
// events without starting a REPEATABLE READ transaction.
func streamableHistoryPageHandler(
	ledgerState *ledger.State,
	responseAgeObserver responseAgeMetric,
	action pageAction,
	streamHandler sse.StreamHandler,
) pageActionHandler {
	return pageActionHandler{
		action:              action,
		ledgerState:         ledgerState,
		streamable:          true,
		streamHandler:       streamHandler,
		repeatableRead:      false,
		responseAgeObserver: &responseAgeObserver,
	}
}

func (handler pageActionHandler) renderPage(w http.ResponseWriter, r *http.Request) {
	records, err := handler.action.GetResourcePage(w, r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}
	if handler.responseAgeObserver != nil {
		handler.responseAgeObserver.ObserveLedgerAgePage(r, records)
	}

	page, err := buildPage(handler.ledgerState, r, records)
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

type responseAgeMetric struct {
	ledgerState        *ledger.State
	ledgerAgeHistogram *prometheus.HistogramVec
}

func (m responseAgeMetric) ObserveLedgerAge(r *http.Request, ledger int) {
	latestLedger := int(m.ledgerState.CurrentStatus().ExpHistoryLatest)
	var ledgersAgo float64

	if ledger < latestLedger {
		ledgersAgo = float64(latestLedger) - float64(ledger)
	}

	m.ledgerAgeHistogram.With(prometheus.Labels{
		"route":     db.Route(r.Context()),
		"streaming": strconv.FormatBool(render.Negotiate(r) == render.MimeEventStream),
	}).Observe(ledgersAgo)
}

func (m responseAgeMetric) ObserveLedgerAgePage(r *http.Request, records []hal.Pageable) {
	latestLedger := int(m.ledgerState.CurrentStatus().ExpHistoryLatest)
	route := db.Route(r.Context())
	for _, record := range records {
		var ledgersAgo float64
		ledger, err := ledgerFromPagingToken(record.PagingToken())
		if err != nil {
			log.Warnf("invalid paging token %v", record.PagingToken())
		}

		if ledger < latestLedger {
			ledgersAgo = float64(latestLedger) - float64(ledger)
		}
		m.ledgerAgeHistogram.With(prometheus.Labels{
			"route":     route,
			"streaming": strconv.FormatBool(render.Negotiate(r) == render.MimeEventStream),
		}).Observe(ledgersAgo)
	}
}

func ledgerFromPagingToken(token string) (int, error) {
	if strings.Contains(token, "-") {
		token = strings.Split(token, "-")[0]
	}

	cursorInt, err := strconv.Atoi(token)
	if err != nil {
		return 0, err
	}
	tid := toid.Parse(int64(cursorInt))
	return int(tid.LedgerSequence), nil
}

func (handler pageActionHandler) renderStream(w http.ResponseWriter, r *http.Request) {
	// Use pq to Get SSE limit.
	pq, err := actions.GetPageQuery(handler.ledgerState, r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}

	var generateEvents sse.GenerateEventsFunc = func() ([]sse.Event, error) {
		records, err := handler.action.GetResourcePage(w, r)
		if err != nil {
			return nil, err
		}

		if handler.responseAgeObserver != nil {
			handler.responseAgeObserver.ObserveLedgerAgePage(r, records)
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

func buildPage(ledgerState *ledger.State, r *http.Request, records []hal.Pageable) (hal.Page, error) {
	// Always DisableCursorValidation - we can assume it's valid since the
	// validation is done in GetResourcePage.
	pageQuery, err := actions.GetPageQuery(ledgerState, r, actions.DisableCursorValidation)
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
