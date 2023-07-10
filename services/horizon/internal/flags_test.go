package horizon

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_createCaptiveCoreDefaultNetworkConfig(t *testing.T) {

	var errorMsgDefaultConfig = "error generating default captive core config. invalid config: %s not allowed with %s network"
	var errorMsgConfig = "error generating captive core config. %s must be set"
	tests := []struct {
		name                     string
		config                   Config
		networkPassphrase        string
		historyArchiveURLs       []string
		usingDefaultPubnetConfig bool
		errStr                   string
	}{
		{
			name:               "testnet default config",
			config:             Config{Network: StellarTestnet},
			networkPassphrase:  testnetConf.networkPassphrase,
			historyArchiveURLs: testnetConf.historyArchiveURLs,
		},
		{
			name:                     "pubnet default config",
			config:                   Config{Network: StellarPubnet},
			networkPassphrase:        pubnetConf.networkPassphrase,
			historyArchiveURLs:       pubnetConf.historyArchiveURLs,
			usingDefaultPubnetConfig: true,
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
			errStr: fmt.Sprintf("error generating default captive core config. " +
				"no default configuration found for network unknown"),
		},
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
			e := setCaptiveCoreConfiguration(&tt.config)
			if tt.errStr == "" {
				assert.NoError(t, e)
				assert.Equal(t, tt.usingDefaultPubnetConfig, tt.config.UsingDefaultPubnetConfig)
				assert.Equal(t, tt.networkPassphrase, tt.config.NetworkPassphrase)
				assert.Equal(t, tt.historyArchiveURLs, tt.config.HistoryArchiveURLs)
			} else {
				assert.Equal(t, tt.errStr, e.Error())
			}
		})
	}
}
