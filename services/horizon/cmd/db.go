package cmd

import (
	"database/sql"
	"fmt"
	"go/types"
	"log"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stellar/go/services/horizon/internal/db2/schema"
	"github.com/stellar/go/services/horizon/internal/expingest"
	support "github.com/stellar/go/support/config"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	hlog "github.com/stellar/go/support/log"
)

var dbCmd = &cobra.Command{
	Use:   "db [command]",
	Short: "commands to manage horizon's postgres db",
}

var dbInitCmd = &cobra.Command{
	Use:   "init",
	Short: "install schema",
	Long:  "init initializes the postgres database used by horizon.",
	Run: func(cmd *cobra.Command, args []string) {
		dbURLConfigOption.Require()
		dbURLConfigOption.SetValue()

		db, err := sql.Open("postgres", viper.GetString("db-url"))
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

var dbMigrateCmd = &cobra.Command{
	Use:   "migrate [up|down|redo] [COUNT]",
	Short: "migrate schema",
	Long:  "performs a schema migration command",
	Run: func(cmd *cobra.Command, args []string) {
		// Allow invokations with 1 or 2 args.  All other args counts are erroneous.
		if len(args) < 1 || len(args) > 2 {
			cmd.Usage()
			os.Exit(1)
		}

		dir := schema.MigrateDir(args[0])
		count := 0

		// If a second arg is present, parse it to an int and use it as the count
		// argument to the migration call.
		if len(args) == 2 {
			var err error
			count, err = strconv.Atoi(args[1])
			if err != nil {
				log.Println(err)
				cmd.Usage()
				os.Exit(1)
			}
		}

		dbURLConfigOption.Require()
		dbURLConfigOption.SetValue()

		db, err := sql.Open("postgres", viper.GetString("db-url"))
		if err != nil {
			log.Fatal(err)
		}
		pingDB(db)

		numMigrationsRun, err := schema.Migrate(db, dir, count)
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

var dbReapCmd = &cobra.Command{
	Use:   "reap",
	Short: "reaps (i.e. removes) any reapable history data",
	Long:  "reap removes any historical data that is earlier than the configured retention cutoff",
	Run: func(cmd *cobra.Command, args []string) {
		app := initApp()
		app.UpdateLedgerState()
		err := app.DeleteUnretainedHistory()
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
		if reingestForce && parallelWorkers > 1 {
			log.Fatal("--force is incompatible with --parallel-workers > 1")
		}

		if len(args) != 2 {
			cmd.Usage()
			os.Exit(1)
		}

		argsInt32 := make([]uint32, 2)
		for i, arg := range args {
			seq, err := strconv.Atoi(arg)
			if err != nil {
				cmd.Usage()
				log.Fatalf(`Invalid sequence number "%s"`, arg)
			}
			argsInt32[i] = uint32(seq)
		}

		initRootConfig()

		horizonSession, err := db.Open("postgres", config.DatabaseURL)
		if err != nil {
			log.Fatalf("cannot open Horizon DB: %v", err)
		}

		ingestConfig := expingest.Config{
			NetworkPassphrase:           config.NetworkPassphrase,
			HistorySession:              horizonSession,
			HistoryArchiveURL:           config.HistoryArchiveURLs[0],
			MaxReingestRetries:          int(retries),
			ReingestRetryBackoffSeconds: int(retryBackoffSeconds),
		}

		if config.EnableCaptiveCoreIngestion {
			ingestConfig.StellarCoreBinaryPath = config.StellarCoreBinaryPath
			ingestConfig.RemoteCaptiveCoreURL = config.RemoteCaptiveCoreURL
		} else {
			if config.StellarCoreDatabaseURL == "" {
				log.Fatalf("flag --%s cannot be empty", stellarCoreDBURLFlagName)
			}
			coreSession, dbErr := db.Open("postgres", config.StellarCoreDatabaseURL)
			if dbErr != nil {
				log.Fatalf("cannot open Core DB: %v", dbErr)
			}
			ingestConfig.CoreSession = coreSession
		}

		if parallelWorkers < 2 {
			system, systemErr := expingest.NewSystem(ingestConfig)
			if systemErr != nil {
				log.Fatal(systemErr)
			}

			err = system.ReingestRange(
				argsInt32[0],
				argsInt32[1],
				reingestForce,
			)
		} else {
			system, systemErr := expingest.NewParallelSystems(ingestConfig, parallelWorkers)
			if systemErr != nil {
				log.Fatal(systemErr)
			}

			err = system.ReingestRange(
				argsInt32[0],
				argsInt32[1],
				parallelJobSize,
			)
		}

		if err == nil {
			hlog.Info("Range run successfully!")
			return
		}

		if errors.Cause(err) == expingest.ErrReingestRangeConflict {
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
	},
}

func init() {
	for _, co := range reingestRangeCmdOpts {
		err := co.Init(dbReingestRangeCmd)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	viper.BindPFlags(dbReingestRangeCmd.PersistentFlags())

	rootCmd.AddCommand(dbCmd)
	dbCmd.AddCommand(
		dbInitCmd,
		dbMigrateCmd,
		dbReapCmd,
		dbReingestCmd,
	)
	dbReingestCmd.AddCommand(dbReingestRangeCmd)
}
