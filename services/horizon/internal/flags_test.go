package horizon

import (
	"fmt"
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
			name: "testnet default config",
			config: Config{Network: StellarTestnet,
				CaptiveCoreBinaryPath: "/path/to/captive-core/binary",
			},
			networkPassphrase:  TestnetConf.NetworkPassphrase,
			historyArchiveURLs: TestnetConf.HistoryArchiveURLs,
		},
		{
			name: "pubnet default config",
			config: Config{Network: StellarPubnet,
				CaptiveCoreBinaryPath: "/path/to/captive-core/binary",
			},
			networkPassphrase:  PubnetConf.NetworkPassphrase,
			historyArchiveURLs: PubnetConf.HistoryArchiveURLs,
		},
		{
			name: "testnet validation; history archive urls supplied",
			config: Config{Network: StellarTestnet,
				HistoryArchiveURLs:    []string{"network history archive urls supplied"},
				CaptiveCoreBinaryPath: "/path/to/captive-core/binary",
			},
			errStr: fmt.Sprintf(errorMsgDefaultConfig, HistoryArchiveURLsFlagName),
		},
		{
			name: "pubnet validation; history archive urls supplied",
			config: Config{Network: StellarPubnet,
				HistoryArchiveURLs:    []string{"network history archive urls supplied"},
				CaptiveCoreBinaryPath: "/path/to/captive-core/binary",
			},
			errStr: fmt.Sprintf(errorMsgDefaultConfig, HistoryArchiveURLsFlagName),
		},
		{
			name: "testnet validation; network passphrase supplied",
			config: Config{Network: StellarTestnet,
				NetworkPassphrase:     "network passphrase supplied",
				HistoryArchiveURLs:    []string{},
				CaptiveCoreBinaryPath: "/path/to/captive-core/binary",
			},
			errStr: fmt.Sprintf(errorMsgDefaultConfig, NetworkPassphraseFlagName),
		},
		{
			name: "pubnet validation; network passphrase supplied",
			config: Config{Network: StellarPubnet,
				NetworkPassphrase:     "pubnet network passphrase supplied",
				HistoryArchiveURLs:    []string{},
				CaptiveCoreBinaryPath: "/path/to/captive-core/binary",
			},
			errStr: fmt.Sprintf(errorMsgDefaultConfig, NetworkPassphraseFlagName),
		},
		{
			name: "unknown network specified",
			config: Config{Network: "unknown",
				NetworkPassphrase:     "",
				HistoryArchiveURLs:    []string{},
				CaptiveCoreBinaryPath: "/path/to/captive-core/binary",
			},
			errStr: "no default configuration found for network unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := setCaptiveCoreConfiguration(&tt.config,
				ApplyOptions{RequireCaptiveCoreFullConfig: true})
			if tt.errStr == "" {
				assert.NoError(t, e)
				assert.Equal(t, tt.networkPassphrase, tt.config.NetworkPassphrase)
				assert.Equal(t, tt.historyArchiveURLs, tt.config.HistoryArchiveURLs)
				assert.Equal(t, tt.networkPassphrase, tt.config.CaptiveCoreTomlParams.NetworkPassphrase)
				assert.Equal(t, tt.historyArchiveURLs, tt.config.CaptiveCoreTomlParams.HistoryArchiveURLs)
			} else {
				assert.Equal(t, tt.errStr, e.Error())
			}
		})
	}
}

func Test_createCaptiveCoreConfig(t *testing.T) {

	var errorMsgConfig = "%s must be set"
	tests := []struct {
		name                     string
		requireCaptiveCoreConfig bool
		config                   Config
		networkPassphrase        string
		historyArchiveURLs       []string
		errStr                   string
	}{
		{
			name:                     "no network specified; valid parameters",
			requireCaptiveCoreConfig: true,
			config: Config{
				NetworkPassphrase:     PubnetConf.NetworkPassphrase,
				HistoryArchiveURLs:    PubnetConf.HistoryArchiveURLs,
				CaptiveCoreConfigPath: "configs/captive-core-pubnet.cfg",
				CaptiveCoreBinaryPath: "/path/to/captive-core/binary",
			},
			networkPassphrase:  PubnetConf.NetworkPassphrase,
			historyArchiveURLs: PubnetConf.HistoryArchiveURLs,
		},
		{
			name:                     "no network specified; passphrase not supplied",
			requireCaptiveCoreConfig: true,
			config: Config{
				HistoryArchiveURLs:    []string{"HistoryArchiveURLs"},
				CaptiveCoreBinaryPath: "/path/to/captive-core/binary",
			},
			errStr: fmt.Sprintf(errorMsgConfig, NetworkPassphraseFlagName),
		},
		{
			name:                     "no network specified; history archive urls not supplied",
			requireCaptiveCoreConfig: true,
			config: Config{
				NetworkPassphrase:     "NetworkPassphrase",
				CaptiveCoreBinaryPath: "/path/to/captive-core/binary",
			},
			errStr: fmt.Sprintf(errorMsgConfig, HistoryArchiveURLsFlagName),
		},
		{
			name:                     "no network specified; captive-core-config-path not supplied",
			requireCaptiveCoreConfig: true,
			config: Config{
				NetworkPassphrase:     PubnetConf.NetworkPassphrase,
				HistoryArchiveURLs:    PubnetConf.HistoryArchiveURLs,
				CaptiveCoreBinaryPath: "/path/to/captive-core/binary",
			},
			errStr: fmt.Sprintf("invalid config: captive core requires that --%s is set or "+
				"you can set the --%s parameter to use the default captive core config", CaptiveCoreConfigPathName, NetworkFlagName),
		},
		{
			name:                     "no network specified; captive-core-config-path invalid file",
			requireCaptiveCoreConfig: true,
			config: Config{
				NetworkPassphrase:     PubnetConf.NetworkPassphrase,
				HistoryArchiveURLs:    PubnetConf.HistoryArchiveURLs,
				CaptiveCoreConfigPath: "xyz.cfg",
				CaptiveCoreBinaryPath: "/path/to/captive-core/binary",
			},
			errStr: "invalid captive core toml file: could not load toml path:" +
				" open xyz.cfg: no such file or directory",
		},
		{
			name:                     "no network specified; captive-core-config-path incorrect config",
			requireCaptiveCoreConfig: true,
			config: Config{
				NetworkPassphrase:     PubnetConf.NetworkPassphrase,
				HistoryArchiveURLs:    PubnetConf.HistoryArchiveURLs,
				CaptiveCoreConfigPath: "configs/captive-core-testnet.cfg",
				CaptiveCoreBinaryPath: "/path/to/captive-core/binary",
			},
			errStr: fmt.Sprintf("invalid captive core toml file: invalid captive core toml: "+
				"NETWORK_PASSPHRASE in captive core config file: %s does not match Horizon "+
				"network-passphrase flag: %s", TestnetConf.NetworkPassphrase, PubnetConf.NetworkPassphrase),
		},
		{
			name:                     "no network specified; captive-core-config not required",
			requireCaptiveCoreConfig: false,
			config: Config{
				NetworkPassphrase:     PubnetConf.NetworkPassphrase,
				HistoryArchiveURLs:    PubnetConf.HistoryArchiveURLs,
				CaptiveCoreBinaryPath: "/path/to/captive-core/binary",
			},
			networkPassphrase:  PubnetConf.NetworkPassphrase,
			historyArchiveURLs: PubnetConf.HistoryArchiveURLs,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := setCaptiveCoreConfiguration(&tt.config,
				ApplyOptions{RequireCaptiveCoreFullConfig: tt.requireCaptiveCoreConfig})
			if tt.errStr == "" {
				assert.NoError(t, e)
				assert.Equal(t, tt.networkPassphrase, tt.config.NetworkPassphrase)
				assert.Equal(t, tt.historyArchiveURLs, tt.config.HistoryArchiveURLs)
				assert.Equal(t, tt.networkPassphrase, tt.config.CaptiveCoreTomlParams.NetworkPassphrase)
				assert.Equal(t, tt.historyArchiveURLs, tt.config.CaptiveCoreTomlParams.HistoryArchiveURLs)
			} else {
				require.Error(t, e)
				assert.Equal(t, tt.errStr, e.Error())
			}
		})
	}
}
