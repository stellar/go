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
	StartLedger uint32 `toml:"start_ledger"`
	EndLedger   uint32 `toml:"end_ledger"`
}

func LoadConfig(config *Config) error {
	// Parse command-line options
	startLedger := flag.Uint("start-ledger", 0, "Starting ledger")
	endLedger := flag.Uint("end-ledger", 0, "Ending ledger")
	startFromLastNLedger := flag.Uint("start-from-last-n-ledgers", 0, "Start streaming from last N ledgers")

	configFilePath := flag.String("config-file", "config.toml", "Path to the TOML config file")
	flag.Parse()

	// Load config TOML file
	cfg, err := toml.LoadFile(*configFilePath)
	if err != nil {
		return err
	}

	// Unmarshal TOML data into the Config struct
	err = cfg.Unmarshal(config)
	logFatalIf(err, "Error unmarshalling TOML config.")

	//TODO: Validate config params

	// Retrieve the latest ledger sequence from history archives
	latestNetworkLedger, err := GetLatestLedgerSequenceFromHistoryArchives(config.StellarCoreConfig.HistoryArchiveUrls)
	if err != nil {
		return errors.Wrap(err, "could not retrieve the latest ledger sequence from history archives")
	}

	// Validate and build the appropriate range
	config.StartLedger = uint32(*startLedger)
	config.EndLedger = uint32(*endLedger)

	if *startFromLastNLedger != 0 {
		config.StartLedger = ordered.Max(2, latestNetworkLedger-uint32(*startFromLastNLedger))
		logger.Infof("Setting start ledger to %d, latest ledger minus startFromLastNLedger %d - %d",
			config.StartLedger, latestNetworkLedger, *startFromLastNLedger)
	}

	if config.EndLedger > latestNetworkLedger {
		config.EndLedger = latestNetworkLedger
		logger.Warnf("End ledger %d exceeds latest network ledger %d, setting end ledger to %d",
			*endLedger, latestNetworkLedger, config.EndLedger)
	}

	err = validateAndAdjustLedgerRange(config)
	if err != nil {
		return errors.Wrap(err, "error validating ledger range")
	}
	return nil
}

func validateAndAdjustLedgerRange(config *Config) error {
	logger.Infof("Requested ledger range -start-ledger=%v, -end-ledger=%v", config.StartLedger, config.EndLedger)

	if config.EndLedger != 0 && config.EndLedger < config.StartLedger {
		return errors.New("invalid end ledger value, must be >= start ledger")
	}

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

	logger.Infof("Adjusted ledger range: -start-ledger=%v, -end-ledger=%v", config.StartLedger, config.EndLedger)
	return nil
}
