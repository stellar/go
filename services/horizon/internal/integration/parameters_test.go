//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite
package integration

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	stdLog "log"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/stellar/go/services/horizon/internal/paths"
	"github.com/stellar/go/services/horizon/internal/simplepath"

	horizon "github.com/stellar/go/services/horizon/internal"
	"github.com/stellar/go/services/horizon/internal/test/integration"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var defaultCaptiveCoreParameters = map[string]string{
	horizon.StellarCoreBinaryPathName: os.Getenv("CAPTIVE_CORE_BIN"),
	horizon.StellarCoreURLFlagName:    "",
	horizon.StellarCoreDBURLFlagName:  "",
}

var networkParamArgs = map[string]string{
	horizon.EnableCaptiveCoreIngestionFlagName: "true",
	horizon.CaptiveCoreConfigPathName:          "",
	horizon.CaptiveCoreHTTPPortFlagName:        "",
	horizon.StellarCoreBinaryPathName:          "",
	horizon.StellarCoreURLFlagName:             "",
	horizon.HistoryArchiveURLsFlagName:         "",
	horizon.NetworkPassphraseFlagName:          "",
}

const (
	SIMPLE_CAPTIVE_CORE_TOML = `
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
)

func NewParameterTestSkipContainerCreation(t *testing.T, params map[string]string) *integration.Test {
	return NewParameterTestWithEnvSkipContainerCreation(t, params, map[string]string{})
}

func NewParameterTestWithEnvSkipContainerCreation(t *testing.T, params, envvars map[string]string) *integration.Test {
	config := integration.Config{
		ProtocolVersion:         17,
		SkipHorizonStart:        true,
		SkipContainerCreation:   true,
		HorizonIngestParameters: params,
		HorizonEnvironment:      envvars}
	return integration.NewTest(t, config)
}

func NewParameterTest(t *testing.T, params map[string]string) *integration.Test {
	return NewParameterTestWithEnv(t, params, map[string]string{})
}

func NewParameterTestWithEnv(t *testing.T, params, envvars map[string]string) *integration.Test {
	config := integration.Config{
		ProtocolVersion:         17,
		SkipHorizonStart:        true,
		HorizonIngestParameters: params,
		HorizonEnvironment:      envvars,
	}
	return integration.NewTest(t, config)
}

func TestFatalScenarios(t *testing.T) {
	suite.Run(t, new(FatalTestCase))
}

// Ensures that BUCKET_DIR_PATH is not an allowed value for Captive Core.
func (suite *FatalTestCase) TestBucketDirDisallowed() {
	// This is a bit of a hacky workaround.
	//
	// In CI, we run our integration tests twice: once with Captive Core
	// enabled, and once without. *These* tests only run with Captive Core
	// configured properly (specifically, w/ the CAPTIVE_CORE_BIN envvar set).
	if !integration.RunWithCaptiveCore {
		suite.T().Skip()
	}

	config := `BUCKET_DIR_PATH="/tmp"
		` + SIMPLE_CAPTIVE_CORE_TOML

	confName, _, cleanup := createCaptiveCoreConfig(config)
	defer cleanup()

	test := NewParameterTest(suite.T(), map[string]string{
		horizon.CaptiveCoreConfigPathName: confName,
		horizon.StellarCoreBinaryPathName: os.Getenv("CAPTIVE_CORE_BIN"),
	})

	suite.Exits(func() { test.StartHorizon() })
}

func (suite *FatalTestCase) TestEnvironmentPreserved() {
	// Who tests the tests? This test.
	//
	// It ensures that the global OS environmental variables are preserved after
	// running an integration test.
	t := suite.T()

	// Note that we ALSO need to make sure we don't modify parent env state.
	value, isSet := os.LookupEnv("CAPTIVE_CORE_CONFIG_PATH")
	defer func() {
		if isSet {
			_ = os.Setenv("CAPTIVE_CORE_CONFIG_PATH", value)
		} else {
			_ = os.Unsetenv("CAPTIVE_CORE_CONFIG_PATH")
		}
	}()

	err := os.Setenv("CAPTIVE_CORE_CONFIG_PATH", "original value")
	assert.NoError(t, err)

	confName, _, cleanup := createCaptiveCoreConfig(SIMPLE_CAPTIVE_CORE_TOML)
	defer cleanup()
	test := NewParameterTestWithEnv(t, map[string]string{}, map[string]string{
		"CAPTIVE_CORE_CONFIG_PATH": confName,
	})

	err = test.StartHorizon()
	assert.NoError(t, err)
	test.WaitForHorizon()

	envValue := os.Getenv("CAPTIVE_CORE_CONFIG_PATH")
	assert.Equal(t, confName, envValue)

	test.Shutdown()

	envValue = os.Getenv("CAPTIVE_CORE_CONFIG_PATH")
	assert.Equal(t, "original value", envValue)
}

// TestNetworkParameter the main objective of this test is to ensure that Horizon successfully starts
// the captive-core subprocess using the default configuration when --network [testnet|pubnet] parameter
// is specified The test will fail for scenarios with an invalid static configuration, such as:
// - The default passphrase in Horizon is different from the one specified in the captive-core config.
// - History archive URLs are not configured.
// - The captive-core TOML config file is invalid.
// The test will also fail if an invalid parameter is specified.
//
// Typically during integration tests, we initiate Horizon in standalone mode and simultaneously start the
// stellar-core container in standalone mode as well. We wait for Horizon to begin ingesting to verify the test's
// success. However, in the case of "pubnet" or "testnet," waiting for Horizon to start ingesting is not practical.
// So we don't start stellar-core containers.
func TestNetworkParameter(t *testing.T) {
	if !integration.RunWithCaptiveCore {
		t.Skip()
	}

	t.Run("network parameter pubnet", func(t *testing.T) {
		localParams := integration.MergeMaps(networkParamArgs, map[string]string{
			horizon.NetworkFlagName: horizon.StellarPubnet,
		})
		test := NewParameterTestSkipContainerCreation(t, localParams)
		err := test.StartHorizon()
		time.Sleep(10 * time.Second)
		assert.NoError(t, err)
		test.StopHorizon()
		test.Shutdown()
	})

	t.Run("network parameter testnet", func(t *testing.T) {
		localParams := integration.MergeMaps(networkParamArgs, map[string]string{
			horizon.NetworkFlagName: horizon.StellarTestnet,
		})
		test := NewParameterTestSkipContainerCreation(t, localParams)
		err := test.StartHorizon()
		time.Sleep(10 * time.Second)
		assert.NoError(t, err)
		test.StopHorizon()
		test.Shutdown()
	})
}

// TestNetworkEnvironmentVariable the main objective of this test is to ensure that Horizon successfully
// starts the captive-core subprocess using the default configuration when the NETWORK environment variable
// is set to either pubnet or testnet.
// The test will fail for scenarios with an invalid configuration, such as:
// - The default passphrase in Horizon is different from the one specified in the captive-core config.
// - History archive URLs are not configured.
// - The captive-core TOML config file is invalid.
// The test will also fail if an invalid parameter is specified.
//
// Typically during integration tests, we initiate Horizon in standalone mode and simultaneously start the
// stellar-core container in standalone mode as well. We wait for Horizon to begin ingesting to verify the test's
// success. However, in the case of "pubnet" or "testnet," waiting for Horizon to start ingesting is not practical.
// So we don't start stellar-core containers.
func TestNetworkEnvironmentVariable(t *testing.T) {
	if !integration.RunWithCaptiveCore {
		t.Skip()
	}

	value, isSet := os.LookupEnv("NETWORK")
	defer func() {
		if isSet {
			_ = os.Setenv("NETWORK", value)
		} else {
			_ = os.Unsetenv("NETWORK")
		}
	}()

	t.Run("NETWORK environment variable pubnet", func(t *testing.T) {
		test := NewParameterTestWithEnvSkipContainerCreation(t, networkParamArgs,
			map[string]string{"NETWORK": horizon.StellarPubnet})
		err := test.StartHorizon()
		time.Sleep(10 * time.Second)
		assert.NoError(t, err)
		test.StopHorizon()
		test.Shutdown()
	})

	t.Run("NETWORK environment variable testnet", func(t *testing.T) {
		test := NewParameterTestWithEnvSkipContainerCreation(t, networkParamArgs,
			map[string]string{"NETWORK": horizon.StellarTestnet})
		err := test.StartHorizon()
		time.Sleep(10 * time.Second)
		assert.NoError(t, err)
		test.StopHorizon()
		test.Shutdown()
	})
}

// Ensures that the filesystem ends up in the correct state with Captive Core.
func TestCaptiveCoreConfigFilesystemState(t *testing.T) {
	if !integration.RunWithCaptiveCore {
		t.Skip() // explained above
	}

	confName, storagePath, cleanup := createCaptiveCoreConfig(SIMPLE_CAPTIVE_CORE_TOML)
	defer cleanup()

	localParams := integration.MergeMaps(defaultCaptiveCoreParameters, map[string]string{
		"captive-core-storage-path":       storagePath,
		horizon.CaptiveCoreConfigPathName: confName,
	})
	test := NewParameterTest(t, localParams)

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
		test := NewParameterTest(t, map[string]string{})
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.WaitForHorizon()
		assert.Equal(t, test.HorizonIngest().Config().MaxAssetsPerPathRequest, 15)
		test.Shutdown()
	})
	t.Run("set to 2", func(t *testing.T) {
		test := NewParameterTest(t, map[string]string{"max-assets-per-path-request": "2"})
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.WaitForHorizon()
		assert.Equal(t, test.HorizonIngest().Config().MaxAssetsPerPathRequest, 2)
		test.Shutdown()
	})
}

func TestMaxPathFindingRequests(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		test := NewParameterTest(t, map[string]string{})
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.WaitForHorizon()
		assert.Equal(t, test.HorizonIngest().Config().MaxPathFindingRequests, uint(0))
		_, ok := test.HorizonIngest().Paths().(simplepath.InMemoryFinder)
		assert.True(t, ok)
		test.Shutdown()
	})
	t.Run("set to 5", func(t *testing.T) {
		test := NewParameterTest(t, map[string]string{"max-path-finding-requests": "5"})
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
		test := NewParameterTest(t, map[string]string{})
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.WaitForHorizon()
		assert.Equal(t, test.HorizonIngest().Config().MaxPathFindingRequests, uint(0))
		_, ok := test.HorizonIngest().Paths().(simplepath.InMemoryFinder)
		assert.True(t, ok)
		test.Shutdown()
	})
	t.Run("set to true", func(t *testing.T) {
		test := NewParameterTest(t, map[string]string{"disable-path-finding": "true"})
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.WaitForHorizon()
		assert.Nil(t, test.HorizonIngest().Paths())
		test.Shutdown()
	})
}

func TestIngestionFilteringAlwaysDefaultingToTrue(t *testing.T) {
	t.Run("ingestion filtering flag set to default value", func(t *testing.T) {
		test := NewParameterTest(t, map[string]string{})
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.WaitForHorizon()
		assert.Equal(t, test.HorizonIngest().Config().EnableIngestionFiltering, true)
		test.Shutdown()
	})
	t.Run("ingestion filtering flag set to false", func(t *testing.T) {
		test := NewParameterTest(t, map[string]string{"exp-enable-ingestion-filtering": "false"})
		err := test.StartHorizon()
		assert.NoError(t, err)
		test.WaitForHorizon()
		assert.Equal(t, test.HorizonIngest().Config().EnableIngestionFiltering, true)
		test.Shutdown()
	})
}

func TestDeprecatedOutputForIngestionFilteringFlag(t *testing.T) {
	originalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	stdLog.SetOutput(os.Stderr)

	test := NewParameterTest(t, map[string]string{"exp-enable-ingestion-filtering": "false"})
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
		"no filtering of historical data. If you have never added filter rules to this deployment, then nothing further needed. "+
		"If you have defined ingestion filter rules prior but disabled filtering overall by setting this flag disabled with "+
		"--exp-enable-ingestion-filtering=false, then you should now delete the filter rules using the Horizon Admin API to achieve "+
		"the same no-filtering result. Remove usage of this flag in all cases.")
}

func TestHelpOutputForNoIngestionFilteringFlag(t *testing.T) {
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

	var writer io.Writer = &bytes.Buffer{}
	horizonCmd.SetOutput(writer)

	horizonCmd.SetArgs([]string{"-h"})
	if err := flags.Init(horizonCmd); err != nil {
		fmt.Println(err)
	}
	if err := horizonCmd.Execute(); err != nil {
		fmt.Println(err)
	}

	output := writer.(*bytes.Buffer).String()
	assert.NotContains(t, output, "--exp-enable-ingestion-filtering")
}

// Pattern taken from testify issue:
// https://github.com/stretchr/testify/issues/858#issuecomment-600491003
//
// This lets us run test cases that are *expected* to fail from a fatal error.
//
// For our purposes, if you *want* `StartHorizon()` to fail, you should wrap it
// in a lambda and pass it to `suite.Exits(...)`.
type FatalTestCase struct {
	suite.Suite
}

func (suite *FatalTestCase) Exits(subprocess func()) {
	testName := suite.T().Name()
	if os.Getenv("ASSERT_EXISTS_"+testName) == "1" {
		subprocess()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run="+testName)
	cmd.Env = append(os.Environ(), "ASSERT_EXISTS_"+testName+"=1")
	err := cmd.Run()

	suite.T().Log("Result:", err)
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}

	suite.Fail("expecting unsuccessful exit, got", err)
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
