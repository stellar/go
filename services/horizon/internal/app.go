package horizon

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/stellarcore"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/log"
	"github.com/stellar/go/services/horizon/internal/paths"
	"github.com/stellar/go/services/horizon/internal/reap"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/services/horizon/internal/txsub"
	"github.com/stellar/go/support/app"
	"github.com/stellar/go/support/db"
	"golang.org/x/net/http2"
	graceful "gopkg.in/tylerb/graceful.v1"
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

	a.web.router.Compile()
	http.Handle("/", a.web.router)

	addr := fmt.Sprintf(":%d", a.config.Port)

	srv := &graceful.Server{
		Timeout: 10 * time.Second,

		Server: &http.Server{
			Addr:    addr,
			Handler: http.DefaultServeMux,
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

	log.Info("stopped")
}

// Close cancels the app and forces the closure of db connections
func (a *App) Close() {
	a.cancel()
	a.ticks.Stop()

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
	// update ledger state and stellar-core info in parallel
	wg.Add(2)
	go func() { a.UpdateLedgerState(); wg.Done() }()
	go func() { a.UpdateStellarCoreInfo(); wg.Done() }()
	wg.Wait()

	if a.ingester != nil {
		go a.ingester.Tick()
	}

	wg.Add(2)
	go func() { a.reaper.Tick(); wg.Done() }()
	go func() { a.submitter.Tick(a.ctx); wg.Done() }()
	wg.Wait()

	sse.Tick()

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
