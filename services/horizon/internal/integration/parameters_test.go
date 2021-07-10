//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite
package integration

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	horizon "github.com/stellar/go/services/horizon/internal"
	"github.com/stellar/go/services/horizon/internal/test/integration"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func NewParameterTest(t *testing.T, params, envvars map[string]string) *integration.Test {
	config := integration.Config{
		ProtocolVersion:    17,
		SkipHorizonStart:   true,
		HorizonParameters:  params,
		HorizonEnvironment: envvars,
	}
	return integration.NewTest(t, config)
}

func TestFatalScenarios(t *testing.T) {
	suite.Run(t, new(FatalTestCase))
}

const (
	CAPTIVE_CORE_CONFIG_STATE_TOML = `
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

	config := `PEER_PORT=11725
		ARTIFICIALLY_ACCELERATE_TIME_FOR_TESTING=true

		UNSAFE_QUORUM=true
		FAILURE_SAFETY=0
		BUCKET_DIR_PATH="/tmp"

		[[VALIDATORS]]
		NAME="local_core"
		HOME_DOMAIN="core.local"
		# From SACJC372QBSSKJYTV5A7LWT4NXWHTQO6GHG4QDAVC2XDPX6CNNXFZ4JK
		PUBLIC_KEY="GD5KD2KEZJIGTC63IGW6UMUSMVUVG5IHG64HUTFWCHVZH2N2IBOQN7PS"
		ADDRESS="localhost"
		QUALITY="MEDIUM"`

	confName, _, cleanup := createCaptiveCoreConfig(config)
	defer cleanup()

	test := NewParameterTest(suite.T(), map[string]string{
		horizon.CaptiveCoreConfigPathName: confName,
		horizon.StellarCoreBinaryPathName: os.Getenv("CAPTIVE_CORE_BIN"),
	}, map[string]string{})

	suite.Exits(func() { test.StartHorizon() })
}

func (suite *FatalTestCase) TestEnvironmentPreserved() {
	// Who tests the tests? This test.
	//
	// It ensures that the global OS environmental variables are preserved after
	// running an integration test.
	t := suite.T()

	// Note that we ALSO need to make sure we don't modify parent env state.
	if value, isSet := os.LookupEnv("CAPTIVE_CORE_CONFIG_PATH"); isSet {
		defer func() {
			os.Setenv("CAPTIVE_CORE_CONFIG_PATH", value)
		}()
	}

	err := os.Setenv("CAPTIVE_CORE_CONFIG_PATH", "original value")
	assert.NoError(t, err)

	confName, _, cleanup := createCaptiveCoreConfig(CAPTIVE_CORE_CONFIG_STATE_TOML)
	defer cleanup()
	test := NewParameterTest(t,
		map[string]string{},
		map[string]string{
			"CAPTIVE_CORE_CONFIG_PATH": confName,
		},
	)

	test.StartHorizon()
	test.WaitForHorizon()

	envValue := os.Getenv("CAPTIVE_CORE_CONFIG_PATH")
	assert.Equal(t, confName, envValue)

	test.Shutdown()

	envValue = os.Getenv("CAPTIVE_CORE_CONFIG_PATH")
	assert.Equal(t, "original value", envValue)
}

// Ensures that the filesystem ends up in the correct state with Captive Core.
func TestCaptiveCoreConfigFilesystemState(t *testing.T) {
	if !integration.RunWithCaptiveCore {
		t.Skip() // explained above
	}

	confName, storagePath, cleanup := createCaptiveCoreConfig(CAPTIVE_CORE_CONFIG_STATE_TOML)
	defer cleanup()

	test := NewParameterTest(t, map[string]string{
		"captive-core-storage-path":       storagePath,
		"captive-core-reuse-storage-path": "true",
		horizon.StellarCoreBinaryPathName: os.Getenv("CAPTIVE_CORE_BIN"),
		horizon.CaptiveCoreConfigPathName: confName,
		horizon.StellarCoreURLFlagName:    "",
		horizon.StellarCoreDBURLFlagName:  "",
	}, map[string]string{})

	err := test.StartHorizon()
	assert.NoError(t, err)
	test.WaitForHorizon()

	runParameterMatrix(test, []ValidatorFunc{
		func() { validateCaptiveCoreDiskState(test, storagePath) },
		func() { validateNoBucketDirPath(test, storagePath) },
	})
}

// Pattern taken from testify issue:
// https://github.com/stretchr/testify/issues/858#issuecomment-600491003
//
// This lets us run test cases that are *expected* to fail from a fatal error.
type FatalTestCase struct {
	suite.Suite
}

func (t *FatalTestCase) Exits(subprocess func()) {
	testName := t.T().Name()
	if os.Getenv("ASSERT_EXISTS_"+testName) == "1" {
		subprocess()
		return
	}

	t.T().Log("Hello there! From Exits()")
	cmd := exec.Command(os.Args[0], "-test.run="+testName)
	cmd.Env = append(os.Environ(), "ASSERT_EXISTS_"+testName+"=1")
	err := cmd.Run()

	t.T().Log("Result:", err)
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}

	t.Fail("expecting unsuccessful exit, got", err)
}

type ValidatorFunc func()

func runParameterMatrix(itest *integration.Test, expectations []ValidatorFunc) {
	for _, validator := range expectations {
		validator()
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

	storagePath, err := ioutil.TempDir("", "captive-core-test-*-storage")
	if err != nil {
		panic(err)
	}

	filename := tomlFile.Name()
	return filename, storagePath, func() {
		os.Remove(filename)
		os.RemoveAll(storagePath)
	}
}
