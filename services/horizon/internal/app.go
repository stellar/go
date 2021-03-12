package horizon

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stellar/go/clients/stellarcore"
	proto "github.com/stellar/go/protocols/stellarcore"
	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/httpx"
	"github.com/stellar/go/services/horizon/internal/ingest"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/logmetrics"
	"github.com/stellar/go/services/horizon/internal/operationfeestats"
	"github.com/stellar/go/services/horizon/internal/paths"
	"github.com/stellar/go/services/horizon/internal/reap"
	"github.com/stellar/go/services/horizon/internal/txsub"
	"github.com/stellar/go/support/app"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

type coreSettingsStore struct {
	sync.RWMutex
	actions.CoreSettings
}

func (c *coreSettingsStore) set(resp *proto.InfoResponse) {
	c.Lock()
	defer c.Unlock()
	c.Synced = resp.IsSynced()
	c.CoreVersion = resp.Info.Build
	c.CurrentProtocolVersion = int32(resp.Info.Ledger.Version)
	c.CoreSupportedProtocolVersion = int32(resp.Info.ProtocolVersion)
}

func (c *coreSettingsStore) get() actions.CoreSettings {
	c.RLock()
	defer c.RUnlock()
	return c.CoreSettings
}

// App represents the root of the state of a horizon instance.
type App struct {
	done            chan struct{}
	config          Config
	webServer       *httpx.Server
	historyQ        *history.Q
	ctx             context.Context
	cancel          func()
	horizonVersion  string
	coreSettings    coreSettingsStore
	orderBookStream *ingest.OrderBookStream
	submitter       *txsub.System
	paths           paths.Finder
	ingester        ingest.System
	reaper          *reap.System
	ticks           *time.Ticker
	ledgerState     *ledger.State

	// metrics
	prometheusRegistry                *prometheus.Registry
	buildInfoGauge                    *prometheus.GaugeVec
	ingestingGauge                    prometheus.Gauge
	historyLatestLedgerCounter        prometheus.CounterFunc
	historyLatestLedgerClosedAgoGauge prometheus.GaugeFunc
	historyElderLedgerCounter         prometheus.CounterFunc
	dbMaxOpenConnectionsGauge         prometheus.GaugeFunc
	dbOpenConnectionsGauge            prometheus.GaugeFunc
	dbInUseConnectionsGauge           prometheus.GaugeFunc
	dbWaitCountCounter                prometheus.CounterFunc
	dbWaitDurationCounter             prometheus.CounterFunc
	coreLatestLedgerCounter           prometheus.CounterFunc
	coreSynced                        prometheus.GaugeFunc
}

func (a *App) GetCoreSettings() actions.CoreSettings {
	return a.coreSettings.get()
}

// NewApp constructs an new App instance from the provided config.
func NewApp(config Config) (*App, error) {
	a := &App{
		config:         config,
		ledgerState:    &ledger.State{},
		horizonVersion: app.Version(),
		ticks:          time.NewTicker(1 * time.Second),
		done:           make(chan struct{}),
	}

	if err := a.init(); err != nil {
		return nil, err
	}
	return a, nil
}

// Serve starts the horizon web server, binding it to a socket, setting up
// the shutdown signals.
func (a *App) Serve() {

	log.Infof("Starting horizon on :%d (ingest: %v)", a.config.Port, a.config.Ingest)

	if a.config.AdminPort != 0 {
		log.Infof("Starting internal server on :%d", a.config.AdminPort)
	}

	go a.run()
	go a.orderBookStream.Run(a.ctx)

	// WaitGroup for all go routines. Makes sure that DB is closed when
	// all services gracefully shutdown.
	var wg sync.WaitGroup

	if a.ingester != nil {
		wg.Add(1)
		go func() {
			a.ingester.Run()
			wg.Done()
		}()
	}

	// configure shutdown signal handler
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-signalChan:
			a.Close()
		case <-a.done:
			return
		}
	}()

	wg.Add(1)
	go func() {
		a.waitForDone()
		wg.Done()
	}()

	err := a.webServer.Serve()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

	wg.Wait()
	a.CloseDB()

	log.Info("stopped")
}

// Close cancels the app. It does not close DB connections - use App.CloseDB().
func (a *App) Close() {
	close(a.done)
}

func (a *App) waitForDone() {
	<-a.done
	webShutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	a.webServer.Shutdown(webShutdownCtx)
	a.cancel()
	if a.ingester != nil {
		a.ingester.Shutdown()
	}
	a.ticks.Stop()
}

// CloseDB closes DB connections. When using during web server shut down make
// sure all requests are first properly finished to avoid "sql: database is
// closed" errors.
func (a *App) CloseDB() {
	a.historyQ.Session.DB.Close()
}

// HistoryQ returns a helper object for performing sql queries against the
// history portion of horizon's database.
func (a *App) HistoryQ() *history.Q {
	return a.historyQ
}

// Ingestion returns the ingestion system associated with this Horizon instance
func (a *App) Ingestion() ingest.System {
	return a.ingester
}

// HorizonSession returns a new session that loads data from the horizon
// database. The returned session is bound to `ctx`.
func (a *App) HorizonSession(ctx context.Context) *db.Session {
	return &db.Session{DB: a.historyQ.Session.DB, Ctx: ctx}
}

// UpdateLedgerState triggers a refresh of several metrics gauges, such as open
// db connections and ledger state
func (a *App) UpdateLedgerState() {
	var next ledger.Status

	logErr := func(err error, msg string) {
		log.WithStack(err).WithField("err", err.Error()).Error(msg)
	}

	coreClient := &stellarcore.Client{
		HTTP: http.DefaultClient,
		URL:  a.config.StellarCoreURL,
	}

	coreInfo, err := coreClient.Info(a.ctx)
	if err != nil {
		logErr(err, "failed to load the stellar-core info")
		return
	}
	next.CoreLatest = int32(coreInfo.Info.Ledger.Num)

	next.HistoryLatest, next.HistoryLatestClosedAt, err =
		a.HistoryQ().LatestLedgerSequenceClosedAt()
	if err != nil {
		logErr(err, "failed to load the latest known ledger state from history DB")
		return
	}

	err = a.HistoryQ().ElderLedger(&next.HistoryElder)
	if err != nil {
		logErr(err, "failed to load the oldest known ledger state from history DB")
		return
	}

	next.ExpHistoryLatest, err = a.HistoryQ().GetLastLedgerIngestNonBlocking()
	if err != nil {
		logErr(err, "failed to load the oldest known exp ledger state from history DB")
		return
	}

	a.ledgerState.SetStatus(next)
}

// UpdateFeeStatsState triggers a refresh of several operation fee metrics.
func (a *App) UpdateFeeStatsState() {
	var (
		next          operationfeestats.State
		latest        history.LatestLedger
		feeStats      history.FeeStats
		capacityStats history.LedgerCapacityUsageStats
	)

	logErr := func(err error, msg string) {
		// If DB is empty ignore the error
		if errors.Cause(err) == sql.ErrNoRows {
			return
		}

		log.WithStack(err).WithField("err", err.Error()).Error(msg)
	}

	cur, ok := operationfeestats.CurrentState()

	err := a.HistoryQ().LatestLedgerBaseFeeAndSequence(&latest)
	if err != nil {
		logErr(err, "failed to load the latest known ledger's base fee and sequence number")
		return
	}

	// finish early if no new ledgers
	if ok && cur.LastLedger == uint32(latest.Sequence) {
		return
	}

	next.LastBaseFee = int64(latest.BaseFee)
	next.LastLedger = uint32(latest.Sequence)

	err = a.HistoryQ().FeeStats(latest.Sequence, &feeStats)
	if err != nil {
		logErr(err, "failed to load operation fee stats")
		return
	}

	err = a.HistoryQ().LedgerCapacityUsageStats(latest.Sequence, &capacityStats)
	if err != nil {
		logErr(err, "failed to load ledger capacity usage stats")
		return
	}

	next.LedgerCapacityUsage = capacityStats.CapacityUsage.String

	// if no transactions in last 5 ledgers, return
	// latest ledger's base fee for all
	if !feeStats.MaxFeeMode.Valid && !feeStats.MaxFeeMin.Valid {
		// MaxFee
		next.MaxFeeMax = next.LastBaseFee
		next.MaxFeeMin = next.LastBaseFee
		next.MaxFeeMode = next.LastBaseFee
		next.MaxFeeP10 = next.LastBaseFee
		next.MaxFeeP20 = next.LastBaseFee
		next.MaxFeeP30 = next.LastBaseFee
		next.MaxFeeP40 = next.LastBaseFee
		next.MaxFeeP50 = next.LastBaseFee
		next.MaxFeeP60 = next.LastBaseFee
		next.MaxFeeP70 = next.LastBaseFee
		next.MaxFeeP80 = next.LastBaseFee
		next.MaxFeeP90 = next.LastBaseFee
		next.MaxFeeP95 = next.LastBaseFee
		next.MaxFeeP99 = next.LastBaseFee

		// FeeCharged
		next.FeeChargedMax = next.LastBaseFee
		next.FeeChargedMin = next.LastBaseFee
		next.FeeChargedMode = next.LastBaseFee
		next.FeeChargedP10 = next.LastBaseFee
		next.FeeChargedP20 = next.LastBaseFee
		next.FeeChargedP30 = next.LastBaseFee
		next.FeeChargedP40 = next.LastBaseFee
		next.FeeChargedP50 = next.LastBaseFee
		next.FeeChargedP60 = next.LastBaseFee
		next.FeeChargedP70 = next.LastBaseFee
		next.FeeChargedP80 = next.LastBaseFee
		next.FeeChargedP90 = next.LastBaseFee
		next.FeeChargedP95 = next.LastBaseFee
		next.FeeChargedP99 = next.LastBaseFee

	} else {
		// MaxFee
		next.MaxFeeMax = feeStats.MaxFeeMax.Int64
		next.MaxFeeMin = feeStats.MaxFeeMin.Int64
		next.MaxFeeMode = feeStats.MaxFeeMode.Int64
		next.MaxFeeP10 = feeStats.MaxFeeP10.Int64
		next.MaxFeeP20 = feeStats.MaxFeeP20.Int64
		next.MaxFeeP30 = feeStats.MaxFeeP30.Int64
		next.MaxFeeP40 = feeStats.MaxFeeP40.Int64
		next.MaxFeeP50 = feeStats.MaxFeeP50.Int64
		next.MaxFeeP60 = feeStats.MaxFeeP60.Int64
		next.MaxFeeP70 = feeStats.MaxFeeP70.Int64
		next.MaxFeeP80 = feeStats.MaxFeeP80.Int64
		next.MaxFeeP90 = feeStats.MaxFeeP90.Int64
		next.MaxFeeP95 = feeStats.MaxFeeP95.Int64
		next.MaxFeeP99 = feeStats.MaxFeeP99.Int64

		// FeeCharged
		next.FeeChargedMax = feeStats.FeeChargedMax.Int64
		next.FeeChargedMin = feeStats.FeeChargedMin.Int64
		next.FeeChargedMode = feeStats.FeeChargedMode.Int64
		next.FeeChargedP10 = feeStats.FeeChargedP10.Int64
		next.FeeChargedP20 = feeStats.FeeChargedP20.Int64
		next.FeeChargedP30 = feeStats.FeeChargedP30.Int64
		next.FeeChargedP40 = feeStats.FeeChargedP40.Int64
		next.FeeChargedP50 = feeStats.FeeChargedP50.Int64
		next.FeeChargedP60 = feeStats.FeeChargedP60.Int64
		next.FeeChargedP70 = feeStats.FeeChargedP70.Int64
		next.FeeChargedP80 = feeStats.FeeChargedP80.Int64
		next.FeeChargedP90 = feeStats.FeeChargedP90.Int64
		next.FeeChargedP95 = feeStats.FeeChargedP95.Int64
		next.FeeChargedP99 = feeStats.FeeChargedP99.Int64
	}

	operationfeestats.SetState(next)
}

// UpdateStellarCoreInfo updates the value of CoreVersion,
// CurrentProtocolVersion, and CoreSupportedProtocolVersion from the Stellar
// core API.
func (a *App) UpdateStellarCoreInfo() {
	if a.config.StellarCoreURL == "" {
		return
	}

	core := &stellarcore.Client{
		URL: a.config.StellarCoreURL,
	}

	resp, err := core.Info(context.Background())
	if err != nil {
		log.Warnf("could not load stellar-core info: %s", err)
		return
	}

	// Check if NetworkPassphrase is different, if so exit Horizon as it can break the
	// state of the application.
	if resp.Info.Network != a.config.NetworkPassphrase {
		log.Errorf(
			"Network passphrase of stellar-core (%s) does not match Horizon configuration (%s). Exiting...",
			resp.Info.Network,
			a.config.NetworkPassphrase,
		)
		os.Exit(1)
	}

	a.coreSettings.set(resp)
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
	go func() { a.UpdateFeeStatsState(); wg.Done() }()
	go func() { a.UpdateStellarCoreInfo(); wg.Done() }()
	wg.Wait()

	wg.Add(2)
	go func() { a.reaper.Tick(); wg.Done() }()
	go func() { a.submitter.Tick(a.ctx); wg.Done() }()
	wg.Wait()

	log.Debug("finished ticking app")
}

// Init initializes app, using the config to populate db connections and
// whatnot.
func (a *App) init() error {
	// app-context
	a.ctx, a.cancel = context.WithCancel(context.Background())

	// log
	log.DefaultLogger.Logger.Level = a.config.LogLevel
	log.DefaultLogger.Logger.Hooks.Add(logmetrics.DefaultMetrics)

	// sentry
	initSentry(a)

	// loggly
	initLogglyLog(a)

	// stellarCoreInfo
	a.UpdateStellarCoreInfo()

	// horizon-db and core-db
	mustInitHorizonDB(a)

	if a.config.Ingest {
		// ingester
		initIngester(a)
	}
	initPathFinder(a)

	// txsub
	initSubmissionSystem(a)

	// reaper
	a.reaper = reap.New(a.config.HistoryRetentionCount, a.HorizonSession(context.Background()), a.ledgerState)

	// metrics and log.metrics
	a.prometheusRegistry = prometheus.NewRegistry()
	for _, meter := range *logmetrics.DefaultMetrics {
		a.prometheusRegistry.MustRegister(meter)
	}

	// go metrics
	initGoMetrics(a)

	// process metrics
	initProcessMetrics(a)

	// db-metrics
	initDbMetrics(a)

	// ingest.metrics
	initIngestMetrics(a)

	// txsub.metrics
	initTxSubMetrics(a)

	routerConfig := httpx.RouterConfig{
		DBSession:             a.historyQ.Session,
		TxSubmitter:           a.submitter,
		RateQuota:             a.config.RateQuota,
		BehindCloudflare:      a.config.BehindCloudflare,
		BehindAWSLoadBalancer: a.config.BehindAWSLoadBalancer,
		SSEUpdateFrequency:    a.config.SSEUpdateFrequency,
		StaleThreshold:        a.config.StaleThreshold,
		ConnectionTimeout:     a.config.ConnectionTimeout,
		NetworkPassphrase:     a.config.NetworkPassphrase,
		MaxPathLength:         a.config.MaxPathLength,
		PathFinder:            a.paths,
		PrometheusRegistry:    a.prometheusRegistry,
		CoreGetter:            a,
		HorizonVersion:        a.horizonVersion,
		FriendbotURL:          a.config.FriendbotURL,
		HealthCheck: healthCheck{
			session: a.historyQ.Session,
			ctx:     a.ctx,
			core: &stellarcore.Client{
				HTTP: &http.Client{Timeout: infoRequestTimeout},
				URL:  a.config.StellarCoreURL,
			},
			cache: newHealthCache(healthCacheTTL),
		},
	}

	var err error
	config := httpx.ServerConfig{
		Port:      uint16(a.config.Port),
		AdminPort: uint16(a.config.AdminPort),
	}
	if a.config.TLSCert != "" && a.config.TLSKey != "" {
		config.TLSConfig = &httpx.TLSConfig{
			CertPath: a.config.TLSCert,
			KeyPath:  a.config.TLSKey,
		}
	}
	a.webServer, err = httpx.NewServer(config, routerConfig, a.ledgerState)
	if err != nil {
		return err
	}

	// web.metrics
	initWebMetrics(a)

	return nil
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
