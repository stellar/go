package httpx

import (
	"compress/flate"
	"fmt"
	"net/http"
	"net/http/pprof"
	"net/url"
	"time"

	"github.com/go-chi/chi"
	chimiddleware "github.com/go-chi/chi/middleware"
	"github.com/rcrowley/go-metrics"
	"github.com/rs/cors"
	"github.com/sebest/xff"
	"github.com/stellar/throttled"

	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/paths"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/services/horizon/internal/txsub"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/render/problem"
)

type RouterConfig struct {
	DBSession   *db.Session
	TxSubmitter *txsub.System
	RateQuota   *throttled.RateQuota

	SSEUpdateFrequency time.Duration
	StaleThreshold     uint
	ConnectionTimeout  time.Duration
	NetworkPassphrase  string
	MaxPathLength      uint
	PathFinder         paths.Finder
	MetricsRegistry    metrics.Registry
	CoreGetter         actions.CoreSettingsGetter
	HorizonVersion     string
	FriendbotURL       *url.URL
}

type Router struct {
	*chi.Mux
	Internal *chi.Mux
}

func NewRouter(config *RouterConfig, serverMetrics *ServerMetrics) (*Router, error) {
	result := Router{
		Mux:      chi.NewMux(),
		Internal: chi.NewMux(),
	}
	var rateLimiter *throttled.HTTPRateLimiter
	if config.RateQuota != nil {
		var err error
		rateLimiter, err = newRateLimiter(config.RateQuota)
		if err != nil {
			return nil, fmt.Errorf("unable to create RateLimiter: %v", err)
		}
	}
	result.addMiddleware(config, rateLimiter, serverMetrics)
	result.addRoutes(config, rateLimiter)
	return &result, nil
}

func (r *Router) addMiddleware(config *RouterConfig,
	rateLimitter *throttled.HTTPRateLimiter,
	serverMetrics *ServerMetrics) {

	r.Use(chimiddleware.StripSlashes)

	r.Use(requestCacheHeadersMiddleware)
	r.Use(chimiddleware.RequestID)
	r.Use(contextMiddleware)
	r.Use(xff.Handler)
	r.Use(loggerMiddleware)
	r.Use(timeoutMiddleware(config.ConnectionTimeout))
	r.Use(requestMetricsMiddleware(serverMetrics))
	r.Use(recoverMiddleware)
	r.Use(chimiddleware.Compress(flate.DefaultCompression, "application/hal+json"))

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedHeaders: []string{"*"},
		ExposedHeaders: []string{"Date"},
	})
	r.Use(c.Handler)

	if rateLimitter != nil {
		r.Use(rateLimitter.RateLimit)
	}

	// Internal middlewares
	r.Internal.Use(chimiddleware.StripSlashes)
	r.Internal.Use(chimiddleware.RequestID)
	r.Internal.Use(loggerMiddleware)
}

func (r *Router) addRoutes(config *RouterConfig, rateLimiter *throttled.HTTPRateLimiter) {
	stateMiddleware := StateMiddleware{
		HorizonSession: config.DBSession,
	}

	r.Method(http.MethodGet, "/", ObjectActionHandler{Action: actions.GetRootHandler{
		CoreSettingsGetter: config.CoreGetter,
		NetworkPassphrase:  config.NetworkPassphrase,
		FriendbotURL:       config.FriendbotURL,
		HorizonVersion:     config.HorizonVersion,
	}})

	streamHandler := sse.StreamHandler{
		RateLimiter:         rateLimiter,
		LedgerSourceFactory: historyLedgerSourceFactory{updateFrequency: config.SSEUpdateFrequency},
	}

	historyMiddleware := NewHistoryMiddleware(int32(config.StaleThreshold), config.DBSession)

	// State endpoints behind stateMiddleware
	r.Group(func(r chi.Router) {
		r.Use(stateMiddleware.Wrap)

		r.Route("/accounts", func(r chi.Router) {
			r.Method(http.MethodGet, "/", restPageHandler(actions.GetAccountsHandler{}))
			r.Route("/{account_id}", func(r chi.Router) {
				r.Method(
					http.MethodGet,
					"/",
					streamableObjectActionHandler{
						streamHandler: streamHandler,
						action:        actions.GetAccountByIDHandler{},
					},
				)
				accountData := actions.GetAccountDataHandler{}
				r.Method(http.MethodGet, "/data/{key}", WrapRaw(
					streamableObjectActionHandler{streamHandler: streamHandler, action: accountData},
					accountData,
				))
				r.Method(http.MethodGet, "/offers", streamableStatePageHandler(actions.GetAccountOffersHandler{}, streamHandler))
			})
		})

		r.Route("/claimable_balances", func(r chi.Router) {
			r.Method(http.MethodGet, "/{id}", ObjectActionHandler{actions.GetClaimableBalanceByIDHandler{}})
		})

		r.Route("/offers", func(r chi.Router) {
			r.Method(http.MethodGet, "/", restPageHandler(actions.GetOffersHandler{}))
			r.Method(http.MethodGet, "/{id}", ObjectActionHandler{actions.GetOfferByID{}})
		})

		r.Method(http.MethodGet, "/assets", restPageHandler(actions.AssetStatsHandler{}))

		findPaths := ObjectActionHandler{actions.FindPathsHandler{
			StaleThreshold:       config.StaleThreshold,
			SetLastLedgerHeader:  true,
			MaxPathLength:        config.MaxPathLength,
			MaxAssetsParamLength: maxAssetsForPathFinding,
			PathFinder:           config.PathFinder,
		}}
		findFixedPaths := ObjectActionHandler{actions.FindFixedPathsHandler{
			MaxPathLength:        config.MaxPathLength,
			SetLastLedgerHeader:  true,
			MaxAssetsParamLength: maxAssetsForPathFinding,
			PathFinder:           config.PathFinder,
		}}

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
			r.Method(http.MethodGet, "/", ObjectActionHandler{actions.GetLedgerByIDHandler{}})
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
			r.Method(http.MethodGet, "/", ObjectActionHandler{actions.GetTransactionByHashHandler{}})
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
		r.Method(http.MethodGet, "/{id}", ObjectActionHandler{actions.GetOperationByIDHandler{}})
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
		r.Method(http.MethodGet, "/trade_aggregations", ObjectActionHandler{actions.GetTradeAggregationsHandler{}})
		// /offers/{offer_id} has been created above so we need to use absolute
		// routes here.
		r.Method(http.MethodGet, "/offers/{offer_id}/trades", streamableHistoryPageHandler(actions.GetTradesHandler{}, streamHandler))
	})

	// Transaction submission API
	r.Method(http.MethodPost, "/transactions", ObjectActionHandler{actions.SubmitTransactionHandler{
		Submitter:         config.TxSubmitter,
		NetworkPassphrase: config.NetworkPassphrase,
	}})

	// Network state related endpoints
	r.Method(http.MethodGet, "/fee_stats", ObjectActionHandler{actions.FeeStatsHandler{}})

	// friendbot
	if config.FriendbotURL != nil {
		redirectFriendbot := func(w http.ResponseWriter, r *http.Request) {
			redirectURL := config.FriendbotURL.String() + "?" + r.URL.RawQuery
			http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		}
		r.Post("/friendbot", redirectFriendbot)
		r.Get("/friendbot", redirectFriendbot)
	}

	r.NotFound(func(w http.ResponseWriter, request *http.Request) {
		problem.Render(request.Context(), w, problem.NotFound)
	})

	// internal
	r.Internal.Get("/metrics", HandleRaw(&actions.MetricsHandler{config.MetricsRegistry}))
	r.Internal.Get("/debug/pprof/heap", pprof.Index)
	r.Internal.Get("/debug/pprof/profile", pprof.Profile)
}
