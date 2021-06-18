package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stellar/go/services/horizon/internal/test/integration"

	"github.com/stretchr/testify/assert"
)

func NewParameterTest(t *testing.T, params map[string]string) *integration.Test {
	config := integration.Config{ProtocolVersion: 17, HorizonParameters: params}
	return integration.NewTest(t, config)
}

func TestBasicParameters(t *testing.T) {
	tt := assert.New(t)
	var err error

	tomlFile, err := os.Create("./captive-core.toml")
	tt.NoError(err)
	defer func() {
		tomlFile.Close()
		os.Remove("./captive-core.toml")
	}()

	tomlConfig := []string{
		"PEER_PORT=11725",
		"ARTIFICIALLY_ACCELERATE_TIME_FOR_TESTING=true",

		"UNSAFE_QUORUM=true",
		"FAILURE_SAFETY=0",
		"BUCKET_DIR_PATH=/tmp",

		"[[VALIDATORS]]",
		"NAME=\"local_core\"",
		"HOME_DOMAIN=\"core.local\"",
		"# From \"SACJC372QBSSKJYTV5A7LWT4NXWHTQO6GHG4QDAVC2XDPX6CNNXFZ4JK\"",
		"PUBLIC_KEY=\"GD5KD2KEZJIGTC63IGW6UMUSMVUVG5IHG64HUTFWCHVZH2N2IBOQN7PS\"",
		"ADDRESS=\"localhost\"",
		"QUALITY=\"MEDIUM\"",
	}

	for _, line := range tomlConfig {
		_, innerErr := tomlFile.WriteString(fmt.Sprintf("%s\n", line))
		tt.NoError(innerErr)
	}

	NewParameterTest(t, map[string]string{
		"--captive-core-storage-path": "./test-basic-parameters",
		"--stellar-core-binary-path":  "/usr/local/bin/stellar-core",
		"--captive-core-config-path":  "./captive-core.toml",
	})

	cwd, err := os.Getwd()
	tt.NoError(err)
	fmt.Println(cwd)

	// confirm filesystem state
	_, err = os.Stat("./test-basic-parameters")
	tt.NoError(err)
	_, err = os.Stat("./test-basic-parameters/captive-core")
	tt.NoError(err)

	// confirm captive core config state
	result, err := ioutil.ReadFile("./test-basic-parameters/captive-core/stellar-core.conf")
	fmt.Println(string(result))
	tt.NoError(err)

	bucketPathSet := strings.Contains(string(result), "BUCKET_DIR_PATH")
	tt.True(bucketPathSet)

	// os.RemoveAll("./test-basic-parameters")
}

func confirmParameterState(parameter string, value string, expectation func() bool) bool {
	return expectation()
}
