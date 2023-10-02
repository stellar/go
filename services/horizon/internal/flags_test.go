package horizon

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_createCaptiveCoreDefaultConfig(t *testing.T) {

	var errorMsgDefaultConfig = "invalid config: %s parameter not allowed with the network parameter"
	tests := []struct {
		name               string
		config             Config
		networkPassphrase  string
		historyArchiveURLs []string
		errStr             string
	}{
		{
			name:               "testnet default config",
			config:             Config{Network: StellarTestnet},
			networkPassphrase:  TestnetConf.NetworkPassphrase,
			historyArchiveURLs: TestnetConf.HistoryArchiveURLs,
		},
		{
			name:               "pubnet default config",
			config:             Config{Network: StellarPubnet},
			networkPassphrase:  PubnetConf.NetworkPassphrase,
			historyArchiveURLs: PubnetConf.HistoryArchiveURLs,
		},
		{
			name: "testnet validation; history archive urls supplied",
			config: Config{Network: StellarTestnet,
				HistoryArchiveURLs: []string{"network history archive urls supplied"},
			},
			errStr: fmt.Sprintf(errorMsgDefaultConfig, HistoryArchiveURLsFlagName),
		},
		{
			name: "pubnet validation; history archive urls supplied",
			config: Config{Network: StellarPubnet,
				HistoryArchiveURLs: []string{"network history archive urls supplied"},
			},
			errStr: fmt.Sprintf(errorMsgDefaultConfig, HistoryArchiveURLsFlagName),
		},
		{
			name: "testnet validation; network passphrase supplied",
			config: Config{Network: StellarTestnet,
				NetworkPassphrase:  "network passphrase supplied",
				HistoryArchiveURLs: []string{},
			},
			errStr: fmt.Sprintf(errorMsgDefaultConfig, NetworkPassphraseFlagName),
		},
		{
			name: "pubnet validation; network passphrase supplied",
			config: Config{Network: StellarPubnet,
				NetworkPassphrase:  "pubnet network passphrase supplied",
				HistoryArchiveURLs: []string{},
			},
			errStr: fmt.Sprintf(errorMsgDefaultConfig, NetworkPassphraseFlagName),
		},
		{
			name: "unknown network specified",
			config: Config{Network: "unknown",
				NetworkPassphrase:  "",
				HistoryArchiveURLs: []string{},
			},
			errStr: "no default configuration found for network unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := createCaptiveCoreConfigFromNetwork(&tt.config)
			if tt.errStr == "" {
				assert.NoError(t, e)
				assert.Equal(t, tt.networkPassphrase, tt.config.NetworkPassphrase)
				assert.Equal(t, tt.historyArchiveURLs, tt.config.HistoryArchiveURLs)
			} else {
				assert.Equal(t, tt.errStr, e.Error())
			}
		})
	}
}

func Test_createCaptiveCoreConfig(t *testing.T) {

	var errorMsgConfig = "%s must be set"
	tests := []struct {
		name               string
		config             Config
		networkPassphrase  string
		historyArchiveURLs []string
		errStr             string
	}{
		{
			name: "no network specified",
			config: Config{
				NetworkPassphrase:  "NetworkPassphrase",
				HistoryArchiveURLs: []string{"HistoryArchiveURLs"},
			},
			networkPassphrase:  "NetworkPassphrase",
			historyArchiveURLs: []string{"HistoryArchiveURLs"},
		},
		{
			name: "no network specified; passphrase not supplied",
			config: Config{
				HistoryArchiveURLs: []string{"HistoryArchiveURLs"},
			},
			errStr: fmt.Sprintf(errorMsgConfig, NetworkPassphraseFlagName),
		},
		{
			name: "no network specified; history archive urls not supplied",
			config: Config{
				NetworkPassphrase: "NetworkPassphrase",
			},
			errStr: fmt.Sprintf(errorMsgConfig, HistoryArchiveURLsFlagName),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := createCaptiveCoreConfigFromParameters(&tt.config)
			if tt.errStr == "" {
				assert.NoError(t, e)
				assert.Equal(t, tt.networkPassphrase, tt.config.NetworkPassphrase)
				assert.Equal(t, tt.historyArchiveURLs, tt.config.HistoryArchiveURLs)
			} else {
				require.Error(t, e)
				assert.Equal(t, tt.errStr, e.Error())
			}
		})
	}
}

func TestEnvironmentVariables(t *testing.T) {
	environmentVars := map[string]string{
		"INGEST":                        "false",
		"HISTORY_ARCHIVE_URLS":          "http://localhost:1570",
		"DATABASE_URL":                  "postgres://postgres@localhost/test_332cb65e6b00?sslmode=disable&timezone=UTC",
		"STELLAR_CORE_URL":              "http://localhost:11626",
		"NETWORK_PASSPHRASE":            "Standalone Network ; February 2017",
		"APPLY_MIGRATIONS":              "true",
		"ENABLE_CAPTIVE_CORE_INGESTION": "false",
		"CHECKPOINT_FREQUENCY":          "8",
		"MAX_DB_CONNECTIONS":            "50",
		"ADMIN_PORT":                    "6060",
		"PORT":                          "8001",
		"CAPTIVE_CORE_BINARY_PATH":      os.Getenv("HORIZON_INTEGRATION_TESTS_CAPTIVE_CORE_BIN"),
		"CAPTIVE_CORE_CONFIG_PATH":      "../docker/captive-core-classic-integration-tests.cfg",
		"CAPTIVE_CORE_USE_DB":           "true",
	}
	envManager := NewEnvironmentManager()
	if err := envManager.InitializeEnvironmentVariables(environmentVars); err != nil {
		fmt.Println(err)
	}
	config, flags := Flags()
	horizonCmd := &cobra.Command{
		Use:           "horizon",
		Short:         "Client-facing api server for the Stellar network",
		SilenceErrors: true,
		SilenceUsage:  true,
		Long:          "Client-facing API server for the Stellar network.",
	}
	if err := flags.Init(horizonCmd); err != nil {
		fmt.Println(err)
	}
	if err := ApplyFlags(config, flags, ApplyOptions{RequireCaptiveCoreConfig: true, AlwaysIngest: false}); err != nil {
		fmt.Println(err)
	}
	assert.Equal(t, config.Ingest, false)
	assert.Equal(t, config.HistoryArchiveURLs, []string{"http://localhost:1570"})
	assert.Equal(t, config.DatabaseURL, "postgres://postgres@localhost/test_332cb65e6b00?sslmode=disable&timezone=UTC")
	assert.Equal(t, config.StellarCoreURL, "http://localhost:11626")
	assert.Equal(t, config.NetworkPassphrase, "Standalone Network ; February 2017")
	assert.Equal(t, config.ApplyMigrations, true)
	assert.Equal(t, config.EnableCaptiveCoreIngestion, false)
	assert.Equal(t, config.CheckpointFrequency, uint32(8))
	assert.Equal(t, config.MaxDBConnections, 50)
	assert.Equal(t, config.AdminPort, uint(6060))
	assert.Equal(t, config.Port, uint(8001))
	assert.Equal(t, config.CaptiveCoreBinaryPath, os.Getenv("HORIZON_INTEGRATION_TESTS_CAPTIVE_CORE_BIN"))
	assert.Equal(t, config.CaptiveCoreConfigPath, "../docker/captive-core-classic-integration-tests.cfg")
	assert.Equal(t, config.CaptiveCoreConfigUseDB, true)
	envManager.Restore()
}
