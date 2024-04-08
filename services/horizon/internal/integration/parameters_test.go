//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite
package integration

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	stdLog "log"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stellar/go/services/horizon/internal/paths"
	"github.com/stellar/go/services/horizon/internal/simplepath"

	horizoncmd "github.com/stellar/go/services/horizon/cmd"
	horizon "github.com/stellar/go/services/horizon/internal"
	"github.com/stellar/go/services/horizon/internal/test/integration"

	"github.com/stretchr/testify/assert"
)

var defaultCaptiveCoreParameters = map[string]string{
	horizon.StellarCoreBinaryPathName: os.Getenv("CAPTIVE_CORE_BIN"),
	horizon.StellarCoreURLFlagName:    "",
}

var networkParamArgs = map[string]string{
	horizon.CaptiveCoreConfigPathName:   "",
	horizon.CaptiveCoreHTTPPortFlagName: "",
	horizon.StellarCoreBinaryPathName:   "",
	horizon.StellarCoreURLFlagName:      "",
	horizon.HistoryArchiveURLsFlagName:  "",
	horizon.NetworkPassphraseFlagName:   "",
}

const (
	SimpleCaptiveCoreToml = `
		PEER_PORT=11725
		ARTIFICIALLY_ACCELERATE_TIME_FOR_TESTING=true

		UNSAFE_QUORUM=true
		FAILURE_SAFETY=0

		[[VALIDATORS]]
		NAME="local_core"
		HOME_DOMAIN="core.local"
		PUBLIC_KEY="GD5KD2KEZJIGTC63IGW6UMUSMVUVG5IHG64HUTFWCHVZH2N2IBOQN7PS"
		ADDRESS="localhost"
		QUALITY="MEDIUM"`

	StellarCoreURL = "http://localhost:11626"
)

var (
	CaptiveCoreConfigErrMsg = "error generating captive core configuration: invalid config: "
)

// Ensures that BUCKET_DIR_PATH is not an allowed value for Captive Core.
func TestBucketDirDisallowed(t *testing.T) {
	// This is a bit of a hacky workaround.
	//
	// In CI, we run our integration tests twice: once with Captive Core
	// enabled, and once without. *These* tests only run with Captive Core
	// configured properly (specifically, w/ the CAPTIVE_CORE_BIN envvar set).
	if !integration.RunWithCaptiveCore {
		t.Skip()
	}

	config := `BUCKET_DIR_PATH="/tmp"
		` + SimpleCaptiveCoreToml

	confName, _, cleanup := createCaptiveCoreConfig(config)
	defer cleanup()
	testConfig := integration.GetTestConfig()
	testConfig.HorizonIngestParameters = map[string]string{
		horizon.CaptiveCoreConfigPathName: confName,
		horizon.StellarCoreBinaryPathName: os.Getenv("CAPTIVE_CORE_BIN"),
	}
	test := integration.NewTest(t, *testConfig)
	err := test.StartHorizon()
	assert.Equal(t, err.Error(), integration.HorizonInitErrStr+": error generating captive core configuration:"+
		" invalid captive core toml file: could not unmarshal captive core toml: setting BUCKET_DIR_PATH is disallowed"+
		" for Captive Core, use CAPTIVE_CORE_STORAGE_PATH instead")
	time.Sleep(1 * time.Second)
	test.StopHorizon()
	test.Shutdown()
}

func TestEnvironmentPreserved(t *testing.T) {
	// Who tests the tests? This test.
	//
	// It ensures that the global OS environmental variables are preserved after
	// running an integration test.

	// Note that we ALSO need to make sure we don't modify parent env state.
	value, isSet := os.LookupEnv("STELLAR_CORE_URL")
	defer func() {
		if isSet {
			_ = os.Setenv("STELLAR_CORE_URL", value)
		} else {
			_ = os.Unsetenv("STELLAR_CORE_URL")
		}
	}()

	err := os.Setenv("STELLAR_CORE_URL", "original value")
	assert.NoError(t, err)

	testConfig := integration.GetTestConfig()
	testConfig.HorizonEnvironment = map[string]string{
		"STELLAR_CORE_URL": StellarCoreURL,
	}
	test := integration.NewTest(t, *testConfig)

	err = test.StartHorizon()
	assert.NoError(t, err)
	test.WaitForHorizon()

	envValue := os.Getenv("STELLAR_CORE_URL")
	assert.Equal(t, StellarCoreURL, envValue)

	test.Shutdown()

	envValue = os.Getenv("STELLAR_CORE_URL")
	assert.Equal(t, "original value", envValue)
}

// TestInvalidNetworkParameters Ensure that Horizon returns an error when
// using NETWORK environment variables, history archive urls or network passphrase
// parameters are also set.
func TestInvalidNetworkParameters(t *testing.T) {
	if !integration.RunWithCaptiveCore {
		t.Skip()
	}

	var captiveCoreConfigErrMsg = integration.HorizonInitErrStr + ": error generating captive " +
		"core configuration: invalid config: %s parameter not allowed with the %s parameter"
	testCases := []struct {
		name         string
		errMsg       string
		networkValue string
		param        string
	}{
		{
			name: "history archive urls validation",
			errMsg: fmt.Sprintf(captiveCoreConfigErrMsg, horizon.HistoryArchiveURLsFlagName,
				horizon.NetworkFlagName),
			networkValue: horizon.StellarPubnet,
			param:        horizon.HistoryArchiveURLsFlagName,
		},
		{
			name: "network-passphrase validation",
			errMsg: fmt.Sprintf(captiveCoreConfigErrMsg, horizon.NetworkPassphraseFlagName,
				horizon.NetworkFlagName),
			networkValue: horizon.StellarTestnet,
			param:        horizon.NetworkPassphraseFlagName,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			localParams := integration.MergeMaps(networkParamArgs, map[string]string{
				horizon.NetworkFlagName: testCase.networkValue,
				testCase.param:          testCase.param, // set any value
			})
			testConfig := integration.GetTestConfig()
			testConfig.SkipCoreContainerCreation = true
			testConfig.HorizonIngestParameters = localParams
			test := integration.NewTest(t, *testConfig)
			err := test.StartHorizon()
			// Adding sleep as a workaround for the race condition in the ingestion system.
			// https://github.com/stellar/go/issues/5005
			time.Sleep(2 * time.Second)
			assert.Equal(t, testCase.errMsg, err.Error())
			test.Shutdown()
		})
	}
}

// TestNetworkParameter Ensure that Horizon successfully starts the captive-core
// subprocess using the default configuration when --network [testnet|pubnet]
// commandline parameter.
//
// In integration tests, we start Horizon and stellar-core containers in standalone mode
// simultaneously. We usually wait for Horizon to begin ingesting to verify the test's
// success. However, for "pubnet" or "testnet," we can not wait for Horizon to catch up,
// so we skip starting stellar-core containers.
func TestNetworkParameter(t *testing.T) {
	if !integration.RunWithCaptiveCore {
		t.Skip()
	}
	testCases := []struct {
		networkValue       string
		networkPassphrase  string
		historyArchiveURLs []string
	}{
		{
			networkValue:       horizon.StellarTestnet,
			networkPassphrase:  horizon.TestnetConf.NetworkPassphrase,
			historyArchiveURLs: horizon.TestnetConf.HistoryArchiveURLs,
		},
		{
			networkValue:       horizon.StellarPubnet,
			networkPassphrase:  horizon.PubnetConf.NetworkPassphrase,
			historyArchiveURLs: horizon.PubnetConf.HistoryArchiveURLs,
		},
	}
	for _, tt := range testCases {
		t.Run(fmt.Sprintf("NETWORK parameter %s", tt.networkValue), func(t *testing.T) {
			localParams := integration.MergeMaps(networkParamArgs, map[string]string{
				horizon.NetworkFlagName: tt.networkValue,
			})
			testConfig := integration.GetTestConfig()
			testConfig.SkipCoreContainerCreation = true
			testConfig.HorizonIngestParameters = localParams
			test := integration.NewTest(t, *testConfig)
			err := test.StartHorizon()
			// Adding sleep as a workaround for the race condition in the ingestion system.
			// https://github.com/stellar/go/issues/5005
			time.Sleep(2 * time.Second)
			assert.NoError(t, err)
			assert.Equal(t, test.GetHorizonIngestConfig().HistoryArchiveURLs, tt.historyArchiveURLs)
			assert.Equal(t, test.GetHorizonIngestConfig().NetworkPassphrase, tt.networkPassphrase)

			test.Shutdown()
		})
	}
}

// TestNetworkEnvironmentVariable Ensure that Horizon successfully starts the captive-core
// subprocess using the default configuration when the NETWORK environment variable is set
// to either pubnet or testnet.
//
// In integration tests, we start Horizon and stellar-core containers in standalone mode
// simultaneously. We usually wait for Horizon to begin ingesting to verify the test's
// success. However, for "pubnet" or "testnet," we can not wait for Horizon to catch up,
// so we skip starting stellar-core containers.
func TestNetworkEnvironmentVariable(t *testing.T) {
	if !integration.RunWithCaptiveCore {
		t.Skip()
	}
	testCases := []string{
		horizon.StellarPubnet,
		horizon.StellarTestnet,
	}

	for _, networkValue := range testCases {
		t.Run(fmt.Sprintf("NETWORK environment variable %s", networkValue), func(t *testing.T) {
			value, isSet := os.LookupEnv("NETWORK")
			defer func() {
				if isSet {
					_ = os.Setenv("NETWORK", value)
				} else {
					_ = os.Unsetenv("NETWORK")
				}
			}()

			testConfig := integration.GetTestConfig()
			testConfig.SkipCoreContainerCreation = true
			testConfig.HorizonIngestParameters = networkParamArgs
			testConfig.HorizonEnvironment = map[string]string{"NETWORK": networkValue}
			test := integration.NewTest(t, *testConfig)
			err := test.StartHorizon()
			// Adding sleep here as a workaround for the race condition in the ingestion system.
			// More details can be found at https://github.com/stellar/go/issues/5005
			time.Sleep(2 * time.Second)
			assert.NoError(t, err)
			test.Shutdown()
		})
	}
}

// Ensures that the filesystem ends up in the correct state with Captive Core.
func TestCaptiveCoreConfigFilesystemState(t *testing.T) {
	if !integration.RunWithCaptiveCore {
		t.Skip() // explained above
	}

	confName, storagePath, cleanup := createCaptiveCoreConfig(SimpleCaptiveCoreToml)
	defer cleanup()

	localParams := integration.MergeMaps(defaultCaptiveCoreParameters, map[string]string{
		"captive-core-storage-path":       storagePath,
		horizon.CaptiveCoreConfigPathName: confName,
	})
	testConfig := integration.GetTestConfig()
	testConfig.HorizonIngestParameters = localParams
	test := integration.NewTest(t, *testConfig)

	err := test.StartHorizon()
	assert.NoError(t, err)
	test.WaitForHorizon()

	t.Run("disk state", func(t *testing.T) {
		validateCaptiveCoreDiskState(test, storagePath)
	})

	t.Run("no bucket dir", func(t *testing.T) {
		validateNoBucketDirPath(test, storagePath)
	})
}

func TestMaxAssetsForPathRequests(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		test := integration.NewTest(t, *integration.GetTestConfig())
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.WaitForHorizon()
		assert.Equal(t, test.HorizonIngest().Config().MaxAssetsPerPathRequest, 15)
		test.Shutdown()
	})
	t.Run("set to 2", func(t *testing.T) {
		testConfig := integration.GetTestConfig()
		testConfig.HorizonIngestParameters = map[string]string{"max-assets-per-path-request": "2"}
		test := integration.NewTest(t, *testConfig)
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.WaitForHorizon()
		assert.Equal(t, test.HorizonIngest().Config().MaxAssetsPerPathRequest, 2)
		test.Shutdown()
	})
}

func TestMaxPathFindingRequests(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		test := integration.NewTest(t, *integration.GetTestConfig())
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.WaitForHorizon()
		assert.Equal(t, test.HorizonIngest().Config().MaxPathFindingRequests, uint(0))
		_, ok := test.HorizonIngest().Paths().(simplepath.InMemoryFinder)
		assert.True(t, ok)
		test.Shutdown()
	})
	t.Run("set to 5", func(t *testing.T) {
		testConfig := integration.GetTestConfig()
		testConfig.HorizonIngestParameters = map[string]string{"max-path-finding-requests": "5"}
		test := integration.NewTest(t, *testConfig)
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.WaitForHorizon()
		assert.Equal(t, test.HorizonIngest().Config().MaxPathFindingRequests, uint(5))
		finder, ok := test.HorizonIngest().Paths().(*paths.RateLimitedFinder)
		assert.True(t, ok)
		assert.Equal(t, finder.Limit(), 5)
		test.Shutdown()
	})
}

func TestDisablePathFinding(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		test := integration.NewTest(t, *integration.GetTestConfig())
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.WaitForHorizon()
		assert.Equal(t, test.HorizonIngest().Config().MaxPathFindingRequests, uint(0))
		_, ok := test.HorizonIngest().Paths().(simplepath.InMemoryFinder)
		assert.True(t, ok)
		test.Shutdown()
	})
	t.Run("set to true", func(t *testing.T) {
		testConfig := integration.GetTestConfig()
		testConfig.HorizonIngestParameters = map[string]string{"disable-path-finding": "true"}
		test := integration.NewTest(t, *testConfig)
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.WaitForHorizon()
		assert.Nil(t, test.HorizonIngest().Paths())
		test.Shutdown()
	})
}

func TestIngestionFilteringAlwaysDefaultingToTrue(t *testing.T) {
	t.Run("ingestion filtering flag set to default value", func(t *testing.T) {
		test := integration.NewTest(t, *integration.GetTestConfig())
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.WaitForHorizon()
		assert.Equal(t, test.HorizonIngest().Config().EnableIngestionFiltering, true)
		test.Shutdown()
	})
	t.Run("ingestion filtering flag set to false", func(t *testing.T) {
		testConfig := integration.GetTestConfig()
		testConfig.HorizonIngestParameters = map[string]string{"exp-enable-ingestion-filtering": "false"}
		test := integration.NewTest(t, *testConfig)
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.WaitForHorizon()
		assert.Equal(t, test.HorizonIngest().Config().EnableIngestionFiltering, true)
		test.Shutdown()
	})
}

func TestDisableTxSub(t *testing.T) {
	t.Run("require stellar-core-url when both DISABLE_TX_SUB=false and INGEST=false", func(t *testing.T) {
		localParams := integration.MergeMaps(networkParamArgs, map[string]string{
			horizon.NetworkFlagName:      "testnet",
			horizon.IngestFlagName:       "false",
			horizon.DisableTxSubFlagName: "false",
		})
		testConfig := integration.GetTestConfig()
		testConfig.HorizonIngestParameters = localParams
		testConfig.SkipCoreContainerCreation = true
		test := integration.NewTest(t, *testConfig)
		err := test.StartHorizon()
		assert.ErrorContains(t, err, "cannot initialize Horizon: flag --stellar-core-url cannot be empty")
		test.Shutdown()
	})
	t.Run("horizon starts successfully when DISABLE_TX_SUB=false, INGEST=false and stellar-core-url is provided", func(t *testing.T) {
		localParams := integration.MergeMaps(networkParamArgs, map[string]string{
			horizon.NetworkFlagName:        "testnet",
			horizon.IngestFlagName:         "false",
			horizon.DisableTxSubFlagName:   "false",
			horizon.StellarCoreURLFlagName: "http://localhost:11626",
		})
		testConfig := integration.GetTestConfig()
		testConfig.HorizonIngestParameters = localParams
		testConfig.SkipCoreContainerCreation = true
		test := integration.NewTest(t, *testConfig)
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.Shutdown()
	})
	t.Run("horizon starts successfully when DISABLE_TX_SUB=true and INGEST=true", func(t *testing.T) {
		testConfig := integration.GetTestConfig()
		testConfig.HorizonIngestParameters = map[string]string{
			"disable-tx-sub": "true",
			"ingest":         "true",
		}
		test := integration.NewTest(t, *testConfig)
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.WaitForHorizon()
		test.Shutdown()
	})
	t.Run("do not require stellar-core-url when both DISABLE_TX_SUB=true and INGEST=false", func(t *testing.T) {
		localParams := integration.MergeMaps(networkParamArgs, map[string]string{
			horizon.NetworkFlagName:      "testnet",
			horizon.IngestFlagName:       "false",
			horizon.DisableTxSubFlagName: "true",
		})
		testConfig := integration.GetTestConfig()
		testConfig.HorizonIngestParameters = localParams
		testConfig.SkipCoreContainerCreation = true
		test := integration.NewTest(t, *testConfig)
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.Shutdown()
	})
}

func TestDeprecatedOutputs(t *testing.T) {
	t.Run("deprecated output for ingestion filtering", func(t *testing.T) {
		originalStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w
		stdLog.SetOutput(os.Stderr)

		testConfig := integration.GetTestConfig()
		testConfig.HorizonIngestParameters = map[string]string{"exp-enable-ingestion-filtering": "false"}
		test := integration.NewTest(t, *testConfig)
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.WaitForHorizon()

		// Use a wait group to wait for the goroutine to finish before proceeding
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := w.Close(); err != nil {
				t.Errorf("Failed to close Stdout")
				return
			}
		}()

		outputBytes, _ := io.ReadAll(r)
		wg.Wait() // Wait for the goroutine to finish before proceeding
		_ = r.Close()
		os.Stderr = originalStderr

		assert.Contains(t, string(outputBytes), "DEPRECATED - No ingestion filter rules are defined by default, which equates to "+
			"no filtering of historical data. If you have never added filter rules to this deployment, then no further action is needed. "+
			"If you have defined ingestion filter rules previously but disabled filtering overall by setting the env variable EXP_ENABLE_INGESTION_FILTERING=false, "+
			"then you should now delete the filter rules using the Horizon Admin API to achieve the same no-filtering result. Remove usage of this variable in all cases.")
	})
	t.Run("deprecated output for command-line flags", func(t *testing.T) {
		originalStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w
		stdLog.SetOutput(os.Stderr)

		config, flags := horizon.Flags()

		horizonCmd := &cobra.Command{
			Use:           "horizon",
			Short:         "Client-facing api server for the Stellar network",
			SilenceErrors: true,
			SilenceUsage:  true,
			Long:          "Client-facing API server for the Stellar network.",
			RunE: func(cmd *cobra.Command, args []string) error {
				_, err := horizon.NewAppFromFlags(config, flags)
				if err != nil {
					return err
				}
				return nil
			},
		}

		horizonCmd.SetArgs([]string{"--disable-tx-sub=true"})
		if err := flags.Init(horizonCmd); err != nil {
			fmt.Println(err)
		}
		if err := horizonCmd.Execute(); err != nil {
			fmt.Println(err)
		}

		// Use a wait group to wait for the goroutine to finish before proceeding
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := w.Close(); err != nil {
				t.Errorf("Failed to close Stdout")
				return
			}
		}()

		outputBytes, _ := io.ReadAll(r)
		wg.Wait() // Wait for the goroutine to finish before proceeding
		_ = r.Close()
		os.Stderr = originalStderr

		assert.Contains(t, string(outputBytes), "DEPRECATED - the use of command-line flags: "+
			"[--disable-tx-sub], has been deprecated in favor of environment variables. Please consult our "+
			"Configuring section in the developer documentation on how to use them - "+
			"https://developers.stellar.org/docs/run-api-server/configuring")
	})
	t.Run("deprecated output for --captive-core-use-db", func(t *testing.T) {
		originalStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w
		stdLog.SetOutput(os.Stderr)

		testConfig := integration.GetTestConfig()
		testConfig.HorizonIngestParameters = map[string]string{"captive-core-use-db": "false"}
		test := integration.NewTest(t, *testConfig)
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.WaitForHorizon()

		// Use a wait group to wait for the goroutine to finish before proceeding
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := w.Close(); err != nil {
				t.Errorf("Failed to close Stdout")
				return
			}
		}()

		outputBytes, _ := io.ReadAll(r)
		wg.Wait() // Wait for the goroutine to finish before proceeding
		_ = r.Close()
		os.Stderr = originalStderr

		assert.Contains(t, string(outputBytes), "The usage of the flag --captive-core-use-db has been deprecated. "+
			"Setting it to false to achieve in-memory functionality on captive core will be removed in "+
			"future releases. We recommend removing usage of this flag now in preparation.")
	})
}

func TestGlobalFlagsOutput(t *testing.T) {

	// verify Help and Usage output from cli, both help and usage output follow the same
	// output rules of no globals when sub-comands exist, and only relevant globals
	// when down to leaf node command.

	dbParams := []string{"--max-db-connections", "--db-url"}
	// the space after '--ingest' is intentional to ensure correct matching behavior to
	// help output, as other flags also start with same prefix.
	apiParams := []string{"--port ", "--per-hour-rate-limit", "--ingest ", "sentry-dsn"}
	ingestionParams := []string{"--stellar-core-binary-path", "--history-archive-urls", "--ingest-state-verification-checkpoint-frequency"}
	allParams := append(apiParams, append(dbParams, ingestionParams...)...)

	testCases := []struct {
		horizonHelpCommand          []string
		helpPrintedGlobalParams     []string
		helpPrintedSubCommandParams []string
		helpSkippedGlobalParams     []string
	}{
		{
			horizonHelpCommand:          []string{"ingest", "trigger-state-rebuild", "-h"},
			helpPrintedGlobalParams:     dbParams,
			helpPrintedSubCommandParams: []string{},
			helpSkippedGlobalParams:     append(apiParams, ingestionParams...),
		},
		{
			horizonHelpCommand:          []string{"ingest", "verify-range", "-h"},
			helpPrintedGlobalParams:     append(dbParams, ingestionParams...),
			helpPrintedSubCommandParams: []string{"--verify-state", "--from"},
			helpSkippedGlobalParams:     apiParams,
		},
		{
			horizonHelpCommand:          []string{"db", "reingest", "range", "-h"},
			helpPrintedGlobalParams:     append(dbParams, ingestionParams...),
			helpPrintedSubCommandParams: []string{"--parallel-workers", "--force"},
			helpSkippedGlobalParams:     apiParams,
		},
		{
			horizonHelpCommand:          []string{"db", "reingest", "range"},
			helpPrintedGlobalParams:     append(dbParams, ingestionParams...),
			helpPrintedSubCommandParams: []string{"--parallel-workers", "--force"},
			helpSkippedGlobalParams:     apiParams,
		},
		{
			horizonHelpCommand:          []string{"db", "fill-gaps", "-h"},
			helpPrintedGlobalParams:     append(dbParams, ingestionParams...),
			helpPrintedSubCommandParams: []string{"--parallel-workers", "--force"},
			helpSkippedGlobalParams:     apiParams,
		},
		{
			horizonHelpCommand:          []string{"db", "migrate", "up", "-h"},
			helpPrintedGlobalParams:     dbParams,
			helpPrintedSubCommandParams: []string{},
			helpSkippedGlobalParams:     append(apiParams, ingestionParams...),
		},
		{
			horizonHelpCommand:          []string{"db", "-h"},
			helpPrintedGlobalParams:     []string{},
			helpPrintedSubCommandParams: []string{},
			helpSkippedGlobalParams:     allParams,
		},
		{
			horizonHelpCommand:          []string{"db"},
			helpPrintedGlobalParams:     []string{},
			helpPrintedSubCommandParams: []string{},
			helpSkippedGlobalParams:     allParams,
		},
		{
			horizonHelpCommand:          []string{"-h"},
			helpPrintedGlobalParams:     []string{},
			helpPrintedSubCommandParams: []string{},
			helpSkippedGlobalParams:     allParams,
		},
		{
			horizonHelpCommand:          []string{"db", "reingest", "-h"},
			helpPrintedGlobalParams:     []string{},
			helpPrintedSubCommandParams: []string{},
			helpSkippedGlobalParams:     apiParams,
		},
		{
			horizonHelpCommand:          []string{"db", "reingest"},
			helpPrintedGlobalParams:     []string{},
			helpPrintedSubCommandParams: []string{},
			helpSkippedGlobalParams:     apiParams,
		},
		{
			horizonHelpCommand:          []string{"serve", "-h"},
			helpPrintedGlobalParams:     allParams,
			helpPrintedSubCommandParams: []string{},
			helpSkippedGlobalParams:     []string{},
		},
		{
			horizonHelpCommand:          []string{"record-metrics", "-h"},
			helpPrintedGlobalParams:     []string{"--admin-port"},
			helpPrintedSubCommandParams: []string{},
			helpSkippedGlobalParams:     allParams,
		},
	}
	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("Horizon command line parameter %v", testCase.horizonHelpCommand), func(t *testing.T) {
			horizoncmd.RootCmd.SetArgs(testCase.horizonHelpCommand)
			var writer io.Writer = &bytes.Buffer{}
			horizoncmd.RootCmd.SetOutput(writer)
			horizoncmd.RootCmd.Execute()

			output := writer.(*bytes.Buffer).String()
			for _, requiredParam := range testCase.helpPrintedSubCommandParams {
				assert.Contains(t, output, requiredParam, testCase.horizonHelpCommand)
			}
			for _, requiredParam := range testCase.helpPrintedGlobalParams {
				assert.Contains(t, output, requiredParam, testCase.horizonHelpCommand)
			}
			for _, skippedParam := range testCase.helpSkippedGlobalParams {
				assert.NotContains(t, output, skippedParam, testCase.horizonHelpCommand)
			}
		})
	}
}

// validateNoBucketDirPath ensures the Stellar Core auto-generated configuration
// file doesn't contain the BUCKET_DIR_PATH entry, which is forbidden when using
// Captive Core.
//
// Pass "rootDirectory" set to whatever it is you pass to
// "--captive-core-storage-path".
func validateNoBucketDirPath(itest *integration.Test, rootDir string) {
	tt := assert.New(itest.CurrentTest())

	coreConf := path.Join(rootDir, "captive-core", "stellar-core.conf")
	tt.FileExists(coreConf)

	result, err := ioutil.ReadFile(coreConf)
	tt.NoError(err)

	bucketPathSet := strings.Contains(string(result), "BUCKET_DIR_PATH")
	tt.False(bucketPathSet)
}

// validateCaptiveCoreDiskState ensures that running Captive Core creates a
// sensible directory structure.
//
// Pass "rootDirectory" set to whatever it is you pass to
// "--captive-core-storage-path".
func validateCaptiveCoreDiskState(itest *integration.Test, rootDir string) {
	tt := assert.New(itest.CurrentTest())

	storageDir := path.Join(rootDir, "captive-core")
	coreConf := path.Join(storageDir, "stellar-core.conf")

	tt.DirExists(rootDir)
	tt.DirExists(storageDir)
	tt.FileExists(coreConf)
}

// createCaptiveCoreConfig will create a temporary TOML config with the
// specified contents as well as a temporary storage directory. You should
// `defer` the returned function to clean these up when you're done.
func createCaptiveCoreConfig(contents string) (string, string, func()) {
	tomlFile, err := ioutil.TempFile("", "captive-core-test-*.toml")
	defer tomlFile.Close()
	if err != nil {
		panic(err)
	}

	_, err = tomlFile.WriteString(contents)
	if err != nil {
		panic(err)
	}

	storagePath, err := os.MkdirTemp("", "captive-core-test-*-storage")
	if err != nil {
		panic(err)
	}

	filename := tomlFile.Name()
	return filename, storagePath, func() {
		os.Remove(filename)
		os.RemoveAll(storagePath)
	}
}
