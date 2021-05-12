package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/prometheus/client_golang/prometheus"
)

type CtxKey string

var RouteContextKey = CtxKey("route")

// contextRoute returns a string representing the request endpoint, or "undefined" if it wasn't found
func contextRoute(ctx context.Context) string {
	if endpoint, ok := ctx.Value(&RouteContextKey).(string); ok {
		return endpoint
	}
	return "undefined"
}

type SessionWithMetrics struct {
	SessionInterface
	registry                 *prometheus.Registry
	queryCounter             *prometheus.CounterVec
	queryDuration            *prometheus.HistogramVec
	maxOpenConnectionsGauge  prometheus.GaugeFunc
	openConnectionsGauge     prometheus.GaugeFunc
	inUseConnectionsGauge    prometheus.GaugeFunc
	idleConnectionsGauge     prometheus.GaugeFunc
	waitCountCounter         prometheus.CounterFunc
	waitDurationCounter      prometheus.CounterFunc
	maxIdleClosedCounter     prometheus.CounterFunc
	maxIdleTimeClosedCounter prometheus.CounterFunc
	maxLifetimeClosedCounter prometheus.CounterFunc
}

func RegisterMetrics(base *Session, namespace, sub string, registry *prometheus.Registry) SessionInterface {
	subsystem := fmt.Sprintf("db_%s", sub)
	s := &SessionWithMetrics{
		SessionInterface: base,
		registry:         registry,
	}

	s.queryCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: namespace, Subsystem: subsystem, Name: "query_total"},
		[]string{"query_type", "error", "route"},
	)
	registry.MustRegister(s.queryCounter)

	s.queryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace, Subsystem: subsystem,
			Name:    "query_duration_seconds",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 5),
		},
		[]string{"query_type", "error", "route"},
	)
	registry.MustRegister(s.queryDuration)

	// txnCounter: prometheus.NewCounter(
	// 	prometheus.CounterOpts{Namespace: namespace, Subsystem: subsystem, Name: "transaction_total"},
	// ),
	// registry.MustRegister(s.txnCounter)
	// txnDuration: prometheus.NewHistogram(
	// 	prometheus.HistogramOpts{
	// 		Namespace: namespace, Subsystem: subsystem,
	// 		Name:    "transaction_duration_seconds",
	//		Buckets: prometheus.ExponentialBuckets(0.1, 3, 5),
	// 	},
	// ),
	// registry.MustRegister(s.txnDuration)

	s.maxOpenConnectionsGauge = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{Namespace: namespace, Subsystem: subsystem, Name: "max_open_connections"},
		func() float64 {
			// Right now MaxOpenConnections in Horizon is static however it's possible that
			// it will change one day. In such case, using GaugeFunc is very cheap and will
			// prevent issues with this metric in the future.
			return float64(base.DB.Stats().MaxOpenConnections)
		},
	)
	registry.MustRegister(s.maxOpenConnectionsGauge)

	s.openConnectionsGauge = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{Namespace: namespace, Subsystem: subsystem, Name: "open_connections"},
		func() float64 {
			return float64(base.DB.Stats().OpenConnections)
		},
	)
	registry.MustRegister(s.openConnectionsGauge)

	s.inUseConnectionsGauge = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{Namespace: namespace, Subsystem: subsystem, Name: "in_use_connections"},
		func() float64 {
			return float64(base.DB.Stats().InUse)
		},
	)
	registry.MustRegister(s.inUseConnectionsGauge)

	s.idleConnectionsGauge = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{Namespace: namespace, Subsystem: subsystem, Name: "idle_connections"},
		func() float64 {
			return float64(base.DB.Stats().Idle)
		},
	)
	registry.MustRegister(s.idleConnectionsGauge)

	s.waitCountCounter = prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Namespace: namespace, Subsystem: subsystem, Name: "wait_count_total",
			Help: "total number of number of connections waited for",
		},
		func() float64 {
			return float64(base.DB.Stats().WaitCount)
		},
	)
	registry.MustRegister(s.waitCountCounter)

	s.waitDurationCounter = prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Namespace: namespace, Subsystem: subsystem, Name: "wait_duration_seconds_total",
			Help: "total time blocked waiting for a new connection",
		},
		func() float64 {
			return base.DB.Stats().WaitDuration.Seconds()
		},
	)
	registry.MustRegister(s.waitDurationCounter)

	s.maxIdleClosedCounter = prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Namespace: namespace, Subsystem: subsystem, Name: "max_idle_closed_total",
			Help: "total number of number of connections closed due to SetMaxIdleConns",
		},
		func() float64 {
			return float64(base.DB.Stats().MaxIdleClosed)
		},
	)
	registry.MustRegister(s.maxIdleClosedCounter)

	s.maxIdleTimeClosedCounter = prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Namespace: namespace, Subsystem: subsystem, Name: "max_idle_time_closed_total",
			Help: "total number of number of connections closed due to SetConnMaxIdleTime",
		},
		func() float64 {
			return float64(base.DB.Stats().MaxIdleTimeClosed)
		},
	)
	registry.MustRegister(s.maxIdleTimeClosedCounter)

	s.maxLifetimeClosedCounter = prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Namespace: namespace, Subsystem: subsystem, Name: "max_lifetime_closed_total",
			Help: "total number of number of connections closed due to SetConnMaxLifetime",
		},
		func() float64 {
			return float64(base.DB.Stats().MaxLifetimeClosed)
		},
	)
	registry.MustRegister(s.maxLifetimeClosedCounter)

	return s
}

func (s *SessionWithMetrics) Close() error {
	s.registry.Unregister(s.queryCounter)
	s.registry.Unregister(s.queryDuration)
	// s.registry.Unregister(s.txnCounter)
	// s.registry.Unregister(s.txnDuration)
	s.registry.Unregister(s.maxOpenConnectionsGauge)
	s.registry.Unregister(s.openConnectionsGauge)
	s.registry.Unregister(s.inUseConnectionsGauge)
	s.registry.Unregister(s.idleConnectionsGauge)
	s.registry.Unregister(s.waitCountCounter)
	s.registry.Unregister(s.waitDurationCounter)
	s.registry.Unregister(s.maxIdleClosedCounter)
	s.registry.Unregister(s.maxIdleTimeClosedCounter)
	s.registry.Unregister(s.maxLifetimeClosedCounter)
	return s.SessionInterface.Close()
}

// TODO: Implement these
// func (s *SessionWithMetrics) BeginTx(ctx context.Context, opts *sql.TxOptions) error {
// func (s *SessionWithMetrics) Begin(ctx context.Context) error {
// func (s *SessionWithMetrics) Commit(ctx context.Context) error
// func (s *SessionWithMetrics) Rollback(ctx context.Context) error

func (s *SessionWithMetrics) TruncateTables(ctx context.Context, tables []string) (err error) {
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		s.queryDuration.With(prometheus.Labels{
			"query_type": "truncate_tables",
			"error":      fmt.Sprint(err != nil),
			"route":      contextRoute(ctx),
		}).Observe(v)
	}))
	defer func() {
		timer.ObserveDuration()
		s.queryCounter.With(prometheus.Labels{
			"query_type": "truncate_tables",
			"error":      fmt.Sprint(err != nil),
			"route":      contextRoute(ctx),
		}).Inc()
	}()

	err = s.SessionInterface.TruncateTables(ctx, tables)
	return err
}

func (s *SessionWithMetrics) Clone() SessionInterface {
	return &SessionWithMetrics{
		SessionInterface:         s.SessionInterface.Clone(),
		registry:                 s.registry,
		queryCounter:             s.queryCounter,
		queryDuration:            s.queryDuration,
		// txnCounter:               s.txnCounter,
		// txnDuration:              s.txnDuration,
		maxOpenConnectionsGauge:  s.maxOpenConnectionsGauge,
		openConnectionsGauge:     s.openConnectionsGauge,
		inUseConnectionsGauge:    s.inUseConnectionsGauge,
		idleConnectionsGauge:     s.idleConnectionsGauge,
		waitCountCounter:         s.waitCountCounter,
		waitDurationCounter:      s.waitDurationCounter,
		maxIdleClosedCounter:     s.maxIdleClosedCounter,
		maxIdleTimeClosedCounter: s.maxIdleTimeClosedCounter,
		maxLifetimeClosedCounter: s.maxLifetimeClosedCounter,
	}
}

func getQueryType(query squirrel.Sqlizer) string {
	if _, ok := query.(squirrel.DeleteBuilder); ok {
		return "delete"
	}
	if _, ok := query.(squirrel.InsertBuilder); ok {
		return "insert"
	}
	if _, ok := query.(squirrel.SelectBuilder); ok {
		return "select"
	}
	if _, ok := query.(squirrel.UpdateBuilder); ok {
		return "update"
	}
	return "undefined"
}

func (s *SessionWithMetrics) Get(ctx context.Context, dest interface{}, query squirrel.Sqlizer) (err error) {
	queryType := getQueryType(query)
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		s.queryDuration.With(prometheus.Labels{
			"query_type": queryType,
			"error":      fmt.Sprint(err != nil),
			"route":      contextRoute(ctx),
		}).Observe(v)
	}))
	defer func() {
		timer.ObserveDuration()
		s.queryCounter.With(prometheus.Labels{
			"query_type": queryType,
			"error":      fmt.Sprint(err != nil),
			"route":      contextRoute(ctx),
		}).Inc()
	}()

	err = s.SessionInterface.Get(ctx, dest, query)
	return err
}

func (s *SessionWithMetrics) GetRaw(ctx context.Context, dest interface{}, query string, args ...interface{}) (err error) {
	return s.Get(ctx, dest, squirrel.Expr(query, args...))
}

func (s *SessionWithMetrics) Select(ctx context.Context, dest interface{}, query squirrel.Sqlizer) (err error) {
	queryType := getQueryType(query)
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		s.queryDuration.With(prometheus.Labels{
			"query_type": queryType,
			"error":      fmt.Sprint(err != nil),
			"route":      contextRoute(ctx),
		}).Observe(v)
	}))
	defer func() {
		timer.ObserveDuration()
		s.queryCounter.With(prometheus.Labels{
			"query_type": queryType,
			"error":      fmt.Sprint(err != nil),
			"route":      contextRoute(ctx),
		}).Inc()
	}()

	err = s.SessionInterface.Select(ctx, dest, query)
	return err
}

func (s *SessionWithMetrics) SelectRaw(ctx context.Context, dest interface{}, query string, args ...interface{}) (err error) {
	return s.Select(ctx, dest, squirrel.Expr(query, args...))
}

func (s *SessionWithMetrics) Exec(ctx context.Context, query squirrel.Sqlizer) (result sql.Result, err error) {
	queryType := getQueryType(query)
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		s.queryDuration.With(prometheus.Labels{
			"query_type": queryType,
			"error":      fmt.Sprint(err != nil),
			"route":      contextRoute(ctx),
		}).Observe(v)
	}))
	defer func() {
		timer.ObserveDuration()
		s.queryCounter.With(prometheus.Labels{
			"query_type": queryType,
			"error":      fmt.Sprint(err != nil),
			"route":      contextRoute(ctx),
		}).Inc()
	}()

	result, err = s.SessionInterface.Exec(ctx, query)
	return result, err
}

func (s *SessionWithMetrics) ExecRaw(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	return s.Exec(ctx, squirrel.Expr(query, args...))
}

func (s *SessionWithMetrics) Ping(ctx context.Context, timeout time.Duration) (err error) {
	queryType := "ping"
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		s.queryDuration.With(prometheus.Labels{
			"query_type": queryType,
			"error":      fmt.Sprint(err != nil),
			"route":      contextRoute(ctx),
		}).Observe(v)
	}))
	defer func() {
		timer.ObserveDuration()
		s.queryCounter.With(prometheus.Labels{
			"query_type": queryType,
			"error":      fmt.Sprint(err != nil),
			"route":      contextRoute(ctx),
		}).Inc()
	}()

	err = s.SessionInterface.Ping(ctx, timeout)
	return err
}

func (s *SessionWithMetrics) DeleteRange(
	ctx context.Context,
	start, end int64,
	table string,
	idCol string,
) (err error) {
	queryType := "delete"
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		s.queryDuration.With(prometheus.Labels{
			"query_type": queryType,
			"error":      fmt.Sprint(err != nil),
			"route":      contextRoute(ctx),
		}).Observe(v)
	}))
	defer func() {
		timer.ObserveDuration()
		s.queryCounter.With(prometheus.Labels{
			"query_type": queryType,
			"error":      fmt.Sprint(err != nil),
			"route":      contextRoute(ctx),
		}).Inc()
	}()

	err = s.SessionInterface.DeleteRange(ctx, start, end, table, idCol)
	return err
}
