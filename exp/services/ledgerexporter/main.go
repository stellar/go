package main

import (
	"context"
	"flag"
	"github.com/pelletier/go-toml"
	"github.com/stellar/go/ingest/ledgerbackend"
	_ "github.com/stellar/go/network"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stellar/go/support/storage"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	logger             = supportlog.New()
	config             Config
	backend            ledgerbackend.LedgerBackend
	destinationStorage storage.Storage
	exporter           *Exporter
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
	ExporterConfig    ExporterConfig    `toml:"exporter_config"`
	StellarCoreConfig StellarCoreConfig `toml:"stellar_core_config"`

	//From command-line
	StartLedger uint32 `toml:"start_ledger"`
	EndLedger   uint32 `toml:"end_ledger"`
}

func main() {
	logger.SetLevel(supportlog.InfoLevel)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		// Handle OS signals to gracefully terminate the service
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		<-ch
		logger.Info("Received termination signal, shutting down...")
		cancel()
	}()

	loadConfig()

	initLedgerBackend(config)

	initDestinationStorage(config)

	// Initialize exporter
	exporter = NewExporter(
		config.ExporterConfig,
		destinationStorage,
		backend,
	)

	var wg sync.WaitGroup
	wg.Add(1) // for the exporter

	go func() {
		exporter.Run(ctx, config.StartLedger, config.EndLedger)
		wg.Done() // signal completion when Run finishes
	}()

	// TODO:
	//close ledgerbackend. Gracefully shutdown captive core.
	//close destinationstorage

	wg.Wait() // wait for the exporter to finish
	logger.Info("Shutting down service.")

}

func initDestinationStorage(config Config) {
	var err error
	destinationStorage, err = storage.ConnectBackend(config.ExporterConfig.DestinationUrl, storage.ConnectOptions{})
	logFatalIf(err, "Could not connect to destination storage")
}

func loadConfig() {
	// Parse command-line options
	startLedger := flag.Uint("start-ledger", 0, "Starting ledger")
	endLedger := flag.Uint("end-ledger", 0, "Ending ledger")
	configFilePath := flag.String("config-file", "config.toml", "Path to the TOML config file")
	flag.Parse()

	// Load config TOML file
	cfg, err := toml.LoadFile(*configFilePath)
	logFatalIf(err, "Error loading %s TOML file:", *configFilePath)

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
}

// Creates and initializes captive core ledger backend
func initLedgerBackend(config Config) {
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
	backend, err = ledgerbackend.NewCaptive(captiveConfig)
	logFatalIf(err, "Could not create captive core instance")

	var ledgerRange ledgerbackend.Range
	if config.EndLedger == 0 {
		ledgerRange = ledgerbackend.UnboundedRange(config.StartLedger)
	} else {
		ledgerRange = ledgerbackend.BoundedRange(config.StartLedger, config.EndLedger)
	}

	err = backend.PrepareRange(context.Background(), ledgerRange)
	logFatalIf(err, "Could not prepare captive core ledger backend")
}

func logFatalIf(err error, message string, args ...interface{}) {
	if err != nil {
		logger.WithError(err).Fatalf(message, args...)
	}
}
