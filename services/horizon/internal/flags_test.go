package horizon

import (
	"fmt"
	"testing"

	support "github.com/stellar/go/support/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_createCaptiveCoreDefaultConfig(t *testing.T) {

	var errorMsgDefaultConfig = "invalid config: %s not allowed with %s network"
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
			networkPassphrase:  testnetConf.networkPassphrase,
			historyArchiveURLs: testnetConf.historyArchiveURLs,
		},
		{
			name:               "pubnet default config",
			config:             Config{Network: StellarPubnet},
			networkPassphrase:  pubnetConf.networkPassphrase,
			historyArchiveURLs: pubnetConf.historyArchiveURLs,
		},
		{
			name: "testnet validation; history archive urls supplied",
			config: Config{Network: StellarTestnet,
				HistoryArchiveURLs: []string{"network history archive urls supplied"},
			},
			errStr: fmt.Sprintf(errorMsgDefaultConfig, HistoryArchiveURLsFlagName, StellarTestnet),
		},
		{
			name: "pubnet validation; history archive urls supplied",
			config: Config{Network: StellarPubnet,
				HistoryArchiveURLs: []string{"network history archive urls supplied"},
			},
			errStr: fmt.Sprintf(errorMsgDefaultConfig, HistoryArchiveURLsFlagName, StellarPubnet),
		},
		{
			name: "testnet validation; network passphrase supplied",
			config: Config{Network: StellarTestnet,
				NetworkPassphrase:  "network passphrase supplied",
				HistoryArchiveURLs: []string{},
			},
			errStr: fmt.Sprintf(errorMsgDefaultConfig, NetworkPassphraseFlagName, StellarTestnet),
		},
		{
			name: "pubnet validation; network passphrase supplied",
			config: Config{Network: StellarPubnet,
				NetworkPassphrase:  "pubnet network passphrase supplied",
				HistoryArchiveURLs: []string{},
			},
			errStr: fmt.Sprintf(errorMsgDefaultConfig, NetworkPassphraseFlagName, StellarPubnet),
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

func TestInvalidConfigOutputFromApplyFlags(t *testing.T) {
	_, flags := Flags()

	tests := []struct {
		name    string
		config  Config
		flags   support.ConfigOptions
		options ApplyOptions
		errStr  string
	}{
		{
			name: "disable-tx-sub set to false and no network and ingestion params set",
			config: Config{
				NetworkPassphrase:  "",
				HistoryArchiveURLs: []string{},
				DisableTxSub:       false,
			},
			flags:   flags,
			options: ApplyOptions{RequireCaptiveCoreConfig: true, AlwaysIngest: false},
			errStr: "invalid config: --disable-tx-sub set to false but ingestion has been disabled." +
				"Set INGEST=true to enable transaction submission functionality",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ApplyFlags(&tt.config, tt.flags, tt.options)
			assert.Error(t, e, tt.errStr)
		})
	}
}
