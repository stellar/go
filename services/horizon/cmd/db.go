package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"go/types"
	"log"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stellar/go/services/horizon/internal/db2/history"

	horizon "github.com/stellar/go/services/horizon/internal"
	"github.com/stellar/go/services/horizon/internal/db2/schema"
	"github.com/stellar/go/services/horizon/internal/ingest"
	support "github.com/stellar/go/support/config"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	hlog "github.com/stellar/go/support/log"
)

var dbCmd = &cobra.Command{
	Use:   "db [command]",
	Short: "commands to manage horizon's postgres db",
}

var dbMigrateCmd = &cobra.Command{
	Use:   "migrate [command]",
	Short: "commands to run schema migrations on horizon's postgres db",
}

func requireAndSetFlag(name string) {
	for _, flag := range flags {
		if flag.Name == name {
			flag.Require()
			flag.SetValue()
			return
		}
	}
	log.Fatalf("could not find %s flag", name)
}

var dbInitCmd = &cobra.Command{
	Use:   "init",
	Short: "install schema",
	Long:  "init initializes the postgres database used by horizon.",
	Run: func(cmd *cobra.Command, args []string) {
		requireAndSetFlag(horizon.DatabaseURLFlagName)

		db, err := sql.Open("postgres", config.DatabaseURL)
		if err != nil {
			log.Fatal(err)
		}

		numMigrationsRun, err := schema.Migrate(db, schema.MigrateUp, 0)
		if err != nil {
			log.Fatal(err)
		}

		if numMigrationsRun == 0 {
			log.Println("No migrations applied.")
		} else {
			log.Printf("Successfully applied %d migrations.\n", numMigrationsRun)
		}
	},
}

func migrate(dir schema.MigrateDir, count int) {
	dbConn, err := db.Open("postgres", config.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	numMigrationsRun, err := schema.Migrate(dbConn.DB.DB, dir, count)
	if err != nil {
		log.Fatal(err)
	}

	if numMigrationsRun == 0 {
		log.Println("No migrations applied.")
	} else {
		log.Printf("Successfully applied %d migrations.\n", numMigrationsRun)
	}
}

var dbMigrateDownCmd = &cobra.Command{
	Use:   "down COUNT",
	Short: "run upwards db schema migrations",
	Long:  "performs a downards schema migration command",
	Run: func(cmd *cobra.Command, args []string) {
		requireAndSetFlag(horizon.DatabaseURLFlagName)

		// Only allow invokations with 1 args.
		if len(args) != 1 {
			cmd.Usage()
			os.Exit(1)
		}

		count, err := strconv.Atoi(args[0])
		if err != nil {
			log.Println(err)
			cmd.Usage()
			os.Exit(1)
		}

		migrate(schema.MigrateDown, count)
	},
}

var dbMigrateRedoCmd = &cobra.Command{
	Use:   "redo COUNT",
	Short: "redo db schema migrations",
	Long:  "performs a redo schema migration command",
	Run: func(cmd *cobra.Command, args []string) {
		requireAndSetFlag(horizon.DatabaseURLFlagName)

		// Only allow invokations with 1 args.
		if len(args) != 1 {
			cmd.Usage()
			os.Exit(1)
		}

		count, err := strconv.Atoi(args[0])
		if err != nil {
			log.Println(err)
			cmd.Usage()
			os.Exit(1)
		}

		migrate(schema.MigrateRedo, count)
	},
}

var dbMigrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "print current database migration status",
	Long:  "print current database migration status",
	Run: func(cmd *cobra.Command, args []string) {
		requireAndSetFlag(horizon.DatabaseURLFlagName)

		// Only allow invokations with 0 args.
		if len(args) != 0 {
			fmt.Println(args)
			cmd.Usage()
			os.Exit(1)
		}

		dbConn, err := db.Open("postgres", config.DatabaseURL)
		if err != nil {
			log.Fatal(err)
		}

		status, err := schema.Status(dbConn.DB.DB)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(status)
	},
}

var dbMigrateUpCmd = &cobra.Command{
	Use:   "up [COUNT]",
	Short: "run upwards db schema migrations",
	Long:  "performs an upwards schema migration command",
	Run: func(cmd *cobra.Command, args []string) {
		requireAndSetFlag(horizon.DatabaseURLFlagName)

		// Only allow invokations with 0-1 args.
		if len(args) > 1 {
			cmd.Usage()
			os.Exit(1)
		}

		count := 0
		if len(args) == 1 {
			var err error
			count, err = strconv.Atoi(args[0])
			if err != nil {
				log.Println(err)
				cmd.Usage()
				os.Exit(1)
			}
		}

		migrate(schema.MigrateUp, count)
	},
}

var dbReapCmd = &cobra.Command{
	Use:   "reap",
	Short: "reaps (i.e. removes) any reapable history data",
	Long:  "reap removes any historical data that is earlier than the configured retention cutoff",
	Run: func(cmd *cobra.Command, args []string) {
		app := horizon.NewAppFromFlags(config, flags)
		ctx := context.Background()
		app.UpdateLedgerState(ctx)
		err := app.DeleteUnretainedHistory(ctx)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var dbReingestCmd = &cobra.Command{
	Use:   "reingest",
	Short: "reingest commands",
	Long:  "reingest ingests historical data for every ledger or ledgers specified by subcommand",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use one of the subcomands...")
		cmd.Usage()
		os.Exit(1)
	},
}

var (
	reingestForce       bool
	parallelWorkers     uint
	parallelJobSize     uint32
	retries             uint
	retryBackoffSeconds uint
)
var reingestRangeCmdOpts = []*support.ConfigOption{
	{
		Name:        "force",
		ConfigKey:   &reingestForce,
		OptType:     types.Bool,
		Required:    false,
		FlagDefault: false,
		Usage: "[optional] if this flag is set, horizon will be blocked " +
			"from ingesting until the reingestion command completes (incompatible with --parallel-workers > 1)",
	},
	{
		Name:        "parallel-workers",
		ConfigKey:   &parallelWorkers,
		OptType:     types.Uint,
		Required:    false,
		FlagDefault: uint(1),
		Usage:       "[optional] if this flag is set to > 1, horizon will parallelize reingestion using the supplied number of workers",
	},
	{
		Name:        "parallel-job-size",
		ConfigKey:   &parallelJobSize,
		OptType:     types.Uint32,
		Required:    false,
		FlagDefault: uint32(100000),
		Usage:       "[optional] parallel workers will run jobs processing ledger batches of the supplied size",
	},
	{
		Name:        "retries",
		ConfigKey:   &retries,
		OptType:     types.Uint,
		Required:    false,
		FlagDefault: uint(0),
		Usage:       "[optional] number of reingest retries",
	},
	{
		Name:        "retry-backoff-seconds",
		ConfigKey:   &retryBackoffSeconds,
		OptType:     types.Uint,
		Required:    false,
		FlagDefault: uint(5),
		Usage:       "[optional] backoff seconds between reingest retries",
	},
}

var dbReingestRangeCmd = &cobra.Command{
	Use:   "range [Start sequence number] [End sequence number]",
	Short: "reingests ledgers within a range",
	Long:  "reingests ledgers between X and Y sequence number (closed intervals)",
	Run: func(cmd *cobra.Command, args []string) {
		for _, co := range reingestRangeCmdOpts {
			co.Require()
			co.SetValue()
		}

		if len(args) != 2 {
			cmd.Usage()
			os.Exit(1)
		}

		argsUInt32 := make([]uint32, 2)
		for i, arg := range args {
			if seq, err := strconv.Atoi(arg); err != nil {
				cmd.Usage()
				log.Fatalf(`Invalid sequence number "%s"`, arg)
			} else if seq < 0 {
				log.Fatalf("sequence number %s cannot be negative", arg)
			} else {
				argsUInt32[i] = uint32(seq)
			}
		}

		horizon.ApplyFlags(config, flags, horizon.ApplyOptions{RequireCaptiveCoreConfig: false, AlwaysIngest: true})
		err := runDBReingestRange(argsUInt32[0], argsUInt32[1], reingestForce, parallelWorkers, *config)
		if err != nil {
			if errors.Cause(err) == ingest.ErrReingestRangeConflict {
				message := `
			The range you have provided overlaps with Horizon's most recently ingested ledger.
			It is not possible to run the reingest command on this range in parallel with
			Horizon's ingestion system.
			Either reduce the range so that it doesn't overlap with Horizon's ingestion system,
			or, use the force flag to ensure that Horizon's ingestion system is blocked until
			the reingest command completes.
			`
				log.Fatal(message)
			}

			log.Fatal(err)
		}

		hlog.Info("Range run successfully!")
	},
}

func runDBReingestRange(from, to uint32, reingestForce bool, parallelWorkers uint, config horizon.Config) error {
	if reingestForce && parallelWorkers > 1 {
		return errors.New("--force is incompatible with --parallel-workers > 1")
	}
	horizonSession, err := db.Open("postgres", config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("cannot open Horizon DB: %v", err)
	}

	ingestConfig := ingest.Config{
		NetworkPassphrase:           config.NetworkPassphrase,
		HistorySession:              horizonSession,
		HistoryArchiveURL:           config.HistoryArchiveURLs[0],
		CheckpointFrequency:         config.CheckpointFrequency,
		MaxReingestRetries:          int(retries),
		ReingestRetryBackoffSeconds: int(retryBackoffSeconds),
		EnableCaptiveCore:           config.EnableCaptiveCoreIngestion,
		CaptiveCoreBinaryPath:       config.CaptiveCoreBinaryPath,
		RemoteCaptiveCoreURL:        config.RemoteCaptiveCoreURL,
		CaptiveCoreToml:             config.CaptiveCoreToml,
		CaptiveCoreStoragePath:      config.CaptiveCoreStoragePath,
		StellarCoreCursor:           config.CursorName,
		StellarCoreURL:              config.StellarCoreURL,
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

	if parallelWorkers > 1 {
		system, systemErr := ingest.NewParallelSystems(ingestConfig, parallelWorkers)
		if systemErr != nil {
			return systemErr
		}

		return system.ReingestRange(
			from,
			to,
			parallelJobSize,
		)
	}

	system, systemErr := ingest.NewSystem(ingestConfig)
	if systemErr != nil {
		return systemErr
	}

	return system.ReingestRange(
		from,
		to,
		reingestForce,
	)
}

var dbDetectGapsCmd = &cobra.Command{
	Use:   "detect-gaps",
	Short: "detects ingestion gaps in Horizon's database",
	Long:  "detects ingestion gaps in Horizon's database and prints a list of reingest commands needed to fill the gaps",
	Run: func(cmd *cobra.Command, args []string) {
		requireAndSetFlag(horizon.DatabaseURLFlagName)
		if len(args) != 0 {
			cmd.Usage()
			os.Exit(1)
		}
		gaps, err := runDBDetectGaps(*config)
		if err != nil {
			log.Fatal(err)
		}
		if len(gaps) == 0 {
			hlog.Info("No gaps found")
			return
		}
		fmt.Println("Horizon commands to run in order to fill in the gaps:")
		cmdname := os.Args[0]
		for _, g := range gaps {
			fmt.Printf("%s db reingest %d %d\n", cmdname, g.StartSequence, g.EndSequence)
		}
	},
}

func runDBDetectGaps(config horizon.Config) ([]history.LedgerGap, error) {
	horizonSession, err := db.Open("postgres", config.DatabaseURL)
	if err != nil {
		return nil, err
	}
	q := &history.Q{horizonSession}
	return q.GetLedgerGaps(context.Background())
}

func init() {
	for _, co := range reingestRangeCmdOpts {
		err := co.Init(dbReingestRangeCmd)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	viper.BindPFlags(dbReingestRangeCmd.PersistentFlags())

	RootCmd.AddCommand(dbCmd)
	dbCmd.AddCommand(
		dbInitCmd,
		dbMigrateCmd,
		dbReapCmd,
		dbReingestCmd,
		dbDetectGapsCmd,
	)
	dbMigrateCmd.AddCommand(
		dbMigrateDownCmd,
		dbMigrateRedoCmd,
		dbMigrateStatusCmd,
		dbMigrateUpCmd,
	)
	dbReingestCmd.AddCommand(dbReingestRangeCmd)
}
