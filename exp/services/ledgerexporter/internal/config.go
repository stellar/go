package ledgerexporter

import (
	"context"
	_ "embed"
	"os/exec"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/network"

	"github.com/pelletier/go-toml"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/ordered"
	"github.com/stellar/go/support/storage"
)

const Pubnet = "pubnet"
const Testnet = "testnet"

type Flags struct {
	StartLedger    uint32
	EndLedger      uint32
	ConfigFilePath string
	Resume         bool
	AdminPort      uint
}

type StellarCoreConfig struct {
	NetworkPassphrase     string   `toml:"network_passphrase"`
	HistoryArchiveUrls    []string `toml:"history_archive_urls"`
	StellarCoreBinaryPath string   `toml:"stellar_core_binary_path"`
	CaptiveCoreTomlPath   string   `toml:"captive_core_toml_path"`
}

type DataStoreConfig struct {
	Type   string            `toml:"type"`
	Params map[string]string `toml:"params"`
}

type Config struct {
	AdminPort int `toml:"admin_port"`

	Network           string            `toml:"network"`
	DataStoreConfig   DataStoreConfig   `toml:"datastore_config"`
	LedgerBatchConfig LedgerBatchConfig `toml:"exporter_config"`
	StellarCoreConfig StellarCoreConfig `toml:"stellar_core_config"`

	StartLedger uint32
	EndLedger   uint32
	Resume      bool
}

func createHistoryArchiveFromNetworkName(ctx context.Context, networkName string) (historyarchive.ArchiveInterface, error) {
	var historyArchiveUrls []string
	switch networkName {
	case Pubnet:
		historyArchiveUrls = network.PublicNetworkhistoryArchiveURLs
	case Testnet:
		historyArchiveUrls = network.TestNetworkhistoryArchiveURLs
	default:
		return nil, errors.Errorf("Invalid network name %s", networkName)
	}

	return historyarchive.NewArchivePool(historyArchiveUrls, historyarchive.ArchiveOptions{
		ConnectOptions: storage.ConnectOptions{
			UserAgent: "ledger-exporter",
			Context:   ctx,
		},
	})
}

func getLatestLedgerSequenceFromHistoryArchives(archive historyarchive.ArchiveInterface) (uint32, error) {
	has, err := archive.GetRootHAS()
	if err != nil {
		logger.WithError(err).Warnf("Error getting root HAS from archives")
		return 0, errors.Wrap(err, "failed to retrieve the latest ledger sequence from any history archive")
	}

	return has.CurrentLedger, nil
}

func getHistoryArchivesCheckPointFrequency() uint32 {
	// this could evolve to use other sources for checkpoint freq
	return historyarchive.DefaultCheckpointFrequency
}

// This will generate the config based on commandline flags and toml
//
// ctx                   - the caller context
// flags                 - command line flags
//
// return                - *Config or an error if any range validation failed.
func NewConfig(ctx context.Context, flags Flags) (*Config, error) {
	config := &Config{}

	config.StartLedger = uint32(flags.StartLedger)
	config.EndLedger = uint32(flags.EndLedger)
	config.Resume = flags.Resume

	logger.Infof("Requested ledger range start=%d, end=%d, resume=%v", config.StartLedger, config.EndLedger, config.Resume)

	var err error
	if err = config.processToml(flags.ConfigFilePath); err != nil {
		return nil, err
	}
	logger.Infof("Config: %v", *config)

	return config, nil
}

// Validates requested ledger range, and will automatically adjust it
// to be ledgers-per-file boundary aligned
func (config *Config) ValidateAndSetLedgerRange(ctx context.Context, archive historyarchive.ArchiveInterface) error {
	latestNetworkLedger, err := getLatestLedgerSequenceFromHistoryArchives(archive)

	if err != nil {
		return errors.Wrap(err, "Failed to retrieve the latest ledger sequence from history archives.")
	}
	logger.Infof("Latest %v ledger sequence was detected as %d", config.Network, latestNetworkLedger)

	if config.StartLedger > latestNetworkLedger {
		return errors.Errorf("--start %d exceeds latest network ledger %d",
			config.StartLedger, latestNetworkLedger)
	}

	if config.EndLedger != 0 { // Bounded mode
		if config.EndLedger < config.StartLedger {
			return errors.New("invalid --end value, must be >= --start")
		}
		if config.EndLedger > latestNetworkLedger {
			return errors.Errorf("--end %d exceeds latest network ledger %d",
				config.EndLedger, latestNetworkLedger)
		}
	}

	config.adjustLedgerRange()
	return nil
}

func (config *Config) GenerateCaptiveCoreConfig() (ledgerbackend.CaptiveCoreConfig, error) {
	coreConfig := &config.StellarCoreConfig

	// Look for stellar-core binary in $PATH, if not supplied
	if coreConfig.StellarCoreBinaryPath == "" {
		var err error
		if coreConfig.StellarCoreBinaryPath, err = exec.LookPath("stellar-core"); err != nil {
			return ledgerbackend.CaptiveCoreConfig{}, errors.Wrap(err, "Failed to find stellar-core binary")
		}
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
	if err != nil {
		return ledgerbackend.CaptiveCoreConfig{}, errors.Wrap(err, "Failed to create captive-core toml")
	}

	return ledgerbackend.CaptiveCoreConfig{
		BinaryPath:          coreConfig.StellarCoreBinaryPath,
		NetworkPassphrase:   params.NetworkPassphrase,
		HistoryArchiveURLs:  params.HistoryArchiveURLs,
		CheckpointFrequency: getHistoryArchivesCheckPointFrequency(),
		Log:                 logger.WithField("subservice", "stellar-core"),
		Toml:                captiveCoreToml,
		UserAgent:           "ledger-exporter",
		UseDB:               true,
	}, nil
}

func (config *Config) processToml(tomlPath string) error {
	// Load config TOML file
	cfg, err := toml.LoadFile(tomlPath)
	if err != nil {
		return err
	}

	// Unmarshal TOML data into the Config struct
	if err := cfg.Unmarshal(config); err != nil {
		return errors.Wrap(err, "Error unmarshalling TOML config.")
	}

	// validate TOML data
	if config.Network == "" {
		return errors.New("Invalid TOML config, 'network' must be set, supported values are 'testnet' or 'pubnet'")
	}
	return nil
}

func (config *Config) adjustLedgerRange() {

	// Check if either the start or end ledger does not fall on the "LedgersPerFile" boundary
	// and adjust the start and end ledger accordingly.
	// Align the start ledger to the nearest "LedgersPerFile" boundary.
	config.StartLedger = config.LedgerBatchConfig.GetSequenceNumberStartBoundary(config.StartLedger)

	// Ensure that the adjusted start ledger is at least 2.
	config.StartLedger = ordered.Max(2, config.StartLedger)

	// Align the end ledger (for bounded cases) to the nearest "LedgersPerFile" boundary.
	if config.EndLedger != 0 {
		// Add an extra batch only if "LedgersPerFile" is greater than 1 and the end ledger doesn't fall on the boundary.
		if config.LedgerBatchConfig.LedgersPerFile > 1 && config.EndLedger%config.LedgerBatchConfig.LedgersPerFile != 0 {
			config.EndLedger = (config.EndLedger/config.LedgerBatchConfig.LedgersPerFile + 1) * config.LedgerBatchConfig.LedgersPerFile
		}
	}

	logger.Infof("Computed effective export boundary ledger range: start=%d, end=%d", config.StartLedger, config.EndLedger)
}
