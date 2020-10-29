//lint:file-ignore U1001 Ignore all unused code, this is only used in tests.
package integration

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/cobra"

	sdk "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/clients/stellarcore"
	"github.com/stellar/go/keypair"
	proto "github.com/stellar/go/protocols/horizon"
	horizon "github.com/stellar/go/services/horizon/internal"
	"github.com/stellar/go/support/db/dbtest"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

const (
	NetworkPassphrase           = "Standalone Network ; February 2017"
	stellarCorePostgresPassword = "integration-tests-password"
	adminPort                   = 6060
)

var (
	stellarCorePort         = mustPort("tcp", "11626")
	stellarCorePostgresPort = mustPort("tcp", "5432")
	historyArchivePort      = mustPort("tcp", "1570")
)

func mustPort(proto, port string) nat.Port {
	p, err := nat.NewPort(proto, port)
	panicIf(err)
	return p
}

type Config struct {
	ProtocolVersion       int32
	SkipContainerCreation bool
}

type Test struct {
	t         *testing.T
	config    Config
	cli       client.APIClient
	hclient   *sdk.Client
	cclient   *stellarcore.Client
	container container.ContainerCreateCreatedBody
	app       *horizon.App
}

// NewTest starts a new environment for integration test at a given
// protocol version and blocks until Horizon starts ingesting.
//
// Warning: this requires:
//  * Docker installed and all docker env variables set.
//  * HORIZON_BIN_DIR env variable set to the directory with `horizon` binary to test.
//  * Horizon binary must be built for GOOS=linux and GOARCH=amd64.
//
// Skips the test if HORIZON_INTEGRATION_TESTS env variable is not set.
func NewTest(t *testing.T, config Config) *Test {
	if os.Getenv("HORIZON_INTEGRATION_TESTS") == "" {
		t.Skip("skipping integration test")
	}

	i := &Test{t: t, config: config}

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

	i.setupHorizonBinary()

	t.Log("Starting container...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = i.cli.ContainerStart(ctx, i.container.ID, types.ContainerStartOptions{})
	if err != nil {
		t.Fatal(errors.Wrap(err, "error starting docker container"))
	}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	containerInfo, err := i.cli.ContainerInspect(ctx, i.container.ID)
	if err != nil {
		i.t.Fatal(errors.Wrap(err, "error inspecting container"))
	}
	stellarCoreBinding := containerInfo.NetworkSettings.Ports[stellarCorePort][0]
	coreURL := fmt.Sprintf("http://%s:%s", stellarCoreBinding.HostIP, stellarCoreBinding.HostPort)
	// only use horizon from quickstart container when testing captive core
	if os.Getenv("HORIZON_INTEGRATION_ENABLE_CAPTIVE_CORE") == "" {
		i.startHorizon(containerInfo, coreURL)
	}

	doCleanup = false
	i.hclient = &sdk.Client{HorizonURL: "http://localhost:8000"}
	i.cclient = &stellarcore.Client{URL: coreURL}

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

	i.waitForCore()
	i.waitForIngestionAndUpgrade()
	return i
}

func (i *Test) setupHorizonBinary() {
	// only use horizon from quickstart container when testing captive core
	if os.Getenv("HORIZON_INTEGRATION_ENABLE_CAPTIVE_CORE") == "" {
		return
	}

	if os.Getenv("HORIZON_BIN_DIR") == "" {
		i.t.Fatal("HORIZON_BIN_DIR env variable not set")
	}

	horizonBinaryContents, err := ioutil.ReadFile(os.Getenv("HORIZON_BIN_DIR") + "/horizon")
	if err != nil {
		i.t.Fatal(errors.Wrap(err, "error reading horizon binary file"))
	}

	// Create a tar archive with horizon binary (required by docker API).
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	hdr := &tar.Header{
		Name: "stellar-horizon",
		Mode: 0755,
		Size: int64(len(horizonBinaryContents)),
	}
	if err = tw.WriteHeader(hdr); err != nil {
		i.t.Fatal(errors.Wrap(err, "error writing tar header"))
	}
	if _, err = tw.Write(horizonBinaryContents); err != nil {
		i.t.Fatal(errors.Wrap(err, "error writing tar contents"))
	}
	if err = tw.Close(); err != nil {
		i.t.Fatal(errors.Wrap(err, "error closing tar archive"))
	}

	i.t.Log("Copying custom horizon binary...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = i.cli.CopyToContainer(ctx, i.container.ID, "/usr/bin/", &buf, types.CopyToContainerOptions{})
	if err != nil {
		i.t.Fatal(errors.Wrap(err, "error copying custom horizon binary"))
	}
}

func (i *Test) startHorizon(containerInfo types.ContainerJSON, coreURL string) {
	stellarCorePostgres := containerInfo.NetworkSettings.Ports[stellarCorePostgresPort][0]
	stellarCorePostgresURL := fmt.Sprintf(
		"postgres://stellar:%s@%s:%s/core",
		stellarCorePostgresPassword,
		stellarCorePostgres.HostIP,
		stellarCorePostgres.HostPort,
	)

	historyArchive := containerInfo.NetworkSettings.Ports[historyArchivePort][0]

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
		coreURL,
		"--history-archive-urls",
		fmt.Sprintf("http://%s:%s", historyArchive.HostIP, historyArchive.HostPort),
		"--ingest",
		"--db-url",
		horizonPostgresURL,
		"--stellar-core-db-url",
		stellarCorePostgresURL,
		"--network-passphrase",
		NetworkPassphrase,
		"--apply-migrations",
		"--admin-port",
		strconv.Itoa(adminPort),
	})
	configOpts.Init(cmd)

	if err := cmd.Execute(); err != nil {
		i.t.Fatalf("cannot initialize horizon: %s", err)
	}

	if err := i.app.Ingestion().BuildGenesisState(); err != nil {
		i.t.Fatalf("cannot build genesis state: %s", err)
	}

	go i.app.Serve()
}

// Wait for core to be up and manually close the first ledger
func (i *Test) waitForCore() {
	for t := 30 * time.Second; t >= 0; t -= time.Second {
		i.t.Log("Waiting for core to be up...")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, err := i.cclient.Info(ctx)
		cancel()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	// We need to wait for Core to be armed before closing the first ledger
	// Otherwise, for some reason, the protocol version of the ledger stays at 0
	// TODO: instead of sleeping we should ensure Core's status (in GET /info) is "Armed"
	//       but, to do so, we should first expose it in Core's client.
	time.Sleep(time.Second)
	if err := i.CloseCoreLedger(); err != nil {
		i.t.Fatalf("Failed to manually close the second ledger: %s", err)
	}

	// Make sure that the Sleep above was successful
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	info, err := i.cclient.Info(ctx)
	cancel()
	if err != nil || !info.IsSynced() {
		log.Fatal("failed to wait for Core to be synced")
	}
}

func (i *Test) waitForIngestionAndUpgrade() {
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
func (i *Test) Client() *sdk.Client {
	return i.hclient
}

// LedgerIngested returns true if the ledger with a given sequence has been
// ingested by Horizon. Panics in case of errors.
func (i *Test) LedgerIngested(sequence uint32) bool {
	root, err := i.Client().Root()
	panicIf(err)

	return root.IngestSequence >= sequence
}

// LedgerClosed returns true if the ledger with a given sequence has been
// closed by Stellar-Core. Panics in case of errors. Note it's different
// than LedgerIngested because it checks if the ledger was closed, not
// necessarily ingested (ex. when rebuilding state Horizon does not ingest
// recent ledgers).
func (i *Test) LedgerClosed(sequence uint32) bool {
	root, err := i.Client().Root()
	panicIf(err)

	return root.CoreSequence >= int32(sequence)
}

// AdminPort returns Horizon admin port.
func (i *Test) AdminPort() int {
	return 6060
}

// Metrics URL returns Horizon metrics URL.
func (i *Test) MetricsURL() string {
	return fmt.Sprintf("http://localhost:%d/metrics", i.AdminPort())
}

// Master returns a keypair of the network master account.
func (i *Test) Master() *keypair.Full {
	return keypair.Master(NetworkPassphrase).(*keypair.Full)
}

func (i *Test) MasterAccount() txnbuild.Account {
	master, client := i.Master(), i.Client()
	request := sdk.AccountRequest{AccountID: master.Address()}
	account, err := client.AccountDetail(request)
	panicIf(err)
	return &account
}

func (i *Test) CurrentTest() *testing.T {
	return i.t
}

// Close stops and removes the docker container.
func (i *Test) Close() {
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

func createTestContainer(i *Test, image string) error {
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
		args.Add("reference", image)
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
			"--enable-core-manual-close",
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
		hostConfig.PortBindings = map[nat.Port][]nat.PortBinding{
			nat.Port("8000"): {{HostIP: "127.0.0.1", HostPort: "8000"}},
			nat.Port("6060"): {{HostIP: "127.0.0.1", HostPort: "6060"}},
		}
	} else {
		containerConfig.Env = append(containerConfig.Env,
			"POSTGRES_PASSWORD="+stellarCorePostgresPassword,
		)
		containerConfig.ExposedPorts = nat.PortSet{
			stellarCorePort:         struct{}{},
			stellarCorePostgresPort: struct{}{},
			historyArchivePort:      struct{}{},
		}
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
func (i *Test) CreateAccounts(count int, initialBalance string) ([]*keypair.Full, []txnbuild.Account) {
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
func (i *Test) MustEstablishTrustline(
	truster *keypair.Full, account txnbuild.Account, asset txnbuild.Asset,
) (resp proto.Transaction) {
	txResp, err := i.EstablishTrustline(truster, account, asset)
	panicIf(err)
	return txResp
}

// Establishes a trustline for a given asset for a particular account.
func (i *Test) EstablishTrustline(
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
func (i *Test) MustCreateClaimableBalance(
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
func (i *Test) MustGetAccount(source *keypair.Full) proto.Account {
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
func (i *Test) MustSubmitOperations(
	source txnbuild.Account, signer *keypair.Full, ops ...txnbuild.Operation,
) proto.Transaction {
	tx, err := i.SubmitOperations(source, signer, ops...)
	panicIf(err)
	return tx
}

func (i *Test) SubmitOperations(
	source txnbuild.Account, signer *keypair.Full, ops ...txnbuild.Operation,
) (proto.Transaction, error) {
	return i.SubmitMultiSigOperations(source, []*keypair.Full{signer}, ops...)
}

func (i *Test) SubmitMultiSigOperations(
	source txnbuild.Account, signers []*keypair.Full, ops ...txnbuild.Operation,
) (proto.Transaction, error) {
	tx, err := i.CreateSignedTransaction(source, signers, ops...)
	if err != nil {
		return proto.Transaction{}, err
	}
	return i.SubmitTransaction(tx)
}

func (i *Test) CreateSignedTransaction(
	source txnbuild.Account, signers []*keypair.Full, ops ...txnbuild.Operation,
) (*txnbuild.Transaction, error) {
	txParams := txnbuild.TransactionParams{
		SourceAccount:        source,
		Operations:           ops,
		BaseFee:              txnbuild.MinBaseFee,
		Timebounds:           txnbuild.NewInfiniteTimeout(),
		IncrementSequenceNum: true,
	}

	tx, err := txnbuild.NewTransaction(txParams)
	if err != nil {
		return nil, err
	}

	for _, signer := range signers {
		tx, err = tx.Sign(NetworkPassphrase, signer)
		if err != nil {
			return nil, err
		}
	}

	return tx, nil
}

// CloseCoreLedgersUntilSequence will close ledgers until sequence.
func (i *Test) CloseCoreLedgersUntilSequence(seq int) error {
	currentLedger, err := i.GetCurrentCoreLedgerSequence()
	if err != nil {
		return err
	}
	for ; currentLedger < seq; currentLedger++ {
		if err = i.CloseCoreLedger(); err != nil {
			return err
		}
	}
	return nil
}

// CloseCoreLedger will synchronously close at least one ledger.
// Note: because Core's manualclose endpoint doesn't block until ledger is actually
// closed, this method may end up closing multiple ledgers
func (i *Test) CloseCoreLedger() error {
	i.t.Log("Closing one ledger manually...")
	currentLedgerNum, err := i.GetCurrentCoreLedgerSequence()
	if err != nil {
		return err
	}
	targetLedgerNum := currentLedgerNum + 1
	// Core's manualclose endpoint doesn't currently block until the ledger is actually
	// closed. So, we loop until we are certain it happened.
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err = i.cclient.ManualClose(ctx)
		cancel()
		if err != nil {
			return err
		}
		currentLedgerNum, err = i.GetCurrentCoreLedgerSequence()
		if err != nil {
			return err
		}
		if currentLedgerNum >= targetLedgerNum {
			return nil
		}
		// pace ourselves
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

func (i *Test) GetCurrentCoreLedgerSequence() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	info, err := i.cclient.Info(ctx)
	if err != nil {
		return 0, err
	}
	return info.Info.Ledger.Num, nil
}

func (i *Test) SubmitTransaction(tx *txnbuild.Transaction) (proto.Transaction, error) {
	txb64, err := tx.Base64()
	if err != nil {
		return proto.Transaction{}, err
	}
	return i.SubmitTransactionXDR(txb64)
}

func (i *Test) SubmitTransactionXDR(txb64 string) (proto.Transaction, error) {
	// Core runs in manual-close mode to run tests faster, so we need to explicitly
	// close a ledger after the transaction is submitted.
	//
	// Horizon's submission endpoint blocks until the transaction is in a closed ledger.
	// Thus, we close the ledger in parallel to the submission.
	submissionDone := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// We manually-close in a loop to guarantee that (at some point)
		// a ledger-close happens after Core receives the transaction.
		// Otherwise there is a risk of the manual-close happening before the transaction
		// reaches Core, consequently causing the SubmitTransaction() call below to block indefinitely.
		//
		// This approach is ugly, but a better approach would probably require
		// instrumenting Horizon to tell us when the submission is done.
		for {
			time.Sleep(time.Millisecond * 100)
			if err := i.CloseCoreLedger(); err != nil {
				log.Fatalf("failed to CloseCoreLedger(): %s", err)
			}
			select {
			case <-submissionDone:
				// The transaction reached a closed-ledger!
				return
			default:
			}
		}
	}()
	tx, err := i.Client().SubmitTransactionXDR(txb64)
	close(submissionDone)
	wg.Wait()
	return tx, err
}

// A convenience function to provide verbose information about a failing
// transaction to the test output log, if it's expected to succeed.
func (i *Test) LogFailedTx(txResponse proto.Transaction, horizonResult error) {
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

func (i *Test) RunHorizonCLICommand(cmd ...string) {
	fullCmd := append([]string{"/stellar/horizon/bin/horizon"}, cmd...)
	id, err := i.cli.ContainerExecCreate(
		context.Background(),
		i.container.ID,
		types.ExecConfig{
			Cmd: fullCmd,
		},
	)
	panicIf(err)
	err = i.cli.ContainerExecStart(context.Background(), id.ID, types.ExecStartCheck{})
	panicIf(err)
}

// Cluttering code with if err != nil is absolute nonsense.
func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}
