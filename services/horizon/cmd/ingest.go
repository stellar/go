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
	"github.com/stellar/go/support/config"
	support "github.com/stellar/go/support/config"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/log"
)

var ingestBuildStateSequence uint32
var ingestBuildStateSkipChecks bool
var ingestVerifyFrom, ingestVerifyTo, ingestVerifyDebugServerPort uint32
var ingestVerifyState bool
var ingestVerifyLedgerBackendStr string
var ingestVerifyStorageBackendConfigPath string
var ingestVerifyLedgerBackendType ingest.LedgerBackendType
var processVerifyRangeFn = processVerifyRange

var ingestBuildStateCmdOpts = []*support.ConfigOption{
	{
		Name:        "sequence",
		ConfigKey:   &ingestBuildStateSequence,
		OptType:     types.Uint32,
		Required:    true,
		FlagDefault: uint32(0),
		Usage:       "checkpoint ledger sequence",
	},
	{
		Name:        "skip-checks",
		ConfigKey:   &ingestBuildStateSkipChecks,
		OptType:     types.Bool,
		Required:    false,
		FlagDefault: false,
		Usage:       "[optional] set to skip protocol version and bucket list hash verification, can speed up the process because does not require a running Stellar-Core",
	},
}

var ingestVerifyRangeCmdOpts = support.ConfigOptions{
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
	generateLedgerBackendOpt(&ingestVerifyLedgerBackendStr, &ingestVerifyLedgerBackendType),
	{
		Name:      "datastore-config",
		ConfigKey: &ingestVerifyStorageBackendConfigPath,
		OptType:   types.String,
		Required:  false,
		Usage:     "[optional] Specify the path to the datastore config file (required for datastore backend)",
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

func DefineIngestCommands(rootCmd *cobra.Command, horizonConfig *horizon.Config, horizonFlags config.ConfigOptions) {
	var ingestCmd = &cobra.Command{
		Use:   "ingest",
		Short: "ingestion related commands",
	}

	var ingestVerifyRangeCmd = &cobra.Command{
		Use:   "verify-range",
		Short: "runs ingestion pipeline within a range. warning! requires clean DB.",
		Long:  "runs ingestion pipeline between X and Y sequence number (inclusive)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ingestVerifyRangeCmdOpts.RequireE(); err != nil {
				return err
			}
			if err := ingestVerifyRangeCmdOpts.SetValues(); err != nil {
				return err
			}

			if err := horizon.ApplyFlags(horizonConfig, horizonFlags, horizon.ApplyOptions{RequireCaptiveCoreFullConfig: false}); err != nil {
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

			mngr := historyarchive.NewCheckpointManager(horizonConfig.CheckpointFrequency)
			if !mngr.IsCheckpoint(ingestVerifyFrom) && ingestVerifyFrom != 1 {
				return fmt.Errorf("`--from` must be a checkpoint ledger")
			}

			if ingestVerifyState && !mngr.IsCheckpoint(ingestVerifyTo) {
				return fmt.Errorf("`--to` must be a checkpoint ledger when `--verify-state` is set")
			}

			storageBackendConfig := ingest.StorageBackendConfig{}
			if ingestVerifyLedgerBackendType == ingest.BufferedStorageBackend {
				if ingestVerifyStorageBackendConfigPath == "" {
					return fmt.Errorf("datastore-config file path is required with datastore backend")
				}
				var err error
				if storageBackendConfig, err = loadStorageBackendConfig(ingestVerifyStorageBackendConfigPath); err != nil {
					return err
				}
			}

			err := processVerifyRangeFn(horizonConfig, horizonFlags, storageBackendConfig)
			if err != nil {
				return err
			}

			log.Info("Range run successfully!")
			return nil
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

			if err := horizon.ApplyFlags(horizonConfig, horizonFlags, horizon.ApplyOptions{RequireCaptiveCoreFullConfig: false}); err != nil {
				return err
			}

			horizonSession, err := db.Open("postgres", horizonConfig.DatabaseURL)
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
				NetworkPassphrase:      horizonConfig.NetworkPassphrase,
				HistorySession:         horizonSession,
				HistoryArchiveURLs:     horizonConfig.HistoryArchiveURLs,
				HistoryArchiveCaching:  horizonConfig.HistoryArchiveCaching,
				RoundingSlippageFilter: horizonConfig.RoundingSlippageFilter,
				CaptiveCoreBinaryPath:  horizonConfig.CaptiveCoreBinaryPath,
				CaptiveCoreConfigUseDB: horizonConfig.CaptiveCoreConfigUseDB,
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
			if err := horizon.ApplyFlags(horizonConfig, horizonFlags, horizon.ApplyOptions{RequireCaptiveCoreFullConfig: false}); err != nil {
				return err
			}

			horizonSession, err := db.Open("postgres", horizonConfig.DatabaseURL)
			if err != nil {
				return fmt.Errorf("cannot open Horizon DB: %v", err)
			}

			historyQ := &history.Q{SessionInterface: horizonSession}
			if err := historyQ.UpdateIngestVersion(ctx, 0); err != nil {
				return fmt.Errorf("cannot trigger state rebuild: %v", err)
			}

			log.Info("Triggered state rebuild")
			return nil
		},
	}

	var ingestBuildStateCmd = &cobra.Command{
		Use:   "build-state",
		Short: "builds state at a given checkpoint. warning! requires clean DB.",
		Long:  "useful for debugging or starting Horizon at specific checkpoint.",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, co := range ingestBuildStateCmdOpts {
				if err := co.RequireE(); err != nil {
					return err
				}
				co.SetValue()
			}

			if err := horizon.ApplyFlags(horizonConfig, horizonFlags, horizon.ApplyOptions{RequireCaptiveCoreFullConfig: false}); err != nil {
				return err
			}

			horizonSession, err := db.Open("postgres", horizonConfig.DatabaseURL)
			if err != nil {
				return fmt.Errorf("cannot open Horizon DB: %v", err)
			}

			historyQ := &history.Q{SessionInterface: horizonSession}

			lastIngestedLedger, err := historyQ.GetLastLedgerIngestNonBlocking(context.Background())
			if err != nil {
				return fmt.Errorf("cannot get last ledger value: %v", err)
			}

			if lastIngestedLedger != 0 {
				return fmt.Errorf("cannot run on non-empty DB")
			}

			mngr := historyarchive.NewCheckpointManager(globalConfig.CheckpointFrequency)
			if !mngr.IsCheckpoint(ingestBuildStateSequence) {
				return fmt.Errorf("`--sequence` must be a checkpoint ledger")
			}

			ingestConfig := ingest.Config{
				NetworkPassphrase:      horizonConfig.NetworkPassphrase,
				HistorySession:         horizonSession,
				HistoryArchiveURLs:     horizonConfig.HistoryArchiveURLs,
				HistoryArchiveCaching:  horizonConfig.HistoryArchiveCaching,
				CaptiveCoreBinaryPath:  horizonConfig.CaptiveCoreBinaryPath,
				CaptiveCoreConfigUseDB: horizonConfig.CaptiveCoreConfigUseDB,
				CheckpointFrequency:    horizonConfig.CheckpointFrequency,
				CaptiveCoreToml:        horizonConfig.CaptiveCoreToml,
				CaptiveCoreStoragePath: horizonConfig.CaptiveCoreStoragePath,
				RoundingSlippageFilter: horizonConfig.RoundingSlippageFilter,
			}

			system, err := ingest.NewSystem(ingestConfig)
			if err != nil {
				return err
			}

			err = system.BuildState(
				ingestBuildStateSequence,
				ingestBuildStateSkipChecks,
			)
			if err != nil {
				return err
			}

			log.Info("State built successfully!")
			return nil
		},
	}

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

	for _, co := range ingestBuildStateCmdOpts {
		err := co.Init(ingestBuildStateCmd)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	viper.BindPFlags(ingestVerifyRangeCmd.PersistentFlags())
	viper.BindPFlags(ingestBuildStateCmd.PersistentFlags())
	viper.BindPFlags(ingestStressTestCmd.PersistentFlags())

	rootCmd.AddCommand(ingestCmd)
	ingestCmd.AddCommand(
		ingestVerifyRangeCmd,
		ingestStressTestCmd,
		ingestTriggerStateRebuildCmd,
		ingestBuildStateCmd,
	)
}

func init() {
	DefineIngestCommands(RootCmd, globalConfig, globalFlags)
}

func processVerifyRange(horizonConfig *horizon.Config, horizonFlags config.ConfigOptions, backendConfig ingest.StorageBackendConfig) error {
	horizonSession, err := db.Open("postgres", horizonConfig.DatabaseURL)
	if err != nil {
		return fmt.Errorf("cannot open Horizon DB: %v", err)
	}

	ingestConfig := ingest.Config{
		NetworkPassphrase:      horizonConfig.NetworkPassphrase,
		HistorySession:         horizonSession,
		HistoryArchiveURLs:     horizonConfig.HistoryArchiveURLs,
		HistoryArchiveCaching:  horizonConfig.HistoryArchiveCaching,
		CaptiveCoreBinaryPath:  horizonConfig.CaptiveCoreBinaryPath,
		CaptiveCoreConfigUseDB: horizonConfig.CaptiveCoreConfigUseDB,
		CheckpointFrequency:    horizonConfig.CheckpointFrequency,
		CaptiveCoreToml:        horizonConfig.CaptiveCoreToml,
		CaptiveCoreStoragePath: horizonConfig.CaptiveCoreStoragePath,
		RoundingSlippageFilter: horizonConfig.RoundingSlippageFilter,
		LedgerBackendType:      ingestVerifyLedgerBackendType,
		StorageBackendConfig:   backendConfig,
	}

	system, err := ingest.NewSystem(ingestConfig)
	if err != nil {
		return err
	}

	return system.VerifyRange(
		ingestVerifyFrom,
		ingestVerifyTo,
		ingestVerifyState,
	)
}
