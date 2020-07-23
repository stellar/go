package horizon

import (
	"compress/flate"
	"context"
	"database/sql"

	"github.com/pkg/errors"

	"github.com/stellar/go/services/horizon/internal/txsub"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/go-chi/chi"
	chimiddleware "github.com/go-chi/chi/middleware"
	"github.com/rcrowley/go-metrics"
	"github.com/rs/cors"
	"github.com/sebest/xff"

	"github.com/stellar/throttled"

	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/paths"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/services/horizon/internal/txsub/sequence"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/problem"
)

const (
	LRUCacheSize            = 50000
	maxAssetsForPathFinding = 15
)

// Web contains the http server related fields for horizon: the router,
// rate limiter, etc.
type web struct {
	appCtx             context.Context
	router             *chi.Mux
	internalRouter     *chi.Mux
	rateLimiter        *throttled.HTTPRateLimiter
	sseUpdateFrequency time.Duration
	staleThreshold     uint

	historyQ *history.Q

	requestTimer metrics.Timer
	failureMeter metrics.Meter
	successMeter metrics.Meter
}

func init() {
	// register problems
	problem.RegisterError(sql.ErrNoRows, problem.NotFound)
	problem.RegisterError(sequence.ErrNoMoreRoom, hProblem.ServerOverCapacity)
	problem.RegisterError(db2.ErrInvalidCursor, problem.BadRequest)
	problem.RegisterError(db2.ErrInvalidLimit, problem.BadRequest)
	problem.RegisterError(db2.ErrInvalidOrder, problem.BadRequest)
	problem.RegisterError(sse.ErrRateLimited, hProblem.RateLimitExceeded)
	problem.RegisterError(context.DeadlineExceeded, hProblem.Timeout)
	problem.RegisterError(context.Canceled, hProblem.ServiceUnavailable)
	problem.RegisterError(db.ErrCancelled, hProblem.ServiceUnavailable)
}

// mustInitWeb installed a new Web instance onto the provided app object.
func mustInitWeb(ctx context.Context, hq *history.Q, updateFreq time.Duration, threshold uint) *web {
	if hq == nil {
		log.Fatal("missing history DB for installing the web instance")
	}

	return &web{
		appCtx:             ctx,
		router:             chi.NewRouter(),
		internalRouter:     chi.NewRouter(),
		historyQ:           hq,
		sseUpdateFrequency: updateFreq,
		staleThreshold:     threshold,
		requestTimer:       metrics.NewTimer(),
		failureMeter:       metrics.NewMeter(),
		successMeter:       metrics.NewMeter(),
	}
}

// getAccountInfo returns the information about an account based on the provided param.
func (w *web) getAccountInfo(ctx context.Context, qp *showActionQueryParams) (interface{}, error) {
	// Use AppFromContext to prevent larger refactoring of actions code. Will
	// be removed once this endpoint is migrated to use new actions design.
	horizonSession, err := w.horizonSession(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "getting horizon db session")
	}

	err = horizonSession.BeginTx(&sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "error starting transaction")
	}

	defer horizonSession.Rollback()
	historyQ := &history.Q{horizonSession}

	return actions.AccountInfo(ctx, historyQ, qp.AccountID)
}

// mustInstallMiddlewares installs the middleware stack used for horizon onto the
// provided app.
// Note that a request will go through the middlewares from top to bottom.
func (w *web) mustInstallMiddlewares(app *App, connTimeout time.Duration) {
	if w == nil {
		log.Fatal("missing web instance for installing middlewares")
	}

	r := w.router
	r.Use(chimiddleware.StripSlashes)

	//TODO: remove this middleware
	r.Use(appContextMiddleware(app))

	r.Use(requestCacheHeadersMiddleware)
	r.Use(chimiddleware.RequestID)
	r.Use(contextMiddleware)
	r.Use(xff.Handler)
	r.Use(loggerMiddleware)
	r.Use(timeoutMiddleware(connTimeout))
	r.Use(requestMetricsMiddleware)
	r.Use(recoverMiddleware)
	r.Use(chimiddleware.Compress(flate.DefaultCompression, "application/hal+json"))

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedHeaders: []string{"*"},
		ExposedHeaders: []string{"Date"},
	})
	r.Use(c.Handler)

	r.Use(w.RateLimitMiddleware)

	// Internal middlewares
	w.internalRouter.Use(chimiddleware.StripSlashes)
	w.internalRouter.Use(appContextMiddleware(app))
	w.internalRouter.Use(chimiddleware.RequestID)
	w.internalRouter.Use(loggerMiddleware)
}

type historyLedgerSourceFactory struct {
	updateFrequency time.Duration
}

func (f historyLedgerSourceFactory) Get() ledger.Source {
	return ledger.NewHistoryDBSource(f.updateFrequency)
}

// mustInstallActions installs the routing configuration of horizon onto the
// provided app.  All route registration should be implemented here.
func (w *web) mustInstallActions(config Config,
	pathFinder paths.Finder,
	session *db.Session,
	submitter *txsub.System,
	registry metrics.Registry,
	coreGetter actions.CoreSettingsGetter,
	horizonVersion string) {
	if w == nil {
		log.Fatal("missing web instance for installing web actions")
	}

	stateMiddleware := StateMiddleware{
		HorizonSession: session,
	}

	r := w.router

	r.Method(http.MethodGet, "/", objectActionHandler{action: actions.GetRootHandler{
		CoreSettingsGetter: coreGetter,
		NetworkPassphrase:  config.NetworkPassphrase,
		FriendbotURL:       config.FriendbotURL,
		HorizonVersion:     horizonVersion,
	}})

	streamHandler := sse.StreamHandler{
		RateLimiter:         w.rateLimiter,
		LedgerSourceFactory: historyLedgerSourceFactory{updateFrequency: w.sseUpdateFrequency},
	}

	historyMiddleware := NewHistoryMiddleware(int32(w.staleThreshold), session)

	// State endpoints behind stateMiddleware
	r.Group(func(r chi.Router) {
		r.Use(stateMiddleware.Wrap)

		r.Route("/accounts", func(r chi.Router) {
			r.Method(http.MethodGet, "/", restPageHandler(actions.GetAccountsHandler{}))
			r.Route("/{account_id}", func(r chi.Router) {
				r.Get("/", w.streamShowActionHandler(w.getAccountInfo, true))
				accountData := actions.GetAccountDataHandler{}
				r.Method(http.MethodGet, "/data/{key}", WrapRaw(
					streamableObjectActionHandler{streamHandler: streamHandler, action: accountData},
					accountData,
				))
				r.Method(http.MethodGet, "/offers", streamableStatePageHandler(actions.GetAccountOffersHandler{}, streamHandler))
			})
		})

		r.Route("/offers", func(r chi.Router) {
			r.Method(http.MethodGet, "/", restPageHandler(actions.GetOffersHandler{}))
			r.Method(http.MethodGet, "/{id}", objectActionHandler{actions.GetOfferByID{}})
		})

		r.Method(http.MethodGet, "/assets", restPageHandler(actions.AssetStatsHandler{}))

		findPaths := restCustomBuiltPageHandler(actions.FindPathsHandler{
			StaleThreshold:       config.StaleThreshold,
			SetLastLedgerHeader:  true,
			MaxPathLength:        config.MaxPathLength,
			MaxAssetsParamLength: maxAssetsForPathFinding,
			PathFinder:           pathFinder,
		})
		findFixedPaths := restCustomBuiltPageHandler(actions.FindFixedPathsHandler{
			MaxPathLength:        config.MaxPathLength,
			SetLastLedgerHeader:  true,
			MaxAssetsParamLength: maxAssetsForPathFinding,
			PathFinder:           pathFinder,
		})

		r.Method(http.MethodGet, "/paths", findPaths)
		r.Method(http.MethodGet, "/paths/strict-receive", findPaths)
		r.Method(http.MethodGet, "/paths/strict-send", findFixedPaths)

		r.Method(
			http.MethodGet,
			"/order_book",
			streamableObjectActionHandler{
				streamHandler: streamHandler,
				action:        actions.GetOrderbookHandler{},
			},
		)
	})

	// account actions - /accounts/{account_id} has been created above so we
	// need to use absolute routes here. Make sure we use regexp check here for
	// emptiness. Without it, requesting `/accounts//payments` return all payments!
	r.Group(func(r chi.Router) {
		r.Use(historyMiddleware)
		r.Method(http.MethodGet, "/accounts/{account_id:\\w+}/effects", streamableHistoryPageHandler(actions.GetEffectsHandler{}, streamHandler))
		r.Method(http.MethodGet, "/accounts/{account_id:\\w+}/operations", streamableHistoryPageHandler(actions.GetOperationsHandler{
			OnlyPayments: false,
		}, streamHandler))
		r.Method(http.MethodGet, "/accounts/{account_id:\\w+}/payments", streamableHistoryPageHandler(actions.GetOperationsHandler{
			OnlyPayments: true,
		}, streamHandler))
		r.Method(http.MethodGet, "/accounts/{account_id:\\w+}/trades", streamableHistoryPageHandler(actions.GetTradesHandler{}, streamHandler))
		r.Method(http.MethodGet, "/accounts/{account_id:\\w+}/transactions", streamableHistoryPageHandler(actions.GetTransactionsHandler{}, streamHandler))
	})
	// ledger actions
	r.Route("/ledgers", func(r chi.Router) {
		r.Use(historyMiddleware)
		r.Method(http.MethodGet, "/", streamableHistoryPageHandler(actions.GetLedgersHandler{}, streamHandler))
		r.Route("/{ledger_id}", func(r chi.Router) {
			r.Method(http.MethodGet, "/", objectActionHandler{actions.GetLedgerByIDHandler{}})
			r.Method(http.MethodGet, "/transactions", streamableHistoryPageHandler(actions.GetTransactionsHandler{}, streamHandler))
			r.Group(func(r chi.Router) {
				r.Method(http.MethodGet, "/effects", streamableHistoryPageHandler(actions.GetEffectsHandler{}, streamHandler))
				r.Method(http.MethodGet, "/operations", streamableHistoryPageHandler(actions.GetOperationsHandler{
					OnlyPayments: false,
				}, streamHandler))
				r.Method(http.MethodGet, "/payments", streamableHistoryPageHandler(actions.GetOperationsHandler{
					OnlyPayments: true,
				}, streamHandler))
			})
		})
	})

	// transaction history actions
	r.Route("/transactions", func(r chi.Router) {
		r.With(historyMiddleware).Method(http.MethodGet, "/", streamableHistoryPageHandler(actions.GetTransactionsHandler{}, streamHandler))
		r.Route("/{tx_id}", func(r chi.Router) {
			r.Use(historyMiddleware)
			r.Method(http.MethodGet, "/", objectActionHandler{actions.GetTransactionByHashHandler{}})
			r.Method(http.MethodGet, "/effects", streamableHistoryPageHandler(actions.GetEffectsHandler{}, streamHandler))
			r.Method(http.MethodGet, "/operations", streamableHistoryPageHandler(actions.GetOperationsHandler{
				OnlyPayments: false,
			}, streamHandler))
			r.Method(http.MethodGet, "/payments", streamableHistoryPageHandler(actions.GetOperationsHandler{
				OnlyPayments: true,
			}, streamHandler))
		})
	})

	// operation actions
	r.Route("/operations", func(r chi.Router) {
		r.Use(historyMiddleware)
		r.Method(http.MethodGet, "/", streamableHistoryPageHandler(actions.GetOperationsHandler{
			OnlyPayments: false,
		}, streamHandler))
		r.Method(http.MethodGet, "/{id}", objectActionHandler{actions.GetOperationByIDHandler{}})
		r.Method(http.MethodGet, "/{op_id}/effects", streamableHistoryPageHandler(actions.GetEffectsHandler{}, streamHandler))
	})

	r.Group(func(r chi.Router) {
		r.Use(historyMiddleware)
		// payment actions
		r.Method(http.MethodGet, "/payments", streamableHistoryPageHandler(actions.GetOperationsHandler{
			OnlyPayments: true,
		}, streamHandler))

		// effect actions
		r.Method(http.MethodGet, "/effects", streamableHistoryPageHandler(actions.GetEffectsHandler{}, streamHandler))

		// trading related endpoints
		r.Method(http.MethodGet, "/trades", streamableHistoryPageHandler(actions.GetTradesHandler{}, streamHandler))
		r.Method(http.MethodGet, "/trade_aggregations", restCustomBuiltPageHandler(actions.GetTradeAggregationsHandler{}))
		// /offers/{offer_id} has been created above so we need to use absolute
		// routes here.
		r.Method(http.MethodGet, "/offers/{offer_id}/trades", streamableHistoryPageHandler(actions.GetTradesHandler{}, streamHandler))
	})

	// Transaction submission API
	r.Method(http.MethodPost, "/transactions", objectActionHandler{actions.SubmitTransactionHandler{
		Submitter:         submitter,
		NetworkPassphrase: config.NetworkPassphrase,
	}})

	// Network state related endpoints
	r.Method(http.MethodGet, "/fee_stats", objectActionHandler{actions.FeeStatsHandler{}})

	// friendbot
	if config.FriendbotURL != nil {
		redirectFriendbot := func(w http.ResponseWriter, r *http.Request) {
			redirectURL := config.FriendbotURL.String() + "?" + r.URL.RawQuery
			http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		}
		r.Post("/friendbot", redirectFriendbot)
		r.Get("/friendbot", redirectFriendbot)
	}

	r.NotFound(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		problem.Render(request.Context(), w, problem.NotFound)
	}))

	// internal
	w.internalRouter.Get("/metrics", HandleRaw(&actions.MetricsHandler{registry}))
	w.internalRouter.Get("/debug/pprof/heap", pprof.Index)
	w.internalRouter.Get("/debug/pprof/profile", pprof.Profile)
}

func maybeInitWebRateLimiter(rateQuota *throttled.RateQuota) *throttled.HTTPRateLimiter {
	// Disabled
	if rateQuota == nil {
		return nil
	}

	rateLimiter, err := throttled.NewGCRARateLimiter(LRUCacheSize, *rateQuota)
	if err != nil {
		log.Fatalf("unable to create RateLimiter: %v", err)
	}

	return &throttled.HTTPRateLimiter{
		RateLimiter: rateLimiter,
		DeniedHandler: http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			problem.Render(request.Context(), w, hProblem.RateLimitExceeded)
		}),
		VaryBy: VaryByRemoteIP{},
	}
}

type VaryByRemoteIP struct{}

func (v VaryByRemoteIP) Key(r *http.Request) string {
	return remoteAddrIP(r)
}

func remoteAddrIP(r *http.Request) string {
	// To support IPv6
	lastSemicolon := strings.LastIndex(r.RemoteAddr, ":")
	if lastSemicolon == -1 {
		return r.RemoteAddr
	} else {
		return r.RemoteAddr[0:lastSemicolon]
	}
}

// horizonSession returns a new session that loads data from the horizon
// database. The returned session is bound to `ctx`.
func (w *web) horizonSession(ctx context.Context) (*db.Session, error) {
	err := errorIfHistoryIsStale(w.isHistoryStale())
	if err != nil {
		return nil, err
	}

	return &db.Session{DB: w.historyQ.Session.DB, Ctx: ctx}, nil
}

// isHistoryStale returns true if the latest history ledger is more than
// `StaleThreshold` ledgers behind the latest core ledger
func (w *web) isHistoryStale() bool {
	if w.staleThreshold == 0 {
		return false
	}

	ls := ledger.CurrentState()
	return (ls.CoreLatest - ls.HistoryLatest) > int32(w.staleThreshold)
}

// errorIfHistoryIsStale returns a formatted error if isStale is true.
func errorIfHistoryIsStale(isStale bool) error {
	if !isStale {
		return nil
	}

	ls := ledger.CurrentState()
	err := hProblem.StaleHistory
	err.Extras = map[string]interface{}{
		"history_latest_ledger": ls.HistoryLatest,
		"core_latest_ledger":    ls.CoreLatest,
	}
	return err
}
