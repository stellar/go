package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pelletier/go-toml"
	"github.com/stellar/go/ingest/ledgerbackend"
	_ "github.com/stellar/go/network"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stellar/go/support/storage"
)

var (
	logger = supportlog.New()
)

type StellarCoreConfig struct {
	NetworkPassphrase     string   `toml:"network_passphrase"`
	HistoryArchiveUrls    []string `toml:"history_archive_urls"`
	StellarCoreBinaryPath string   `toml:"stellar_core_binary_path"`
	CaptiveCoreTomlPath   string   `toml:"captive_core_toml_path"`
	CaptiveCoreUseDb      bool     `toml:"captive_core_use_db"`
}

type Config struct {
	Network           string            `toml:"network"`
	DestinationUrl    string            `toml:"destination_url"`
	ExporterConfig    ExporterConfig    `toml:"exporter_config"`
	StellarCoreConfig StellarCoreConfig `toml:"stellar_core_config"`

	//From command-line
	StartLedger uint32 `toml:"start_ledger"`
	EndLedger   uint32 `toml:"end_ledger"`
}

type App struct {
	config             Config
	backend            ledgerbackend.LedgerBackend
	destinationStorage storage.Storage
	exportManager      *ExportManager
	uploader           *Uploader
}

func NewApp() *App {
	app := App{}
	app.config = loadConfig()
	app.destinationStorage = NewDestinationStorage(app.config)
	app.backend = NewLedgerBackend(app.config)

	// Create a channel to send LedgerCloseMetaObject from ExportManager to Uploader
	ledgerCloseMetaObjectCh := make(chan *LedgerCloseMetaObject)

	app.exportManager = NewExportManager(
		app.config.ExporterConfig,
		app.backend,
		ledgerCloseMetaObjectCh,
	)

	app.uploader = NewUploader(app.destinationStorage, ledgerCloseMetaObjectCh)
	return &app
}

func (a *App) Shutdown() {

	// TODO:
	//close ledgerbackend. Gracefully shutdown captive core.
	//close destinationstorage

}

func main() {
	logger.SetLevel(supportlog.InfoLevel)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := NewApp()

	var wg sync.WaitGroup
	wg.Add(2) // for the uploader and export manager

	go func() {
		defer wg.Done()
		app.uploader.Run(ctx)
	}()

	go func() {
		defer wg.Done()
		app.exportManager.Run(ctx, app.config.StartLedger, app.config.EndLedger)
	}()

	go func() {
		// Handle OS signals to gracefully terminate the service
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		<-ch
		logger.Info("Received termination signal, shutting down...")
		cancel()
	}()

	wg.Wait() // wait for the exporter to finish
	logger.Info("Shutting down service.")
	app.Shutdown()
}

func NewDestinationStorage(config Config) storage.Storage {
	destinationStorage, err := storage.ConnectBackend(config.DestinationUrl, storage.ConnectOptions{})
	logFatalIf(err, "Could not connect to destination storage")
	return destinationStorage
}

func loadConfig() Config {
	// Parse command-line options
	startLedger := flag.Uint("start-ledger", 0, "Starting ledger")
	endLedger := flag.Uint("end-ledger", 0, "Ending ledger")
	configFilePath := flag.String("config-file", "config.toml", "Path to the TOML config file")
	flag.Parse()

	// Load config TOML file
	cfg, err := toml.LoadFile(*configFilePath)
	logFatalIf(err, "Error loading %s TOML file:", *configFilePath)

	var config Config
	// Unmarshal TOML data into the Config struct
	err = cfg.Unmarshal(&config)
	logFatalIf(err, "Error unmarshalling TOML config.")

	config.StartLedger = uint32(*startLedger)
	config.EndLedger = uint32(*endLedger)

	// Validate and build the appropriate range
	logger.Infof("processing requested range of -start-ledger=%v, -end-ledger=%v", config.StartLedger, config.EndLedger)

	// TODO: validate end ledger is greater than the latest ledger on the network
	// TODO: validate if either start of end ledger does not fall on "ledgersPerFile" boundary and
	//  adjust the start and end ledger accordingly
	if config.StartLedger < 2 {
		logger.Fatalf("-start-ledger must be >= 2")
	}
	if config.EndLedger != 0 && config.EndLedger < config.StartLedger {
		logger.Fatalf("-end-ledger must be >= -start-ledger")
	}
	return config
}

// Creates and initializes captive core ledger backend
// Only supports captive core for now
func NewLedgerBackend(config Config) ledgerbackend.LedgerBackend {
	coreConfig := config.StellarCoreConfig
	params := ledgerbackend.CaptiveCoreTomlParams{
		NetworkPassphrase:  coreConfig.NetworkPassphrase,
		HistoryArchiveURLs: coreConfig.HistoryArchiveUrls,
		UseDB:              coreConfig.CaptiveCoreUseDb,
	}
	if coreConfig.CaptiveCoreTomlPath == "" {
		logger.Fatal("Missing captive_core_toml_path in the config")
	}

	captiveCoreToml, err := ledgerbackend.NewCaptiveCoreTomlFromFile(coreConfig.CaptiveCoreTomlPath, params)

	captiveConfig := ledgerbackend.CaptiveCoreConfig{
		BinaryPath:          coreConfig.StellarCoreBinaryPath,
		NetworkPassphrase:   params.NetworkPassphrase,
		HistoryArchiveURLs:  params.HistoryArchiveURLs,
		CheckpointFrequency: 64,
		Log:                 logger.WithField("subservice", "stellar-core"),
		Toml:                captiveCoreToml,
		UseDB:               coreConfig.CaptiveCoreUseDb,
	}

	// Create a new captive core backend
	backend, err := ledgerbackend.NewCaptive(captiveConfig)
	logFatalIf(err, "Could not create captive core instance")

	var ledgerRange ledgerbackend.Range
	if config.EndLedger == 0 {
		ledgerRange = ledgerbackend.UnboundedRange(config.StartLedger)
	} else {
		ledgerRange = ledgerbackend.BoundedRange(config.StartLedger, config.EndLedger)
	}

	err = backend.PrepareRange(context.Background(), ledgerRange)
	logFatalIf(err, "Could not prepare captive core ledger backend")
	return backend
}

func logFatalIf(err error, message string, args ...interface{}) {
	if err != nil {
		logger.WithError(err).Fatalf(message, args...)
	}
}
