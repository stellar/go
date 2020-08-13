package test

import (
	"context"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
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
	hclient   horizonclient.Client
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	reader, err := i.cli.ImagePull(ctx, "docker.io/"+image, types.ImagePullOptions{})
	if err != nil {
		t.Fatal(errors.Wrap(err, "error pulling docker image"))
	}
	defer reader.Close()
	ioutil.ReadAll(reader)

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
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: os.Getenv("HORIZON_BIN_DIR"),
					Target: "/custom_bin",
				},
			},
		},
		nil,
		"horizon-integration",
	)
	if err != nil {
		t.Fatal(errors.Wrap(err, "error creating docker container"))
	}

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = i.cli.ContainerStart(ctx, i.container.ID, types.ContainerStartOptions{})
	if err != nil {
		t.Fatal(errors.Wrap(err, "error starting docker container"))
	}

	i.hclient = horizonclient.Client{HorizonURL: "http://localhost:8000"}
	i.waitForIngestionAndUpgrade()
	return i
}

func (i *IntegrationTest) waitForIngestionAndUpgrade() {
	for t := 10 * time.Second; t >= 0; t -= time.Second {
		root, err := i.hclient.Root()
		if err != nil {
			if t == 0 {
				i.t.Fatal("Horizon not ingesting...")
			}
		}
		if root.IngestSequence > 0 &&
			root.HorizonSequence > 0 &&
			root.CurrentProtocolVersion == i.config.ProtocolVersion {
			return
		}
		time.Sleep(time.Second)
	}
}

// Client returns horizon.Client connected to started Horizon instance.
func (i *IntegrationTest) Client() *horizonclient.Client {
	return &horizonclient.Client{HorizonURL: "http://localhost:8000"}
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
