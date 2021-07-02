package integration

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/stellar/go/services/horizon/internal/test/integration"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func NewParameterTest(t *testing.T, params map[string]string) *integration.Test {
	config := integration.Config{
		ProtocolVersion:   17,
		DontStartHorizon:  true,
		HorizonParameters: params,
	}
	return integration.NewTest(t, config)
}

func TestHorizonWorksWithoutCaptiveCore(t *testing.T) {
	// This is a regression test, sourced from
	// https://github.com/stellar/go/issues/3507
	test := NewParameterTest(t(), map[string]string{
		"--enable-captive-core-ingestion": "false",
		"--ingest-failed-transactions":    "true",
		"--ingest":                        "true",
	})

	err := test.StartHorizon()
	assert.NoError(t, err)
}

func TestFatalScenarios(t *testing.T) {
	suite.Run(t, new(FatalTestCase))
}

func (suite *FatalTestCase) TestBucketDirDisallowed() {
	defer createCaptiveCoreConfig(
		"./captive-core.toml", `
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
QUALITY="MEDIUM"`)()

	const STORAGE_PATH string = "./test_no-bucket-dir"

	test := NewParameterTest(suite.T(), map[string]string{
		"--captive-core-storage-path": STORAGE_PATH,
		"--stellar-core-binary-path":  "/usr/local/bin/stellar-core",
		"--captive-core-config-path":  "./captive-core.toml",
		// "--stellar-core-db-url":       "",
		// "--stellar-core-url":          "",
	})
	defer os.RemoveAll(STORAGE_PATH)

	suite.Exits(func() {
		test.StartHorizon()
	})

	// runParameterMatrix(test, []ValidatorFunc{
	// 	func() { validateCaptiveCoreDiskState(test, STORAGE_PATH) },
	// 	func() { validateNoBucketDirPath(test, STORAGE_PATH) },
	// })
}

// Pattern taken from testify issue:
// https://github.com/stretchr/testify/issues/858#issuecomment-600491003
//
// This lets us run test cases that are *expected* to fail from a fatal error.
type FatalTestCase struct {
	suite.Suite
}

func (t *FatalTestCase) Exits(subprocess func()) {
	if os.Getenv("ASSERT_EXISTS_"+t.T().Name()) == "1" {
		subprocess()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run="+t.T().Name())
	cmd.Env = append(os.Environ(), "ASSERT_EXISTS_"+t.T().Name()+"=1")
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

func createCaptiveCoreConfig(path, contents string) func() {
	tomlFile, err := os.Create(path)
	defer tomlFile.Close()
	if err != nil {
		panic(err)
	}

	_, err = tomlFile.WriteString(contents)
	if err != nil {
		panic(err)
	}

	return func() { os.Remove(path) }
}
