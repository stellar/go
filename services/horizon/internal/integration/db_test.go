package integration

import (
	"fmt"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/historyarchive"
	horizoncmd "github.com/stellar/go/services/horizon/cmd"
	"github.com/stellar/go/services/horizon/internal/db2/schema"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/db/dbtest"
	"github.com/stellar/go/txnbuild"
)

func initializeDBIntegrationTest(t *testing.T) (itest *integration.Test, reachedLedger int32) {
	itest = integration.NewTest(t, protocol15Config)
	master := itest.Master()
	tt := assert.New(t)

	// Initialize the database with some ledgers including some transactions we submit
	op := txnbuild.Payment{
		Destination: master.Address(),
		Amount:      "10",
		Asset:       txnbuild.NativeAsset{},
	}
	// TODO: should we enforce certain number of ledgers to be ingested?
	for i := 0; i < 8; i++ {
		txResp := itest.MustSubmitOperations(itest.MasterAccount(), master, &op)
		reachedLedger = txResp.Ledger
	}

	root, err := itest.Client().Root()
	tt.NoError(err)
	tt.LessOrEqual(reachedLedger, root.HorizonSequence)

	return
}

func TestReingestDB(t *testing.T) {
	itest, reachedLedger := initializeDBIntegrationTest(t)
	tt := assert.New(t)

	// Create a fresh Horizon database
	newDB := dbtest.Postgres(t)
	// TODO: Unfortunately Horizon's ingestion System leaves open sessions behind,leading to
	//       a "database  is being accessed by other users" error when trying to drop it
	// defer newDB.Close()
	freshHorizonPostgresURL := newDB.DSN
	horizonConfig := itest.GetHorizonConfig()
	horizonConfig.DatabaseURL = freshHorizonPostgresURL
	// Initialize the DB schema
	dbConn, err := db.Open("postgres", freshHorizonPostgresURL)
	defer dbConn.Close()
	_, err = schema.Migrate(dbConn.DB.DB, schema.MigrateUp, 0)
	tt.NoError(err)

	// cap reachedLedger to the nearest checkpoint ledger because reingest range cannot ingest past the most
	// recent checkpoint ledger when using captive core
	toLedger := uint32(reachedLedger)
	archive, err := historyarchive.Connect(horizonConfig.HistoryArchiveURLs[0], historyarchive.ConnectOptions{
		NetworkPassphrase:   horizonConfig.NetworkPassphrase,
		CheckpointFrequency: horizonConfig.CheckpointFrequency,
	})
	tt.NoError(err)

	// make sure a full checkpoint has elapsed otherwise there will be nothing to reingest
	var latestCheckpoint uint32
	publishedFirstCheckpoint := func() bool {
		has, requestErr := archive.GetRootHAS()
		tt.NoError(requestErr)
		latestCheckpoint = has.CurrentLedger
		return latestCheckpoint > 1
	}
	tt.Eventually(publishedFirstCheckpoint, 10*time.Second, time.Second)

	if toLedger > latestCheckpoint {
		toLedger = latestCheckpoint
	}

	// We just want to test reingestion, so there's no reason for a background
	// Horizon to run. Keeping it running will actually cause the Captive Core
	// subprocesses to conflict.
	itest.StopHorizon()

	horizonConfig.CaptiveCoreConfigPath = filepath.Join(
		filepath.Dir(horizonConfig.CaptiveCoreConfigPath),
		"captive-core-reingest-range-integration-tests.cfg",
	)

	horizoncmd.RootCmd.SetArgs([]string{
		"--stellar-core-url",
		horizonConfig.StellarCoreURL,
		"--history-archive-urls",
		horizonConfig.HistoryArchiveURLs[0],
		"--db-url",
		horizonConfig.DatabaseURL,
		"--stellar-core-db-url",
		horizonConfig.StellarCoreDatabaseURL,
		"--stellar-core-binary-path",
		horizonConfig.CaptiveCoreBinaryPath,
		"--captive-core-config-path",
		horizonConfig.CaptiveCoreConfigPath,
		"--enable-captive-core-ingestion=" + strconv.FormatBool(horizonConfig.EnableCaptiveCoreIngestion),
		"--network-passphrase",
		horizonConfig.NetworkPassphrase,
		// due to ARTIFICIALLY_ACCELERATE_TIME_FOR_TESTING
		"--checkpoint-frequency",
		"8",
		"db",
		"reingest",
		"range",
		"1",
		fmt.Sprintf("%d", toLedger),
	})

	err = horizoncmd.RootCmd.Execute()
	// Reingest into the DB
	tt.NoError(err)
}

func TestResumeFromInitializedDB(t *testing.T) {
	itest, reachedLedger := initializeDBIntegrationTest(t)
	tt := assert.New(t)

	// Stop the integration test, and restart it with the same database
	oldDBURL := itest.GetHorizonConfig().DatabaseURL
	itestConfig := protocol15Config
	itestConfig.PostgresURL = oldDBURL

	err := itest.RestartHorizon()
	tt.NoError(err)

	successfullyResumed := func() bool {
		root, err := itest.Client().Root()
		tt.NoError(err)
		// It must be able to reach the ledger and surpass it
		const ledgersPastStopPoint = 4
		return root.HorizonSequence > (reachedLedger + ledgersPastStopPoint)
	}

	tt.Eventually(successfullyResumed, 1*time.Minute, 1*time.Second)
}
