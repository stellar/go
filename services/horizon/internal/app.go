package horizon

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/rcrowley/go-metrics"
	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/stellarcore"
	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/operationfeestats"
	"github.com/stellar/go/services/horizon/internal/paths"
	"github.com/stellar/go/services/horizon/internal/reap"
	"github.com/stellar/go/services/horizon/internal/txsub"
	"github.com/stellar/go/support/app"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/log"
	"github.com/throttled/throttled"
	"golang.org/x/net/http2"
	"gopkg.in/tylerb/graceful.v1"
)

// App represents the root of the state of a horizon instance.
type App struct {
	config            Config
	web               *Web
	historyQ          *history.Q
	coreQ             *core.Q
	ctx               context.Context
	cancel            func()
	redis             *redis.Pool
	coreVersion       string
	horizonVersion    string
	networkPassphrase string
	protocolVersion   int32
	submitter         *txsub.System
	paths             paths.Finder
	ingester          *ingest.System
	reaper            *reap.System
	ticks             *time.Ticker

	// metrics
	metrics                  metrics.Registry
	historyLatestLedgerGauge metrics.Gauge
	historyElderLedgerGauge  metrics.Gauge
	horizonConnGauge         metrics.Gauge
	coreLatestLedgerGauge    metrics.Gauge
	coreConnGauge            metrics.Gauge
	goroutineGauge           metrics.Gauge
}

// NewApp constructs an new App instance from the provided config.
func NewApp(config Config) (*App, error) {

	result := &App{config: config}
	result.horizonVersion = app.Version()
	result.networkPassphrase = build.TestNetwork.Passphrase
	result.ticks = time.NewTicker(1 * time.Second)
	result.init()
	return result, nil
}

// Serve starts the horizon web server, binding it to a socket, setting up
// the shutdown signals.
func (a *App) Serve() {

	http.Handle("/", a.web.router)

	addr := fmt.Sprintf(":%d", a.config.Port)

	srv := &graceful.Server{
		Timeout: 10 * time.Second,

		Server: &http.Server{
			Addr:              addr,
			Handler:           http.DefaultServeMux,
			ReadHeaderTimeout: 5 * time.Second,
		},

		ShutdownInitiated: func() {
			log.Info("received signal, gracefully stopping")
			a.Close()
		},
	}

	http2.ConfigureServer(srv.Server, nil)

	log.Infof("Starting horizon on %s (ingest: %v)", addr, a.config.Ingest)

	go a.run()

	var err error
	if a.config.TLSCert != "" {
		err = srv.ListenAndServeTLS(a.config.TLSCert, a.config.TLSKey)
	} else {
		err = srv.ListenAndServe()
	}

	if err != nil {
		log.Panic(err)
	}

	a.CloseDB()

	log.Info("stopped")
}

// Close cancels the app. It does not close DB connections - use App.CloseDB().
func (a *App) Close() {
	a.cancel()
	a.ticks.Stop()
}

// CloseDB closes DB connections. When using during web server shut down make
// sure all requests are first properly finished to avoid "sql: database is
// closed" errors.
func (a *App) CloseDB() {
	a.historyQ.Session.DB.Close()
	a.coreQ.Session.DB.Close()
}

// HistoryQ returns a helper object for performing sql queries against the
// history portion of horizon's database.
func (a *App) HistoryQ() *history.Q {
	return a.historyQ
}

// HorizonSession returns a new session that loads data from the horizon
// database. The returned session is bound to `ctx`.
func (a *App) HorizonSession(ctx context.Context) *db.Session {
	return &db.Session{DB: a.historyQ.Session.DB, Ctx: ctx}
}

// CoreSession returns a new session that loads data from the stellar core
// database. The returned session is bound to `ctx`.
func (a *App) CoreSession(ctx context.Context) *db.Session {
	return &db.Session{DB: a.coreQ.Session.DB, Ctx: ctx}
}

// CoreQ returns a helper object for performing sql queries aginst the
// stellar core database.
func (a *App) CoreQ() *core.Q {
	return a.coreQ
}

// IsHistoryStale returns true if the latest history ledger is more than
// `StaleThreshold` ledgers behind the latest core ledger
func (a *App) IsHistoryStale() bool {
	if a.config.StaleThreshold == 0 {
		return false
	}

	ls := ledger.CurrentState()
	return (ls.CoreLatest - ls.HistoryLatest) > int32(a.config.StaleThreshold)
}

// UpdateLedgerState triggers a refresh of several metrics gauges, such as open
// db connections and ledger state
func (a *App) UpdateLedgerState() {
	var err error
	var next ledger.State

	err = a.CoreQ().LatestLedger(&next.CoreLatest)
	if err != nil {
		goto Failed
	}

	err = a.HistoryQ().LatestLedger(&next.HistoryLatest)
	if err != nil {
		goto Failed
	}

	err = a.HistoryQ().ElderLedger(&next.HistoryElder)
	if err != nil {
		goto Failed
	}

	ledger.SetState(next)
	return

Failed:
	log.WithStack(err).
		WithField("err", err.Error()).
		Error("failed to load ledger state")

}

// UpdateOperationFeeStatsState triggers a refresh of several operation fee metrics
func (a *App) UpdateOperationFeeStatsState() {
	var err error
	var next operationfeestats.State

	var latest history.LatestLedger
	var feeStats history.FeeStats

	cur := operationfeestats.CurrentState()

	err = a.HistoryQ().LatestLedgerBaseFeeAndSequence(&latest)
	if err != nil {
		goto Failed
	}

	// finish early if no new ledgers
	if cur.LastLedger == int64(latest.Sequence) {
		return
	}

	next.LastBaseFee = int64(latest.BaseFee)
	next.LastLedger = int64(latest.Sequence)

	err = a.HistoryQ().TransactionsForLastXLedgers(latest.Sequence, &feeStats)
	if err != nil {
		goto Failed
	}

	// if no transactions in last X ledgers, return
	// latest ledger's base fee for all
	if !feeStats.Mode.Valid && !feeStats.Min.Valid {
		next.Min = next.LastBaseFee
		next.Mode = next.LastBaseFee
	} else {
		next.Min = feeStats.Min.Int64
		next.Mode = feeStats.Mode.Int64
	}

	operationfeestats.SetState(next)
	return

Failed:
	// If DB is empty ignore the error
	if err == sql.ErrNoRows {
		return
	}

	log.WithStack(err).
		WithField("err", err.Error()).
		Error("failed to load operation fee stats state")

}

// UpdateStellarCoreInfo updates the value of coreVersion and networkPassphrase
// from the Stellar core API.
func (a *App) UpdateStellarCoreInfo() {
	if a.config.StellarCoreURL == "" {
		return
	}

	fail := func(err error) {
		log.Warnf("could not load stellar-core info: %s", err)
	}

	core := &stellarcore.Client{
		URL: a.config.StellarCoreURL,
	}

	resp, err := core.Info(context.Background())

	if err != nil {
		fail(err)
		return
	}

	a.coreVersion = resp.Info.Build
	a.networkPassphrase = resp.Info.Network
	a.protocolVersion = int32(resp.Info.ProtocolVersion)
}

// UpdateMetrics triggers a refresh of several metrics gauges, such as open
// db connections and ledger state
func (a *App) UpdateMetrics() {
	a.goroutineGauge.Update(int64(runtime.NumGoroutine()))
	ls := ledger.CurrentState()
	a.historyLatestLedgerGauge.Update(int64(ls.HistoryLatest))
	a.historyElderLedgerGauge.Update(int64(ls.HistoryElder))
	a.coreLatestLedgerGauge.Update(int64(ls.CoreLatest))

	a.horizonConnGauge.Update(int64(a.historyQ.Session.DB.Stats().OpenConnections))
	a.coreConnGauge.Update(int64(a.coreQ.Session.DB.Stats().OpenConnections))
}

// DeleteUnretainedHistory forwards to the app's reaper.  See
// `reap.DeleteUnretainedHistory` for details
func (a *App) DeleteUnretainedHistory() error {
	return a.reaper.DeleteUnretainedHistory()
}

// Tick triggers horizon to update all of it's background processes such as
// transaction submission, metrics, ingestion and reaping.
func (a *App) Tick() {
	var wg sync.WaitGroup
	log.Debug("ticking app")
	// update ledger state, operation fee state, and stellar-core info in parallel
	wg.Add(3)
	go func() { a.UpdateLedgerState(); wg.Done() }()
	go func() { a.UpdateOperationFeeStatsState(); wg.Done() }()
	go func() { a.UpdateStellarCoreInfo(); wg.Done() }()
	wg.Wait()

	if a.ingester != nil {
		go a.ingester.Tick()
	}

	wg.Add(2)
	go func() { a.reaper.Tick(); wg.Done() }()
	go func() { a.submitter.Tick(a.ctx); wg.Done() }()
	wg.Wait()

	// finally, update metrics
	a.UpdateMetrics()
	log.Debug("finished ticking app")
}

// Init initializes app, using the config to populate db connections and
// whatnot.
func (a *App) init() {
	appInit.Run(a)
}

// run is the function that runs in the background that triggers Tick each
// second
func (a *App) run() {
	for {
		select {
		case <-a.ticks.C:
			a.Tick()
		case <-a.ctx.Done():
			log.Info("finished background ticker")
			return
		}
	}
}

// Context create a context on from the App type.
func (a *App) Context(ctx context.Context) context.Context {
	return context.WithValue(ctx, &horizonContext.AppContextKey, a)
}

// GetRateLimiter returns the HTTPRateLimiter of the App.
func (a *App) GetRateLimiter() *throttled.HTTPRateLimiter {
	return a.web.rateLimiter
}

// AppFromContext returns the set app, if one has been set, from the
// provided context returns nil if no app has been set.
func AppFromContext(ctx context.Context) *App {
	if ctx == nil {
		return nil
	}

	val := ctx.Value(&horizonContext.AppContextKey)
	if val == nil {
		return nil
	}

	result, ok := val.(*App)

	if ok {
		return result
	}

	return nil
}
