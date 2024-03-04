package ledgerexporter

import (
	_ "embed"
	"flag"
	"os/exec"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/network"

	"github.com/pelletier/go-toml"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/ordered"
)

const Pubnet = "pubnet"
const Testnet = "testnet"

type StellarCoreConfig struct {
	NetworkPassphrase     string   `toml:"network_passphrase"`
	HistoryArchiveUrls    []string `toml:"history_archive_urls"`
	StellarCoreBinaryPath string   `toml:"stellar_core_binary_path"`
	CaptiveCoreTomlPath   string   `toml:"captive_core_toml_path"`
}

type Config struct {
	Network           string            `toml:"network"`
	DestinationURL    string            `toml:"destination_url"`
	ExporterConfig    ExporterConfig    `toml:"exporter_config"`
	StellarCoreConfig StellarCoreConfig `toml:"stellar_core_config"`

	//From command-line
	StartLedger          uint32 `toml:"start"`
	EndLedger            uint32 `toml:"end"`
	StartFromLastLedgers uint32 `toml:"from-last"`
}

func (config *Config) LoadConfig() error {
	// Parse command-line options
	startLedger := flag.Uint("start", 0, "Starting ledger")
	endLedger := flag.Uint("end", 0, "Ending ledger (inclusive)")
	startFromLastNLedger := flag.Uint("from-last", 0, "Start streaming from last N ledgers")

	configFilePath := flag.String("config-file", "config.toml", "Path to the TOML config file")
	flag.Parse()

	config.StartLedger = uint32(*startLedger)
	config.EndLedger = uint32(*endLedger)
	config.StartFromLastLedgers = uint32(*startFromLastNLedger)

	// Load config TOML file
	cfg, err := toml.LoadFile(*configFilePath)
	if err != nil {
		return err
	}

	// Unmarshal TOML data into the Config struct
	err = cfg.Unmarshal(config)
	logFatalIf(err, "Error unmarshalling TOML config.")
	logger.Infof("Config: %v", *config)

	var historyArchiveUrls []string
	switch config.Network {
	case Pubnet:
		historyArchiveUrls = network.PublicNetworkhistoryArchiveURLs
	case Testnet:
		historyArchiveUrls = network.TestNetworkhistoryArchiveURLs
	default:
		logger.Fatalf("Invalid network %s", config.Network)
	}

	// Retrieve the latest ledger sequence from history archives
	latestNetworkLedger, err := getLatestLedgerSequenceFromHistoryArchives(historyArchiveUrls)
	logFatalIf(err, "Failed to retrieve the latest ledger sequence from history archives.")

	// Validate config params
	err = config.validateAndSetLedgerRange(latestNetworkLedger)
	logFatalIf(err, "Error validating config params.")

	// Validate and build the appropriate range
	// TODO: Make it configurable
	config.adjustLedgerRange()

	return nil
}

func (config *Config) validateAndSetLedgerRange(latestNetworkLedger uint32) error {
	if config.StartFromLastLedgers > 0 && (config.StartLedger > 0 || config.EndLedger > 0) {
		return errors.New("--from-last cannot be used with --start or --end")
	}

	if config.StartFromLastLedgers > 0 {
		if config.StartFromLastLedgers > latestNetworkLedger {
			return errors.Errorf("--from-last %d exceeds latest network ledger %d",
				config.StartLedger, latestNetworkLedger)
		}
		config.StartLedger = latestNetworkLedger - config.StartFromLastLedgers
		logger.Infof("Setting start ledger to %d, calculated as latest ledger (%d) minus --from-last value (%d)",
			config.StartLedger, latestNetworkLedger, config.StartFromLastLedgers)
	}

	if config.StartLedger > latestNetworkLedger {
		return errors.Errorf("--start %d exceeds latest network ledger %d",
			config.StartLedger, latestNetworkLedger)
	}

	// Ensure that the start ledger is at least 2.
	config.StartLedger = ordered.Max(2, config.StartLedger)

	if config.EndLedger != 0 { // Bounded mode
		if config.EndLedger < config.StartLedger {
			return errors.New("invalid --end value, must be >= --start")
		}
		if config.EndLedger > latestNetworkLedger {
			return errors.Errorf("--end %d exceeds latest network ledger %d",
				config.EndLedger, latestNetworkLedger)
		}
	}

	return nil
}

func (config *Config) adjustLedgerRange() {
	logger.Infof("Requested ledger range start=%d, end=%d", config.StartLedger, config.EndLedger)

	// Check if either the start or end ledger does not fall on the "LedgersPerFile" boundary
	// and adjust the start and end ledger accordingly.
	// Align the start ledger to the nearest "LedgersPerFile" boundary.
	config.StartLedger = config.StartLedger / config.ExporterConfig.LedgersPerFile * config.ExporterConfig.LedgersPerFile

	// Ensure that the adjusted start ledger is at least 2.
	config.StartLedger = ordered.Max(2, config.StartLedger)

	// Align the end ledger (for bounded cases) to the nearest "LedgersPerFile" boundary.
	if config.EndLedger != 0 {
		// Add an extra batch only if "LedgersPerFile" is greater than 1 and the end ledger doesn't fall on the boundary.
		if config.ExporterConfig.LedgersPerFile > 1 && config.EndLedger%config.ExporterConfig.LedgersPerFile != 0 {
			config.EndLedger = (config.EndLedger/config.ExporterConfig.LedgersPerFile + 1) * config.ExporterConfig.LedgersPerFile
		}
	}

	logger.Infof("Adjusted ledger range: start=%d, end=%d", config.StartLedger, config.EndLedger)
}

func (config *Config) GenerateCaptiveCoreConfig() ledgerbackend.CaptiveCoreConfig {
	coreConfig := &config.StellarCoreConfig

	// Look for stellar-core binary in $PATH, if not supplied
	if coreConfig.StellarCoreBinaryPath == "" {
		var err error
		coreConfig.StellarCoreBinaryPath, err = exec.LookPath("stellar-core")
		logFatalIf(err, "Failed to find stellar-core binary")
	}

	var captiveCoreConfig []byte
	// Default network config
	switch config.Network {
	case Pubnet:
		coreConfig.NetworkPassphrase = network.PublicNetworkPassphrase
		coreConfig.HistoryArchiveUrls = network.PublicNetworkhistoryArchiveURLs
		captiveCoreConfig = ledgerbackend.PubnetDefaultConfig

	case Testnet:
		coreConfig.NetworkPassphrase = network.TestNetworkPassphrase
		coreConfig.HistoryArchiveUrls = network.TestNetworkhistoryArchiveURLs
		captiveCoreConfig = ledgerbackend.TestnetDefaultConfig

	default:
		logger.Fatalf("Invalid network %s", config.Network)
	}

	params := ledgerbackend.CaptiveCoreTomlParams{
		NetworkPassphrase:  coreConfig.NetworkPassphrase,
		HistoryArchiveURLs: coreConfig.HistoryArchiveUrls,
		UseDB:              true,
	}

	captiveCoreToml, err := ledgerbackend.NewCaptiveCoreTomlFromData(captiveCoreConfig, params)
	logFatalIf(err, "Failed to create captive-core toml")

	return ledgerbackend.CaptiveCoreConfig{
		BinaryPath:          coreConfig.StellarCoreBinaryPath,
		NetworkPassphrase:   params.NetworkPassphrase,
		HistoryArchiveURLs:  params.HistoryArchiveURLs,
		CheckpointFrequency: historyarchive.DefaultCheckpointFrequency,
		Log:                 logger.WithField("subservice", "stellar-core"),
		Toml:                captiveCoreToml,
		UserAgent:           "ledger-exporter",
		UseDB:               true,
	}
}
