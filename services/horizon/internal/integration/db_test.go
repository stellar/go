package integration

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	horizon "github.com/stellar/go/services/horizon/internal"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/historyarchive"
	horizoncmd "github.com/stellar/go/services/horizon/cmd"
	"github.com/stellar/go/services/horizon/internal/db2/schema"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/db/dbtest"
	"github.com/stellar/go/txnbuild"
)

func generateLiquidityPoolOps(itest *integration.Test, tt *assert.Assertions) (lastLedger int32) {

	master := itest.Master()
	keys, accounts := itest.CreateAccounts(2, "1000")
	shareKeys, shareAccount := keys[0], accounts[0]
	tradeKeys, tradeAccount := keys[1], accounts[1]

	itest.MustSubmitMultiSigOperations(shareAccount, []*keypair.Full{shareKeys, master},
		&txnbuild.ChangeTrust{
			Line: txnbuild.ChangeTrustAssetWrapper{
				Asset: txnbuild.CreditAsset{
					Code:   "USD",
					Issuer: master.Address(),
				},
			},
			Limit: txnbuild.MaxTrustlineLimit,
		},
		&txnbuild.ChangeTrust{
			Line: txnbuild.LiquidityPoolShareChangeTrustAsset{
				LiquidityPoolParameters: txnbuild.LiquidityPoolParameters{
					AssetA: txnbuild.NativeAsset{},
					AssetB: txnbuild.CreditAsset{
						Code:   "USD",
						Issuer: master.Address(),
					},
					Fee: 30,
				},
			},
			Limit: txnbuild.MaxTrustlineLimit,
		},
		&txnbuild.Payment{
			SourceAccount: master.Address(),
			Destination:   shareAccount.GetAccountID(),
			Asset: txnbuild.CreditAsset{
				Code:   "USD",
				Issuer: master.Address(),
			},
			Amount: "1000",
		},
	)

	poolID, err := xdr.NewPoolId(
		xdr.MustNewNativeAsset(),
		xdr.MustNewCreditAsset("USD", master.Address()),
		30,
	)
	tt.NoError(err)
	poolIDHexString := xdr.Hash(poolID).HexString()

	itest.MustSubmitOperations(shareAccount, shareKeys,
		&txnbuild.LiquidityPoolDeposit{
			LiquidityPoolID: [32]byte(poolID),
			MaxAmountA:      "400",
			MaxAmountB:      "777",
			MinPrice:        "0.5",
			MaxPrice:        "2",
		},
	)

	itest.MustSubmitOperations(tradeAccount, tradeKeys,
		&txnbuild.ChangeTrust{
			Line: txnbuild.ChangeTrustAssetWrapper{
				Asset: txnbuild.CreditAsset{
					Code:   "USD",
					Issuer: master.Address(),
				},
			},
			Limit: txnbuild.MaxTrustlineLimit,
		},
		&txnbuild.PathPaymentStrictReceive{
			SendAsset: txnbuild.NativeAsset{},
			DestAsset: txnbuild.CreditAsset{
				Code:   "USD",
				Issuer: master.Address(),
			},
			SendMax:     "1000",
			DestAmount:  "2",
			Destination: tradeKeys.Address(),
		},
	)

	pool, err := itest.Client().LiquidityPoolDetail(horizonclient.LiquidityPoolRequest{
		LiquidityPoolID: poolIDHexString,
	})
	tt.NoError(err)

	txResp := itest.MustSubmitOperations(shareAccount, shareKeys,
		&txnbuild.LiquidityPoolWithdraw{
			LiquidityPoolID: [32]byte(poolID),
			Amount:          pool.TotalShares,
			MinAmountA:      "10",
			MinAmountB:      "20",
		},
	)

	return txResp.Ledger
}

func generatePaymentOps(itest *integration.Test, tt *assert.Assertions) (lastLedger int32) {
	txResp := itest.MustSubmitOperations(itest.MasterAccount(), itest.Master(),
		&txnbuild.Payment{
			Destination: itest.Master().Address(),
			Amount:      "10",
			Asset:       txnbuild.NativeAsset{},
		},
	)

	return txResp.Ledger
}

func initializeDBIntegrationTest(t *testing.T) (itest *integration.Test, reachedLedger int32) {
	config := integration.Config{ProtocolVersion: 18}
	itest = integration.NewTest(t, config)
	tt := assert.New(t)

	generatePaymentOps(itest, tt)
	reachedLedger = generateLiquidityPoolOps(itest, tt)

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
	tt.NoError(err)
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

	horizoncmd.RootCmd.SetArgs(command(horizonConfig, "db",
		"reingest",
		"range",
		"1",
		fmt.Sprintf("%d", toLedger),
	))

	tt.NoError(horizoncmd.RootCmd.Execute())
	tt.NoError(horizoncmd.RootCmd.Execute(), "Repeat the same reingest range against db, should not have errors.")
}

func command(horizonConfig horizon.Config, args ...string) []string {
	return append([]string{
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
	}, args...)
}

func TestFillGaps(t *testing.T) {
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

	historyQ := history.Q{dbConn}
	var oldestLedger, latestLedger int64
	tt.NoError(historyQ.ElderLedger(context.Background(), &oldestLedger))
	tt.NoError(historyQ.LatestLedger(context.Background(), &latestLedger))
	tt.NoError(historyQ.DeleteRangeAll(context.Background(), oldestLedger, latestLedger))

	horizonConfig.CaptiveCoreConfigPath = filepath.Join(
		filepath.Dir(horizonConfig.CaptiveCoreConfigPath),
		"captive-core-reingest-range-integration-tests.cfg",
	)
	horizoncmd.RootCmd.SetArgs(command(horizonConfig, "db", "fill-gaps"))
	tt.NoError(horizoncmd.RootCmd.Execute())

	tt.NoError(historyQ.LatestLedger(context.Background(), &latestLedger))
	tt.Equal(int64(0), latestLedger)

	horizoncmd.RootCmd.SetArgs(command(horizonConfig, "db", "fill-gaps", "3", "4"))
	tt.NoError(horizoncmd.RootCmd.Execute())
	tt.NoError(historyQ.LatestLedger(context.Background(), &latestLedger))
	tt.NoError(historyQ.ElderLedger(context.Background(), &oldestLedger))
	tt.Equal(int64(3), oldestLedger)
	tt.Equal(int64(4), latestLedger)

	horizoncmd.RootCmd.SetArgs(command(horizonConfig, "db", "fill-gaps", "6", "7"))
	tt.NoError(horizoncmd.RootCmd.Execute())
	tt.NoError(historyQ.LatestLedger(context.Background(), &latestLedger))
	tt.NoError(historyQ.ElderLedger(context.Background(), &oldestLedger))
	tt.Equal(int64(3), oldestLedger)
	tt.Equal(int64(7), latestLedger)
	var gaps []history.LedgerRange
	gaps, err = historyQ.GetLedgerGaps(context.Background())
	tt.NoError(err)
	tt.Equal([]history.LedgerRange{{StartSequence: 5, EndSequence: 5}}, gaps)

	horizoncmd.RootCmd.SetArgs(command(horizonConfig, "db", "fill-gaps"))
	tt.NoError(horizoncmd.RootCmd.Execute())
	tt.NoError(historyQ.LatestLedger(context.Background(), &latestLedger))
	tt.NoError(historyQ.ElderLedger(context.Background(), &oldestLedger))
	tt.Equal(int64(3), oldestLedger)
	tt.Equal(int64(7), latestLedger)
	gaps, err = historyQ.GetLedgerGaps(context.Background())
	tt.NoError(err)
	tt.Empty(gaps)

	horizoncmd.RootCmd.SetArgs(command(horizonConfig, "db", "fill-gaps", "2", "8"))
	tt.NoError(horizoncmd.RootCmd.Execute())
	tt.NoError(historyQ.LatestLedger(context.Background(), &latestLedger))
	tt.NoError(historyQ.ElderLedger(context.Background(), &oldestLedger))
	tt.Equal(int64(2), oldestLedger)
	tt.Equal(int64(8), latestLedger)
	gaps, err = historyQ.GetLedgerGaps(context.Background())
	tt.NoError(err)
	tt.Empty(gaps)
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
