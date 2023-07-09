package horizon

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_createCaptiveCoreDefaultNetworkConfig(t *testing.T) {

	var errStr = "invalid config: %s not allowed with %s network"
	tests := []struct {
		name               string
		config             Config
		networkPassphrase  string
		historyArchiveURLs []string
		configFile         string
		errStr             string
	}{
		{
			name: "testnet default config",
			config: Config{Network: StellarTestnet,
				HistoryArchiveURLs: []string{""},
			},
			networkPassphrase:  testnetConf.networkPassphrase,
			historyArchiveURLs: testnetConf.historyArchiveURLs,
			configFile:         testnetConf.configFileName,
			errStr:             "",
		},
		{
			name: "pubnet default config",
			config: Config{Network: StellarPubnet,
				HistoryArchiveURLs: []string{""},
			},
			networkPassphrase:  pubnetConf.networkPassphrase,
			historyArchiveURLs: pubnetConf.historyArchiveURLs,
			configFile:         pubnetConf.configFileName,
			errStr:             "",
		},
		{
			name: "testnet validation; history archive urls supplied",
			config: Config{Network: StellarTestnet,
				HistoryArchiveURLs: []string{"network history archive urls supplied"},
			},
			errStr: fmt.Sprintf(errStr, HistoryArchiveURLsFlagName, StellarTestnet),
		},
		{
			name: "pubnet validation; history archive urls supplied",
			config: Config{Network: StellarPubnet,
				HistoryArchiveURLs: []string{"network history archive urls supplied"},
			},
			errStr: fmt.Sprintf(errStr, HistoryArchiveURLsFlagName, StellarPubnet),
		},
		{
			name: "testnet validation; network passphrase supplied",
			config: Config{Network: StellarTestnet,
				NetworkPassphrase:  "network passphrase supplied",
				HistoryArchiveURLs: []string{},
			},
			errStr: fmt.Sprintf(errStr, NetworkPassphraseFlagName, StellarTestnet),
		},
		{
			name: "pubnet validation; network passphrase supplied",
			config: Config{Network: StellarPubnet,
				NetworkPassphrase:  "pubnet network passphrase supplied",
				HistoryArchiveURLs: []string{},
			},
			errStr: fmt.Sprintf(errStr, NetworkPassphraseFlagName, StellarPubnet),
		},
		{
			name: "unknown network specified",
			config: Config{Network: "unknown",
				NetworkPassphrase:  "",
				HistoryArchiveURLs: []string{},
			},
			errStr: fmt.Sprintf("error configuring default settings for network unknown." +
				" no default configuration found for network unknown"),
		},
		{
			name: "no network specified",
			config: Config{
				NetworkPassphrase:  "NetworkPassphrase",
				HistoryArchiveURLs: []string{"HistoryArchiveURLs"},
			},
			networkPassphrase:  "NetworkPassphrase",
			historyArchiveURLs: []string{"HistoryArchiveURLs"},
			errStr:             "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := setCaptiveCoreConfiguration(&tt.config)
			if tt.errStr == "" {
				assert.NoError(t, e)
				assert.Equal(t, tt.configFile, tt.config.CaptiveCoreConfigPath)
				assert.Equal(t, tt.networkPassphrase, tt.config.NetworkPassphrase)
				assert.Equal(t, tt.historyArchiveURLs, tt.config.HistoryArchiveURLs)
			} else {
				assert.Equal(t, tt.errStr, e.Error())
			}
		})
	}
}
