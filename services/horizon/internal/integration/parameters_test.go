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

// Ensures that BUCKET_DIR_PATH is not an allowed value for Captive Core.
const (
	BUCKET_DIR_DISALLOWED_TOML = `
		PEER_PORT=11725
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

func (suite *FatalTestCase) TestBucketDirDisallowed() {
	// This is a bit of a hacky workaround.
	//
	// In CI, we run our integration tests twice: once with Captive Core
	// enabled, and once without. *These* tests only run with Captive Core
	// configured properly (specifically, w/ the CAPTIVE_CORE_BIN envvar set).
	if !integration.RunWithCaptiveCore {
		suite.T().Skip()
	}

	confName, cleanup := createCaptiveCoreConfig(BUCKET_DIR_DISALLOWED_TOML)
	defer cleanup()

	const STORAGE_PATH string = "./test_no-bucket-dir"
	test := NewParameterTest(suite.T(), map[string]string{
		"captive-core-storage-path":       STORAGE_PATH,
		horizon.CaptiveCoreConfigPathName: confName,
		horizon.StellarCoreBinaryPathName: os.Getenv("CAPTIVE_CORE_BIN"),
	}, map[string]string{})
	defer os.RemoveAll(STORAGE_PATH)

	suite.Exits(func() { test.StartHorizon() })
}

func (suite *FatalTestCase) TestEnvironmentPreserved() {
	// Who tests the tests? This test.
	//
	// It ensures that the global OS environmental variables are preserved after
	// running an integration test.
	t := suite.T()

	// Note that we ALSO need to make sure we don't modify parent env state.
	if value, isSet := os.LookupEnv("STELLAR_CORE_BINARY_PATH"); isSet {
		defer func() {
			os.Setenv("STELLAR_CORE_BINARY_PATH", value)
		}()
	}

	err := os.Setenv("STELLAR_CORE_BINARY_PATH", "dummy value")
	assert.NoError(t, err)

	test := NewParameterTest(t,
		map[string]string{},
		map[string]string{
			// intentionally invalid parameter combination
			"CAPTIVE_CORE_CONFIG_PATH": "",
			"STELLAR_CORE_BINARY_PATH": "/nonsense",
		},
	)

	suite.Exits(func() { test.StartHorizon() })
	test.Shutdown()

	envValue := os.Getenv("STELLAR_CORE_BINARY_PATH")
	assert.Equal(t, "dummy value", envValue)
}

// Ensures that the filesystem ends up in the correct state with Captive Core.
func TestCaptiveCoreConfigFilesystemState(t *testing.T) {
	if !integration.RunWithCaptiveCore {
		t.Skip() // explained above
	}

	confName, cleanup := createCaptiveCoreConfig(CAPTIVE_CORE_CONFIG_STATE_TOML)
	defer cleanup()

	const STORAGE_PATH string = "./test_captive-core-works"
	test := NewParameterTest(t, map[string]string{
		"captive-core-storage-path":       STORAGE_PATH,
		"captive-core-reuse-storage-path": "true",
		horizon.StellarCoreBinaryPathName: os.Getenv("CAPTIVE_CORE_BIN"),
		horizon.CaptiveCoreConfigPathName: confName,
		horizon.StellarCoreURLFlagName:    "",
		horizon.StellarCoreDBURLFlagName:  "",
	}, map[string]string{})
	defer os.RemoveAll(STORAGE_PATH)

	err := test.StartHorizon()
	assert.NoError(t, err)
	test.WaitForHorizon()

	runParameterMatrix(test, []ValidatorFunc{
		func() { validateCaptiveCoreDiskState(test, STORAGE_PATH) },
		func() { validateNoBucketDirPath(test, STORAGE_PATH) },
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

	cmd := exec.Command(os.Args[0], "-test.run="+testName)
	cmd.Env = append(os.Environ(), "ASSERT_EXISTS_"+testName+"=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}

	t.Fail("expecting unsuccessful exit")
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

func createCaptiveCoreConfig(contents string) (string, func()) {
	tomlFile, err := ioutil.TempFile("", "captive-core-test-*.toml")
	defer tomlFile.Close()
	if err != nil {
		panic(err)
	}

	_, err = tomlFile.WriteString(contents)
	if err != nil {
		panic(err)
	}

	filename := tomlFile.Name()
	return filename, func() { os.Remove(filename) }
}
