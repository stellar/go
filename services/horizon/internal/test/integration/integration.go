package integration

import (
	"context"
	"github.com/spf13/cobra"
	horizon "github.com/stellar/go/services/horizon/internal"
	"github.com/stellar/go/support/db/dbtest"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	sdk "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	proto "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

const (
	IntegrationNetworkPassphrase = "Standalone Network ; February 2017"
	stellarCorePostgresPassword  = "integration-tests-password"
	adminPort                    = 6060
)

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
	app       *horizon.App
}

// NewIntegrationTest starts a new environment for integration test at a given
// protocol version and blocks until Horizon starts ingesting.
//
// Warning: this requires:
//  * Docker installed and all docker env variables set.
//
// Skips the test if HORIZON_INTEGRATION_TESTS env variable is not set.
func NewIntegrationTest(t *testing.T, config IntegrationConfig) *IntegrationTest {
	if os.Getenv("HORIZON_INTEGRATION_TESTS") == "" {
		t.Skip("skipping integration test")
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

	t.Log("Starting container...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = i.cli.ContainerStart(ctx, i.container.ID, types.ContainerStartOptions{})
	if err != nil {
		t.Fatal(errors.Wrap(err, "error starting docker container"))
	}

	// only use horizon from quickstart container when *NOT* testing captive core
	if os.Getenv("HORIZON_INTEGRATION_ENABLE_CAPTIVE_CORE") == "" {
		i.startHorizon()
	}

	doCleanup = false
	i.hclient = &sdk.Client{HorizonURL: "http://localhost:8000"}

	// Register cleanup handlers (on panic and ctrl+c) so the container is
	// removed even if ingestion or testing fails.
	i.t.Cleanup(i.Close)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		i.Close()
		os.Exit(0)
	}()

	i.waitForIngestionAndUpgrade()
	return i
}

func (i *IntegrationTest) startHorizon() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := i.cli.ContainerInspect(ctx, i.container.ID)
	if err != nil {
		i.t.Fatal(errors.Wrap(err, "error inspecting container"))
	}

	stellarCore := info.NetworkSettings.Ports["11626/tcp"][0]

	stellarCorePostgres := info.NetworkSettings.Ports["5432/tcp"][0]
	stellarCorePostgresURL := "postgres://stellar:" + stellarCorePostgresPassword + "@" + stellarCorePostgres.HostIP + ":" + stellarCorePostgres.HostPort + "/core"

	historyArchive := info.NetworkSettings.Ports["1570/tcp"][0]

	horizonPostgresURL := dbtest.Postgres(i.t).DSN

	config, configOpts := horizon.Flags()

	cmd := &cobra.Command{
		Use:   "horizon",
		Short: "client-facing api server for the stellar network",
		Long:  "client-facing api server for the stellar network. It acts as the interface between Stellar Core and applications that want to access the Stellar network. It allows you to submit transactions to the network, check the status of accounts, subscribe to event streams and more.",
		Run: func(cmd *cobra.Command, args []string) {
			i.app = horizon.NewAppFromFlags(config, configOpts)
		},
	}
	cmd.SetArgs([]string{
		"--stellar-core-url",
		"http://" + stellarCore.HostIP + ":" + stellarCore.HostPort,
		"--history-archive-urls",
		"http://" + historyArchive.HostIP + ":" + historyArchive.HostPort,
		"--ingest",
		"--db-url",
		horizonPostgresURL,
		"--stellar-core-db-url",
		stellarCorePostgresURL,
		"--network-passphrase",
		IntegrationNetworkPassphrase,
		"--apply-migrations",
		"--admin-port",
		strconv.Itoa(adminPort),
	})
	configOpts.Init(cmd)

	if err = cmd.Execute(); err != nil {
		i.t.Fatalf("cannot initialize horizon: %s", err)
	}

	if err = i.app.Ingestion().BuildGenesisState(); err != nil {
		i.t.Fatalf("cannot build genesis state: %s", err)
	}

	go i.app.Serve()
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

	if i.app != nil {
		i.app.Close()
	}

	skipCreation := os.Getenv("HORIZON_SKIP_CREATION") != ""
	if !skipCreation {
		i.t.Logf("Removing container %s\n", i.container.ID)
		i.cli.ContainerRemove(
			ctx, i.container.ID,
			types.ContainerRemoveOptions{Force: true})
	} else {
		i.t.Logf("Stopping container %s\n", i.container.ID)
		i.cli.ContainerStop(ctx, i.container.ID, nil)
	}
}

func createTestContainer(i *IntegrationTest, image string) error {
	t := i.CurrentTest()
	t.Logf("Pulling %s...", image)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// If your Internet (or docker.io) is down, integration tests should still try to run.
	reader, err := i.cli.ImagePull(ctx, "docker.io/"+image, types.ImagePullOptions{})
	if err != nil {
		t.Log("  error pulling docker image")
		t.Log("  trying to find local image (might be out-dated)")

		args := filters.NewArgs()
		args.Add("reference", "stellar/quickstart:testing")
		list, innerErr := i.cli.ImageList(ctx, types.ImageListOptions{Filters: args})
		if innerErr != nil || len(list) == 0 {
			t.Fatal(errors.Wrap(err, "failed to find local image"))
		}
		t.Log("  using local", image)
	} else {
		defer reader.Close()
		io.Copy(os.Stdout, reader)
	}

	t.Log("Creating container...")
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	containerConfig := &container.Config{
		Image: image,
		Cmd: []string{
			"--standalone",
			"--protocol-version", strconv.FormatInt(int64(i.config.ProtocolVersion), 10),
		},
	}
	hostConfig := &container.HostConfig{}

	if os.Getenv("HORIZON_INTEGRATION_ENABLE_CAPTIVE_CORE") != "" {
		containerConfig.Env = append(containerConfig.Env,
			"ENABLE_CAPTIVE_CORE_INGESTION=true",
			"STELLAR_CORE_BINARY_PATH=/opt/stellar/core/bin/start",
			"STELLAR_CORE_CONFIG_PATH=/opt/stellar/core/etc/stellar-core.cfg",
		)
		containerConfig.ExposedPorts = nat.PortSet{"8000": struct{}{}, "6060": struct{}{}}
	} else {
		containerConfig.Env = append(containerConfig.Env,
			"POSTGRES_PASSWORD=" + stellarCorePostgresPassword,
		)
		containerConfig.ExposedPorts = nat.PortSet{"11626": struct{}{}, "5432": struct{}{}, "1570": struct{}{}}
		hostConfig.PublishAllPorts = true
	}

	i.container, err = i.cli.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
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
		i.t.Logf("Funded %s (%s) with %s XLM.\n",
			keys.Seed(), keys.Address(), initialBalance)
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
) (proto.Transaction, error) {
	txParams := txnbuild.TransactionParams{
		SourceAccount:        source,
		Operations:           ops,
		BaseFee:              txnbuild.MinBaseFee,
		Timebounds:           txnbuild.NewInfiniteTimeout(),
		IncrementSequenceNum: true,
	}

	tx, err := txnbuild.NewTransaction(txParams)
	if err != nil {
		return proto.Transaction{}, err
	}

	for _, signer := range signers {
		tx, err = tx.Sign(IntegrationNetworkPassphrase, signer)
		if err != nil {
			return proto.Transaction{}, err
		}
	}

	txb64, err := tx.Base64()
	if err != nil {
		return proto.Transaction{}, err
	}

	return i.Client().SubmitTransactionXDR(txb64)
}

// A convenience function to provide verbose information about a failing
// transaction to the test output log, if it's expected to succeed.
func (i *IntegrationTest) LogFailedTx(txResponse proto.Transaction, horizonResult error) {
	t := i.CurrentTest()
	assert.NoErrorf(t, horizonResult, "Submitting the transaction failed")
	if prob := sdk.GetError(horizonResult); prob != nil {
		t.Logf("  problem: %s\n", prob.Problem.Detail)
		t.Logf("  extras: %s\n", prob.Problem.Extras["result_codes"])
		return
	}

	var txResult xdr.TransactionResult
	err := xdr.SafeUnmarshalBase64(txResponse.ResultXdr, &txResult)
	assert.NoErrorf(t, err, "Unmarshalling transaction failed.")
	assert.Equalf(t, xdr.TransactionResultCodeTxSuccess, txResult.Result.Code,
		"Transaction doesn't have success code.")
}

// Cluttering code with if err != nil is absolute nonsense.
func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}
