package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/stellar/go/exp/lighthorizon/actions"
	"github.com/stellar/go/exp/lighthorizon/services"
	supportHttp "github.com/stellar/go/support/http"
)

func newWrapResponseWriter(w http.ResponseWriter, r *http.Request) middleware.WrapResponseWriter {
	mw, ok := w.(middleware.WrapResponseWriter)
	if !ok {
		mw = middleware.NewWrapResponseWriter(w, r.ProtoMajor)
	}

	return mw
}

func prometheusMiddleware(requestDurationMetric *prometheus.SummaryVec) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := supportHttp.GetChiRoutePattern(r)
			mw := newWrapResponseWriter(w, r)

			then := time.Now()
			next.ServeHTTP(mw, r)
			duration := time.Since(then)

			requestDurationMetric.With(prometheus.Labels{
				"status": strconv.FormatInt(int64(mw.Status()), 10),
				"method": r.Method,
				"route":  route,
			}).Observe(float64(duration.Seconds()))
		})
	}
}

func lightHorizonHTTPHandler(registry *prometheus.Registry, lightHorizon services.LightHorizon) http.Handler {
	requestDurationMetric := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: "horizon_lite", Subsystem: "http", Name: "requests_duration_seconds",
			Help: "HTTP requests durations, sliding window = 10m",
		},
		[]string{"status", "method", "route"},
	)
	registry.MustRegister(requestDurationMetric)

	router := chi.NewMux()
	router.Use(prometheusMiddleware(requestDurationMetric))

	router.Route("/accounts/{account_id}", func(r chi.Router) {
		r.MethodFunc(http.MethodGet, "/transactions", actions.NewTXByAccountHandler(lightHorizon))
		r.MethodFunc(http.MethodGet, "/operations", actions.NewOpsByAccountHandler(lightHorizon))
	})

	router.MethodFunc(http.MethodGet, "/", actions.ApiDocs())
	router.Method(http.MethodGet, "/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	return router
}
