//lint:file-ignore U1001 Ignore all unused code, this is only used in tests.
package integration

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	sdk "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/clients/stellarcore"
	"github.com/stellar/go/keypair"
	proto "github.com/stellar/go/protocols/horizon"
	horizon "github.com/stellar/go/services/horizon/internal"
	"github.com/stellar/go/support/db/dbtest"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

const (
	NetworkPassphrase           = "Standalone Network ; February 2017"
	stellarCorePostgresPassword = "mysecretpassword"
	adminPort                   = 6060
	stellarCorePort             = 11626
	stellarCorePostgresPort     = 5641
	historyArchivePort          = 1570
)

type Config struct {
	ProtocolVersion       int32
	SkipContainerCreation bool
}

type Test struct {
	t       *testing.T
	config  Config
	hclient *sdk.Client
	cclient *stellarcore.Client
	app     *horizon.App
}

// NewTest starts a new environment for integration test at a given
// protocol version and blocks until Horizon starts ingesting.
//
// Warning: this requires Docker Compose installed
//
// Skips the test if HORIZON_INTEGRATION_TESTS env variable is not set.
func NewTest(t *testing.T, config Config) *Test {
	if os.Getenv("HORIZON_INTEGRATION_TESTS") == "" {
		t.Skip("skipping integration test")
	}

	i := &Test{t: t, config: config}

	composeDir := findDockerComposePath()
	manualCloseYaml := path.Join(composeDir, "docker-compose.integration-tests.yml")

	// Runs a docker-compose command applied to the above configs
	runComposeCommand := func(args ...string) {
		cmdline := append([]string{"-f", manualCloseYaml}, args...)
		t.Log("Running", cmdline)
		cmd := exec.Command("docker-compose", cmdline...)
		_, innerErr := cmd.Output()
		fatalIf(t, innerErr)
	}

	// Only run Stellar Core container and its dependencies
	runComposeCommand("up", "--detach", "--quiet-pull", "--no-color", "core")

	// FIXME: Only use horizon from quickstart container when testing captive core
	if os.Getenv("HORIZON_INTEGRATION_ENABLE_CAPTIVE_CORE") != "" {
		t.Skip("Testing with captive core isn't working yet.")
	}

	i.cclient = &stellarcore.Client{URL: "http://localhost:" + strconv.Itoa(stellarCorePort)}
	i.waitForCore()

	i.startHorizon()
	i.hclient = &sdk.Client{HorizonURL: "http://localhost:8000"}

	// Register cleanup handlers (on panic and ctrl+c) so the containers are
	// stopped even if ingestion or testing fails.
	cleanup := func() {
		if i.app != nil {
			i.app.Close()
		}
		runComposeCommand("down", "-v", "--remove-orphans")
	}
	i.t.Cleanup(cleanup)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cleanup()
		os.Exit(int(syscall.SIGTERM))
	}()

	i.waitForHorizon()
	return i
}

func (i *Test) startHorizon() {
	horizonPostgresURL := dbtest.Postgres(i.t).DSN

	config, configOpts := horizon.Flags()
	cmd := &cobra.Command{
		Use:   "horizon",
		Short: "client-facing api server for the stellar network",
		Long: `client-facing api server for the stellar network. It acts as the
interface between Stellar Core and applications that want to access the Stellar
network. It allows you to submit transactions to the network, check the status
of accounts, subscribe to event streams and more.`,
		Run: func(cmd *cobra.Command, args []string) {
			i.app = horizon.NewAppFromFlags(config, configOpts)
		},
	}

	// Ideally, we'd be pulling host/port information from the Docker Compose
	// YAML file itself rather than hardcoding it.
	hostname := "localhost"
	cmd.SetArgs([]string{
		"--stellar-core-url",
		fmt.Sprintf("http://%s:%d", hostname, stellarCorePort),
		"--history-archive-urls",
		fmt.Sprintf("http://%s:%d", hostname, historyArchivePort),
		"--ingest",
		"--db-url",
		horizonPostgresURL,
		"--stellar-core-db-url",
		fmt.Sprintf(
			"postgres://postgres:%s@%s:%d/stellar?sslmode=disable",
			stellarCorePostgresPassword,
			hostname,
			stellarCorePostgresPort,
		),
		"--network-passphrase",
		NetworkPassphrase,
		"--apply-migrations",
		"--admin-port",
		strconv.Itoa(i.AdminPort()),
	})
	var err error
	if err = configOpts.Init(cmd); err != nil {
		i.t.Fatalf("Cannot initialize params: %s", err)
	}

	if err = cmd.Execute(); err != nil {
		i.t.Fatalf("cannot initialize horizon: %s", err)
	}

	if err = i.app.Ingestion().BuildGenesisState(); err != nil {
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
		if err != nil {
			i.t.Logf("could not obtain info response: %v", err)
			time.Sleep(time.Second)
			continue
		}
		break
	}

	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := i.cclient.Upgrade(ctx, int(i.config.ProtocolVersion))
		cancel()
		if err != nil {
			i.t.Fatalf("could not upgrade protocol: %v", err)
		}
	}

	if err := i.CloseCoreLedger(); err != nil {
		i.t.Fatalf("Failed to manually close the second ledger: %s", err)
	}

	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		info, err := i.cclient.Info(ctx)
		cancel()
		if err != nil || !info.IsSynced() {
			i.t.Fatal("failed to wait for Core to be synced")
		}
	}
}

func (i *Test) waitForHorizon() {
	for t := 30; t >= 0; t -= 1 {
		i.t.Log("Waiting for ingestion and protocol upgrade...")
		root, err := i.hclient.Root()
		if err != nil {
			i.t.Logf("could not obtain root response %v", err)
			time.Sleep(time.Second)
			continue
		}

		if root.HorizonSequence < 2 ||
			int(root.HorizonSequence) != int(root.IngestSequence) ||
			root.HorizonSequence < root.CoreSequence {
			i.t.Logf("Horizon ingesting... %v", root)
			time.Sleep(time.Second)
			continue
		}

		if root.CurrentProtocolVersion == i.config.ProtocolVersion {
			i.t.Logf("Horizon protocol version matches... %v", root)
			return
		}
	}

	i.t.Fatal("Horizon not ingesting...")
}

// Client returns horizon.Client connected to started Horizon instance.
func (i *Test) Client() *sdk.Client {
	return i.hclient
}

// Horizon returns the horizon.App instance for the current integration test
func (i *Test) Horizon() *horizon.App {
	return i.app
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
	return adminPort
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

// Cluttering code with if err != nil is absolute nonsense.
func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}

func fatalIf(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}

// Performs a best-effort attempt to find the project's Docker Compose files.
func findDockerComposePath() string {
	// Lets you check if a particular directory contains a file.
	directoryContainsFilename := func(dir string, filename string) bool {
		files, innerErr := ioutil.ReadDir(dir)
		panicIf(innerErr)

		for _, file := range files {
			if file.Name() == filename {
				return true
			}
		}

		return false
	}

	current, err := os.Getwd()
	panicIf(err)

	//
	// We have a primary and backup attempt for finding the necessary docker
	// files: via $GOPATH and via local directory traversal.
	//

	if gopath := os.Getenv("GOPATH"); gopath != "" {
		monorepo := path.Join(gopath, "stellar", "go")
		if _, err = os.Stat(monorepo); !os.IsNotExist(err) {
			current = monorepo
		}
	}

	// In either case, we try to walk up the tree until we find "go.mod",
	// which we hope is the root directory of the project.
	for !directoryContainsFilename(current, "go.mod") {
		current, err = filepath.Abs(path.Join(current, ".."))

		// FIXME: This only works on *nix-like systems.
		if err != nil || filepath.Base(current)[0] == filepath.Separator {
			fmt.Println("Failed to establish project root directory.")
			panic(err)
		}
	}

	// Directly jump down to the folder that should contain the configs
	return path.Join(current, "services", "horizon", "docker")
}
