package test

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/errors"
)

const IntegrationNetworkPassphrase = "Standalone Network ; February 2017"

type IntegrationConfig struct {
	ProtocolVersion int32
}

type IntegrationTest struct {
	t         *testing.T
	config    IntegrationConfig
	cli       client.APIClient
	hclient   *horizonclient.Client
	container container.ContainerCreateCreatedBody
}

// NewIntegrationTest starts a new environment for integration test at a given
// protocol version and blocks until Horizon starts ingesting.
//
// Warning: this requires:
//  * Docker installed and all docker env variables set.
//  * HORIZON_BIN_DIR env variable set to the directory with `horizon` binary to test.
//  * Horizon binary must be built for GOOS=linux and GOARCH=amd64.
//
// Skips the test if HORIZON_INTEGRATION_TESTS env variable is not set.
func NewIntegrationTest(t *testing.T, config IntegrationConfig) *IntegrationTest {
	if os.Getenv("HORIZON_INTEGRATION_TESTS") == "" {
		t.Skip("skipping integration test")
	}

	if os.Getenv("HORIZON_BIN_DIR") == "" {
		t.Fatal("HORIZON_BIN_DIR env variable not set")
	}

	i := &IntegrationTest{t: t, config: config}

	var err error
	i.cli, err = client.NewEnvClient()
	if err != nil {
		t.Fatal(errors.Wrap(err, "error creating docker client"))
	}

	image := "stellar/quickstart:testing"

	t.Logf("Pulling %s...", image)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	reader, err := i.cli.ImagePull(ctx, "docker.io/"+image, types.ImagePullOptions{})
	if err != nil {
		t.Fatal(errors.Wrap(err, "error pulling docker image"))
	}
	defer reader.Close()
	io.Copy(os.Stdout, reader)

	t.Log("Creating container...")
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	i.container, err = i.cli.ContainerCreate(
		ctx,
		&container.Config{
			Image: image,
			Cmd: []string{
				"--standalone",
				"--protocol-version", strconv.FormatInt(int64(config.ProtocolVersion), 10),
			},
			ExposedPorts: nat.PortSet{"8000": struct{}{}},
		},
		&container.HostConfig{
			PortBindings: map[nat.Port][]nat.PortBinding{
				nat.Port("8000"): {{HostIP: "127.0.0.1", HostPort: "8000"}},
			},
		},
		nil,
		"horizon-integration",
	)
	if err != nil {
		t.Fatal(errors.Wrap(err, "error creating docker container"))
	}

	horizonBinaryContents, err := ioutil.ReadFile(os.Getenv("HORIZON_BIN_DIR") + "/horizon")
	if err != nil {
		t.Fatal(errors.Wrap(err, "error reading horizon binary file"))
	}

	// Create a tar archive with horizon binary (required by docker API).
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	hdr := &tar.Header{
		Name: "horizon",
		Mode: 0755,
		Size: int64(len(horizonBinaryContents)),
	}
	if err = tw.WriteHeader(hdr); err != nil {
		t.Fatal(errors.Wrap(err, "error writing tar header"))
	}
	if _, err = tw.Write(horizonBinaryContents); err != nil {
		t.Fatal(errors.Wrap(err, "error writing tar contents"))
	}
	if err = tw.Close(); err != nil {
		t.Fatal(errors.Wrap(err, "error closing tar archive"))
	}

	t.Log("Copying custom horizon binary...")
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = i.cli.CopyToContainer(ctx, i.container.ID, "/usr/local/bin", &buf, types.CopyToContainerOptions{})
	if err != nil {
		t.Fatal(errors.Wrap(err, "error copying custom horizon binary"))
	}

	t.Log("Starting container...")
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = i.cli.ContainerStart(ctx, i.container.ID, types.ContainerStartOptions{})
	if err != nil {
		t.Fatal(errors.Wrap(err, "error starting docker container"))
	}

	i.hclient = &horizonclient.Client{HorizonURL: "http://localhost:8000"}
	i.waitForIngestionAndUpgrade()
	return i
}

func (i *IntegrationTest) waitForIngestionAndUpgrade() {
	for t := 30 * time.Second; t >= 0; t -= time.Second {
		i.t.Log("Waiting for ingestion and protocol upgrade...")
		root, _ := i.hclient.Root()
		// We ignore errors here because it's likely connection error due to
		// Horizon not running. We ensure that's is up and correct by checking
		// the root response.
		if root.IngestSequence > 0 &&
			root.HorizonSequence > 0 &&
			root.CurrentProtocolVersion == i.config.ProtocolVersion {
			i.t.Log("Horizon ingesting and protocol version matches...")
			return
		}
		time.Sleep(time.Second)
	}

	i.t.Fatal("Horizon not ingesting...")
}

// Client returns horizon.Client connected to started Horizon instance.
func (i *IntegrationTest) Client() *horizonclient.Client {
	return i.hclient
}

// Master returns a keypair of the network master account.
func (i *IntegrationTest) Master() keypair.KP {
	return keypair.Master(IntegrationNetworkPassphrase)
}

// Close stops and removes the docker container.
func (i *IntegrationTest) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	err := i.cli.ContainerRemove(ctx, i.container.ID, types.ContainerRemoveOptions{
		Force: true,
	})
	if err != nil {
		panic(err)
	}
}
