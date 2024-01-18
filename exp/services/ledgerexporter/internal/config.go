package exporter

import (
	"flag"
	"fmt"

	"github.com/pelletier/go-toml"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/ordered"
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

func LoadConfig(config *Config) error {
	// Parse command-line options
	startLedger := flag.Uint("start-ledger", 0, "Starting ledger")
	endLedger := flag.Uint("end-ledger", 0, "Ending ledger")
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

	// Validate and build the appropriate range
	config.StartLedger = uint32(*startLedger)
	config.EndLedger = uint32(*endLedger)

	logger.Infof("Requested ledger range -start-ledger=%v, -end-ledger=%v", config.StartLedger, config.EndLedger)
	err = validateAndAdjustLedgerRange(config)
	if err != nil {
		return errors.Wrap(err, "Error validating ledger range")
	}
	return nil
}

func validateAndAdjustLedgerRange(config *Config) error {
	// TODO: validate end ledger is greater than the latest ledger on the network
	if config.StartLedger < 2 {
		return fmt.Errorf("-start-ledger must be >= 2")
	}
	if config.EndLedger != 0 && config.EndLedger < config.StartLedger {
		return fmt.Errorf("-end-ledger must be >= -start-ledger")
	}

	// Validate if either the start or end ledger does not fall on the "LedgersPerFile" boundary
	// and adjust the start and end ledger accordingly.
	// Align start ledger to the nearest "LedgersPerFile" boundary.
	config.StartLedger = config.StartLedger / config.ExporterConfig.LedgersPerFile *
		config.ExporterConfig.LedgersPerFile

	// Ensure that the adjusted start ledger is at least 2.
	config.StartLedger = ordered.Max(2, config.StartLedger)

	// Align the end ledger(for bounded case) to the nearest "LedgersPerFile" boundary and add an extra batch
	if config.EndLedger != 0 {
		config.EndLedger = ((config.EndLedger / config.ExporterConfig.LedgersPerFile) + 1) *
			config.ExporterConfig.LedgersPerFile
	}
	// TODO: validate end ledger is greater than the latest ledger on the network

	logger.Infof("Adjusted ledger range: -start-ledger=%v, -end-ledger=%v", config.StartLedger,
		config.EndLedger)

	return nil
}
