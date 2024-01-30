package exporter

import (
	"flag"

	"github.com/pelletier/go-toml"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/ordered"
)

type StellarCoreConfig struct {
	NetworkPassphrase     string   `toml:"network_passphrase"`
	HistoryArchiveUrls    []string `toml:"history_archive_urls"`
	StellarCoreBinaryPath string   `toml:"stellar_core_binary_path"`
	CaptiveCoreTomlPath   string   `toml:"captive_core_toml_path"`
	CaptiveCoreUseDB      bool     `toml:"captive_core_use_db"`
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

func LoadConfig(config *Config) error {
	// Parse command-line options
	startLedger := flag.Uint("start", 0, "Starting ledger")
	endLedger := flag.Uint("end", 0, "Ending ledger")
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

	// Retrieve the latest ledger sequence from history archives
	latestNetworkLedger, err := GetLatestLedgerSequenceFromHistoryArchives(config.StellarCoreConfig.HistoryArchiveUrls)
	logFatalIf(err, "could not retrieve the latest ledger sequence from history archives")

	// Validate config params
	err = ValidateAndSetLedgerRange(config, latestNetworkLedger)
	logFatalIf(err, "Error validating config params.")

	// Validate and build the appropriate range
	// TODO: Make it configurable
	err = AdjustLedgerRange(config)
	if err != nil {
		return errors.Wrap(err, "error validating ledger range")
	}

	return nil
}

func ValidateAndSetLedgerRange(config *Config, latestNetworkLedger uint32) error {
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

func AdjustLedgerRange(config *Config) error {
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
	return nil
}
