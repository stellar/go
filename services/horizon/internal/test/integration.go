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
	sdk "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	proto "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/txnbuild"
	"github.com/stellar/go/support/errors"
)

const IntegrationNetworkPassphrase = "Standalone Network ; February 2017"

type IntegrationConfig struct {
	ProtocolVersion       int32
	SkipContainerCreation bool
}

type IntegrationTest struct {
	t         *testing.T
	config    IntegrationConfig
	cli       client.APIClient
	hclient   *sdk.Client
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
	skipCreation := os.Getenv("HORIZON_SKIP_CREATION") != ""

	if skipCreation {
		t.Log("Trying to skip container creation...")
		containers, _ := i.cli.ContainerList(
			context.Background(),
			types.ContainerListOptions{All: true, Quiet: true})

		for _, container := range containers {
			if container.Image == image {
				i.container.ID = container.ID
				break
			}
		}

		if i.container.ID != "" {
			t.Logf("Found matching container: %s\n", i.container.ID)
		} else {
			t.Log("No matching container found.")
			os.Unsetenv("HORIZON_SKIP_CREATION")
			skipCreation = false
		}
	}

	if !skipCreation {
		err = createTestContainer(i, image)
		if err != nil {
			t.Fatal(errors.Wrap(err, "error creating docker container"))
		}
	}

	// At this point, any of the following actions failing will cause the dead
	// container to stick around, failing any subsequent tests. Thus, we track a
	// flag to determine whether or not we should do this.
	doCleanup := true
	cleanup := func() {
		if doCleanup {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			i.cli.ContainerRemove(
				ctx, i.container.ID,
				types.ContainerRemoveOptions{Force: true})
		}
	}
	defer cleanup()

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	doCleanup = false
	i.hclient = &sdk.Client{HorizonURL: "http://localhost:8000"}
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
func (i *IntegrationTest) Client() *sdk.Client {
	return i.hclient
}

// LedgerIngested returns true if the ledger with a given sequence has been
// ingested by Horizon. Panics in case of errors.
func (i *IntegrationTest) LedgerIngested(sequence uint32) bool {
	root, err := i.Client().Root()
	if err != nil {
		panic(err)
	}

	return root.IngestSequence >= sequence
}

// AdminPort returns Horizon admin port.
func (i *IntegrationTest) AdminPort() int {
	return 6060
}

// Master returns a keypair of the network master account.
func (i *IntegrationTest) Master() *keypair.Full {
	return keypair.Master(IntegrationNetworkPassphrase).(*keypair.Full)
}

func (i *IntegrationTest) MasterAccount() txnbuild.Account {
	master, client := i.Master(), i.Client()
	request := sdk.AccountRequest{AccountID: master.Address()}
	account, err := client.AccountDetail(request)
	panicIf(err)
	return &account
}

func (i *IntegrationTest) CurrentTest() *testing.T {
	return i.t
}

// Close stops and removes the docker container.
func (i *IntegrationTest) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var err error
	skipCreation := os.Getenv("HORIZON_SKIP_CREATION") != ""
	if !skipCreation {
		i.t.Logf("Removing container %s\n", i.container.ID)
		err = i.cli.ContainerRemove(
			ctx, i.container.ID,
			types.ContainerRemoveOptions{Force: true})
	} else {
		i.t.Logf("Stopping container %s\n", i.container.ID)
		err = i.cli.ContainerStop(ctx, i.container.ID, nil)
	}

	panicIf(err)
}

func createTestContainer(i *IntegrationTest, image string) error {
	t := i.CurrentTest()
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
				"--protocol-version", strconv.FormatInt(int64(i.config.ProtocolVersion), 10),
			},
			ExposedPorts: nat.PortSet{"8000": struct{}{}, "6060": struct{}{}},
		},
		&container.HostConfig{
			PortBindings: map[nat.Port][]nat.PortBinding{
				nat.Port("8000"): {{HostIP: "127.0.0.1", HostPort: "8000"}},
				nat.Port("6060"): {{HostIP: "127.0.0.1", HostPort: "6060"}},
			},
		},
		nil,
		"horizon-integration",
	)

	return err
}

/* Utility functions for easier test case creation. */

// Creates new accounts via the master account.
//
// It funds each account with the given balance and then queries the API to
// find the randomized sequence number for future operations.
//
// Returns: The slice of created keypairs and account objects.
//
// Note: panics on any errors, since we assume that tests cannot proceed without
// this method succeeding.
func (i *IntegrationTest) CreateAccounts(count int, initialBalance string) ([]*keypair.Full, []txnbuild.Account) {
	client := i.Client()
	master := i.Master()

	pairs := make([]*keypair.Full, count)
	ops := make([]txnbuild.Operation, count)

	// Two paths here: either caller already did some stuff with the master
	// account so we should retrieve the sequence number, or caller hasn't and
	// we start from scratch.
	seq := int64(0)
	request := sdk.AccountRequest{AccountID: master.Address()}
	account, err := client.AccountDetail(request)
	if err == nil {
		seq, err = strconv.ParseInt(account.Sequence, 10, 8) // why is this a string?
		panicIf(err)
	}

	masterAccount := txnbuild.SimpleAccount{
		AccountID: master.Address(),
		Sequence:  seq,
	}

	for i := 0; i < count; i++ {
		pair, _ := keypair.Random()
		pairs[i] = pair

		ops[i] = &txnbuild.CreateAccount{
			SourceAccount: &masterAccount,
			Destination:   pair.Address(),
			Amount:        initialBalance,
		}
	}

	// Submit transaction, then retrieve new account details.
	_ = i.MustSubmitOperations(&masterAccount, master, ops...)

	accounts := make([]txnbuild.Account, count)
	for i, kp := range pairs {
		request := sdk.AccountRequest{AccountID: kp.Address()}
		account, err := client.AccountDetail(request)
		panicIf(err)

		accounts[i] = &account
	}

	for _, keys := range pairs {
		i.t.Logf("Funded %s (%s).\n", keys.Seed(), keys.Address())
	}

	return pairs, accounts
}

// Panics on any error establishing a trustline.
func (i *IntegrationTest) MustEstablishTrustline(
	truster *keypair.Full, account txnbuild.Account, asset txnbuild.Asset,
) (resp proto.Transaction) {
	txResp, err := i.EstablishTrustline(truster, account, asset)
	panicIf(err)
	return txResp
}

// Establishes a trustline for a given asset for a particular account.
func (i *IntegrationTest) EstablishTrustline(
	truster *keypair.Full, account txnbuild.Account, asset txnbuild.Asset,
) (proto.Transaction, error) {
	if asset.IsNative() {
		return proto.Transaction{}, nil
	}
	return i.SubmitOperations(account, truster, &txnbuild.ChangeTrust{
		Line:  asset,
		Limit: "2000",
	})
}

// Panics on any error creating a claimable balance.
func (i *IntegrationTest) MustCreateClaimableBalance(
	source *keypair.Full, asset txnbuild.Asset, amount string,
	claimants ...txnbuild.Claimant,
) (claim proto.ClaimableBalance) {
	account := i.MustGetAccount(source)
	_ = i.MustSubmitOperations(&account, source,
		&txnbuild.CreateClaimableBalance{
			Destinations: claimants,
			Asset:        asset,
			Amount:       amount,
		},
	)

	// Ensure it exists in the global list
	balances, err := i.Client().ClaimableBalances(sdk.ClaimableBalanceRequest{})
	panicIf(err)

	claims := balances.Embedded.Records
	if len(claims) == 0 {
		panic(-1)
	}

	claim = claims[len(claims)-1] // latest one
	i.t.Logf("Created claimable balance w/ id=%s", claim.BalanceID)
	return
}

// Panics on any error retrieves an account's details from its key.
// This means it must have previously been funded.
func (i *IntegrationTest) MustGetAccount(source *keypair.Full) proto.Account {
	client := i.Client()
	account, err := client.AccountDetail(sdk.AccountRequest{AccountID: source.Address()})
	panicIf(err)
	return account
}

// Submits a signed transaction from an account with standard options.
//
// Namely, we set the standard fee, time bounds, etc. to "non-production"
// defaults that work well for tests.
//
// Most transactions only need one signer, so see the more verbose
// `MustSubmitOperationsWithSigners` below for multi-sig transactions.
//
// Note: We assume that transaction will be successful here so we panic in case
// of all errors. To allow failures, use `SubmitOperations`.
func (i *IntegrationTest) MustSubmitOperations(
	source txnbuild.Account, signer *keypair.Full, ops ...txnbuild.Operation,
) proto.Transaction {
	tx, err := i.SubmitOperations(source, signer, ops...)
	panicIf(err)
	return tx
}

func (i *IntegrationTest) SubmitOperations(
	source txnbuild.Account, signer *keypair.Full, ops ...txnbuild.Operation,
) (proto.Transaction, error) {
	return i.SubmitMultiSigOperations(source, []*keypair.Full{signer}, ops...)
}

func (i *IntegrationTest) SubmitMultiSigOperations(
	source txnbuild.Account, signers []*keypair.Full, ops ...txnbuild.Operation,
) (txResp proto.Transaction, err error) {
	txParams := txnbuild.TransactionParams{
		SourceAccount:        source,
		Operations:           ops,
		BaseFee:              txnbuild.MinBaseFee,
		Timebounds:           txnbuild.NewInfiniteTimeout(),
		IncrementSequenceNum: true,
	}

	tx, err := txnbuild.NewTransaction(txParams)
	if err != nil {
		return
	}

	for _, signer := range signers {
		tx, err = tx.Sign(IntegrationNetworkPassphrase, signer)
		if err != nil {
			return
		}
	}

	txb64, err := tx.Base64()
	if err != nil {
		return
	}

	txResp, err = i.Client().SubmitTransactionXDR(txb64)
	if err != nil {
		i.t.Logf("Submitting the transaction failed: %s\n", txb64)
		if prob := sdk.GetError(err); prob != nil {
			i.t.Logf("Problem: %s\n", prob.Problem.Detail)
			i.t.Logf("Extras: %s\n", prob.Problem.Extras["result_codes"])
		}
		return
	}

	return
}

// Cluttering code with if err != nil is absolute nonsense.
func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}
