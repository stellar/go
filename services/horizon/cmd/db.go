package cmd

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"go/types"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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

var dbMigrateCmd = &cobra.Command{
	Use:   "migrate [up|down|redo] [COUNT]",
	Short: "migrate schema",
	Long:  "performs a schema migration command",
	Run: func(cmd *cobra.Command, args []string) {
		requireAndSetFlag(horizon.DatabaseURLFlagName)

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
	},
}

var dbMigrateHashCmd = &cobra.Command{
	Use:   "migrate-hash",
	Short: "outputs hash for each migration file",
	Long:  "migrate-hash hashes each migration file of the database and outputs it",
	Run: func(cmd *cobra.Command, args []string) {
		migrationFolder := "./services/horizon/internal/db2/schema/migrations/"
		hasher := sha256.New()
		var files []string

		err := filepath.Walk(migrationFolder, func(path string, info os.FileInfo, err error) error {
			files = append(files, path)
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}

		// This is an alphabetical sorting of numbers, but this is fine since it is still deterministic.
		files = sort.StringSlice(files)

		for _, path := range files {
			extension := filepath.Ext(path)
			if extension == ".sql" {
				data, err := ioutil.ReadFile(path)
				if err != nil {
					log.Fatal(err)
				}
				hasher.Write([]byte(data))
				hashed := hex.EncodeToString(hasher.Sum(nil))
				fmt.Println(path + ": " + hashed)
			}
		}
	},
}

var dbReapCmd = &cobra.Command{
	Use:   "reap",
	Short: "reaps (i.e. removes) any reapable history data",
	Long:  "reap removes any historical data that is earlier than the configured retention cutoff",
	Run: func(cmd *cobra.Command, args []string) {
		app := horizon.NewAppFromFlags(config, flags)
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

		if len(args) != 2 {
			cmd.Usage()
			os.Exit(1)
		}

		argsUInt32 := make([]uint32, 2)
		for i, arg := range args {
			seq, err := strconv.Atoi(arg)
			if err != nil {
				cmd.Usage()
				log.Fatalf(`Invalid sequence number "%s"`, arg)
			}
			argsUInt32[i] = uint32(seq)
		}

		horizon.ApplyFlags(config, flags)
		err := RunDBReingestRange(argsUInt32[0], argsUInt32[1], reingestForce, parallelWorkers, *config)
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

func RunDBReingestRange(from, to uint32, reingestForce bool, parallelWorkers uint, config horizon.Config) error {
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
		CaptiveCoreConfigAppendPath: config.CaptiveCoreConfigAppendPath,
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
		dbMigrateHashCmd,
		dbReapCmd,
		dbReingestCmd,
	)
	dbReingestCmd.AddCommand(dbReingestRangeCmd)
}
