package cmd

import (
	"context"
	"fmt"
	"go/types"
	"net/http"
	_ "net/http/pprof"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stellar/go/historyarchive"
	horizon "github.com/stellar/go/services/horizon/internal"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest"
	support "github.com/stellar/go/support/config"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/log"
)

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "ingestion related commands",
}

var ingestVerifyFrom, ingestVerifyTo, ingestVerifyDebugServerPort uint32
var ingestVerifyState bool

var ingestVerifyRangeCmdOpts = []*support.ConfigOption{
	{
		Name:        "from",
		ConfigKey:   &ingestVerifyFrom,
		OptType:     types.Uint32,
		Required:    true,
		FlagDefault: uint32(0),
		Usage:       "first ledger of the range to ingest",
	},
	{
		Name:        "to",
		ConfigKey:   &ingestVerifyTo,
		OptType:     types.Uint32,
		Required:    true,
		FlagDefault: uint32(0),
		Usage:       "last ledger of the range to ingest",
	},
	{
		Name:        "verify-state",
		ConfigKey:   &ingestVerifyState,
		OptType:     types.Bool,
		Required:    false,
		FlagDefault: false,
		Usage:       "[optional] verifies state at the last ledger of the range when true",
	},
	{
		Name:        "debug-server-port",
		ConfigKey:   &ingestVerifyDebugServerPort,
		OptType:     types.Uint32,
		Required:    false,
		FlagDefault: uint32(0),
		Usage:       "[optional] opens a net/http/pprof server at given port",
	},
}

var ingestVerifyRangeCmd = &cobra.Command{
	Use:   "verify-range",
	Short: "runs ingestion pipeline within a range. warning! requires clean DB.",
	Long:  "runs ingestion pipeline between X and Y sequence number (inclusive)",
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, co := range ingestVerifyRangeCmdOpts {
			if err := co.RequireE(); err != nil {
				return err
			}
			co.SetValue()
		}

		if err := horizon.ApplyFlags(config, flags, horizon.ApplyOptions{RequireCaptiveCoreConfig: false, AlwaysIngest: true}); err != nil {
			return err
		}

		if ingestVerifyDebugServerPort != 0 {
			go func() {
				log.Infof("Starting debug server at: %d", ingestVerifyDebugServerPort)
				err := http.ListenAndServe(
					fmt.Sprintf("localhost:%d", ingestVerifyDebugServerPort),
					nil,
				)
				if err != nil {
					log.Error(err)
				}
			}()
		}

		horizonSession, err := db.Open("postgres", config.DatabaseURL)
		if err != nil {
			return fmt.Errorf("cannot open Horizon DB: %v", err)
		}
		mngr := historyarchive.NewCheckpointManager(config.CheckpointFrequency)
		if !mngr.IsCheckpoint(ingestVerifyFrom) && ingestVerifyFrom != 1 {
			return fmt.Errorf("`--from` must be a checkpoint ledger")
		}

		if ingestVerifyState && !mngr.IsCheckpoint(ingestVerifyTo) {
			return fmt.Errorf("`--to` must be a checkpoint ledger when `--verify-state` is set.")
		}

		ingestConfig := ingest.Config{
			NetworkPassphrase:        config.NetworkPassphrase,
			HistorySession:           horizonSession,
			HistoryArchiveURL:        config.HistoryArchiveURLs[0],
			EnableCaptiveCore:        config.EnableCaptiveCoreIngestion,
			CaptiveCoreBinaryPath:    config.CaptiveCoreBinaryPath,
			CaptiveCoreConfigUseDB:   config.CaptiveCoreConfigUseDB,
			RemoteCaptiveCoreURL:     config.RemoteCaptiveCoreURL,
			CheckpointFrequency:      config.CheckpointFrequency,
			CaptiveCoreToml:          config.CaptiveCoreToml,
			CaptiveCoreStoragePath:   config.CaptiveCoreStoragePath,
			RoundingSlippageFilter:   config.RoundingSlippageFilter,
			EnableIngestionFiltering: config.EnableIngestionFiltering,
		}

		if !ingestConfig.EnableCaptiveCore {
			if config.StellarCoreDatabaseURL == "" {
				return fmt.Errorf("flag --%s cannot be empty", horizon.StellarCoreDBURLFlagName)
			}

			coreSession, dbErr := db.Open("postgres", config.StellarCoreDatabaseURL)
			if dbErr != nil {
				return fmt.Errorf("cannot open Core DB: %v", dbErr)
			}
			ingestConfig.CoreSession = coreSession
		}

		system, err := ingest.NewSystem(ingestConfig)
		if err != nil {
			return err
		}

		err = system.VerifyRange(
			ingestVerifyFrom,
			ingestVerifyTo,
			ingestVerifyState,
		)
		if err != nil {
			return err
		}

		log.Info("Range run successfully!")
		return nil
	},
}

var stressTestNumTransactions, stressTestChangesPerTransaction int

var stressTestCmdOpts = []*support.ConfigOption{
	{
		Name:        "transactions",
		ConfigKey:   &stressTestNumTransactions,
		OptType:     types.Int,
		Required:    false,
		FlagDefault: int(1000),
		Usage:       "total number of transactions to ingest (at most 1000)",
	},
	{
		Name:        "changes",
		ConfigKey:   &stressTestChangesPerTransaction,
		OptType:     types.Int,
		Required:    false,
		FlagDefault: int(4000),
		Usage:       "changes per transaction to ingest (at most 4000)",
	},
}

var ingestStressTestCmd = &cobra.Command{
	Use:   "stress-test",
	Short: "runs ingestion pipeline on a ledger with many changes. warning! requires clean DB.",
	Long:  "runs ingestion pipeline on a ledger with many changes. warning! requires clean DB.",
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, co := range stressTestCmdOpts {
			if err := co.RequireE(); err != nil {
				return err
			}
			co.SetValue()
		}

		if err := horizon.ApplyFlags(config, flags, horizon.ApplyOptions{RequireCaptiveCoreConfig: false, AlwaysIngest: true}); err != nil {
			return err
		}

		horizonSession, err := db.Open("postgres", config.DatabaseURL)
		if err != nil {
			return fmt.Errorf("cannot open Horizon DB: %v", err)
		}

		if stressTestNumTransactions <= 0 {
			return fmt.Errorf("`--transactions` must be positive")
		}

		if stressTestChangesPerTransaction <= 0 {
			return fmt.Errorf("`--changes` must be positive")
		}

		ingestConfig := ingest.Config{
			NetworkPassphrase:      config.NetworkPassphrase,
			HistorySession:         horizonSession,
			HistoryArchiveURL:      config.HistoryArchiveURLs[0],
			EnableCaptiveCore:      config.EnableCaptiveCoreIngestion,
			RoundingSlippageFilter: config.RoundingSlippageFilter,
		}

		if config.EnableCaptiveCoreIngestion {
			ingestConfig.CaptiveCoreBinaryPath = config.CaptiveCoreBinaryPath
			ingestConfig.RemoteCaptiveCoreURL = config.RemoteCaptiveCoreURL
			ingestConfig.CaptiveCoreConfigUseDB = config.CaptiveCoreConfigUseDB
		} else {
			if config.StellarCoreDatabaseURL == "" {
				return fmt.Errorf("flag --%s cannot be empty", horizon.StellarCoreDBURLFlagName)
			}

			coreSession, dbErr := db.Open("postgres", config.StellarCoreDatabaseURL)
			if dbErr != nil {
				return fmt.Errorf("cannot open Core DB: %v", dbErr)
			}
			ingestConfig.CoreSession = coreSession
		}

		system, err := ingest.NewSystem(ingestConfig)
		if err != nil {
			return err
		}

		err = system.StressTest(
			stressTestNumTransactions,
			stressTestChangesPerTransaction,
		)
		if err != nil {
			return err
		}

		log.Info("Stress test completed successfully!")
		return nil
	},
}

var ingestTriggerStateRebuildCmd = &cobra.Command{
	Use:   "trigger-state-rebuild",
	Short: "updates a database to trigger state rebuild, state will be rebuilt by a running Horizon instance, DO NOT RUN production DB, some endpoints will be unavailable until state is rebuilt",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if err := horizon.ApplyFlags(config, flags, horizon.ApplyOptions{RequireCaptiveCoreConfig: false, AlwaysIngest: true}); err != nil {
			return err
		}

		horizonSession, err := db.Open("postgres", config.DatabaseURL)
		if err != nil {
			return fmt.Errorf("cannot open Horizon DB: %v", err)
		}

		historyQ := &history.Q{horizonSession}
		if err := historyQ.UpdateIngestVersion(ctx, 0); err != nil {
			return fmt.Errorf("cannot trigger state rebuild: %v", err)
		}

		log.Info("Triggered state rebuild")
		return nil
	},
}

var ingestInitGenesisStateCmd = &cobra.Command{
	Use:   "init-genesis-state",
	Short: "ingests genesis state (ledger 1)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if err := horizon.ApplyFlags(config, flags, horizon.ApplyOptions{RequireCaptiveCoreConfig: false, AlwaysIngest: true}); err != nil {
			return err
		}

		horizonSession, err := db.Open("postgres", config.DatabaseURL)
		if err != nil {
			return fmt.Errorf("cannot open Horizon DB: %v", err)
		}

		historyQ := &history.Q{horizonSession}

		lastIngestedLedger, err := historyQ.GetLastLedgerIngestNonBlocking(ctx)
		if err != nil {
			return fmt.Errorf("cannot get last ledger value: %v", err)
		}

		if lastIngestedLedger != 0 {
			return fmt.Errorf("cannot run on non-empty DB")
		}

		ingestConfig := ingest.Config{
			NetworkPassphrase:        config.NetworkPassphrase,
			HistorySession:           horizonSession,
			HistoryArchiveURL:        config.HistoryArchiveURLs[0],
			EnableCaptiveCore:        config.EnableCaptiveCoreIngestion,
			CheckpointFrequency:      config.CheckpointFrequency,
			RoundingSlippageFilter:   config.RoundingSlippageFilter,
			EnableIngestionFiltering: config.EnableIngestionFiltering,
		}

		if config.EnableCaptiveCoreIngestion {
			ingestConfig.CaptiveCoreBinaryPath = config.CaptiveCoreBinaryPath
			ingestConfig.CaptiveCoreConfigUseDB = config.CaptiveCoreConfigUseDB
		} else {
			if config.StellarCoreDatabaseURL == "" {
				return fmt.Errorf("flag --%s cannot be empty", horizon.StellarCoreDBURLFlagName)
			}

			coreSession, dbErr := db.Open("postgres", config.StellarCoreDatabaseURL)
			if dbErr != nil {
				return fmt.Errorf("cannot open Core DB: %v", dbErr)
			}
			ingestConfig.CoreSession = coreSession
		}

		system, err := ingest.NewSystem(ingestConfig)
		if err != nil {
			return err
		}

		err = system.BuildGenesisState()
		if err != nil {
			return err
		}

		log.Info("Genesis ledger stat successfully ingested!")
		return nil
	},
}

func init() {
	for _, co := range ingestVerifyRangeCmdOpts {
		err := co.Init(ingestVerifyRangeCmd)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	for _, co := range stressTestCmdOpts {
		err := co.Init(ingestStressTestCmd)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	viper.BindPFlags(ingestVerifyRangeCmd.PersistentFlags())

	RootCmd.AddCommand(ingestCmd)
	ingestCmd.AddCommand(
		ingestVerifyRangeCmd,
		ingestStressTestCmd,
		ingestTriggerStateRebuildCmd,
		ingestInitGenesisStateCmd,
	)
}
