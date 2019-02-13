package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stellar/go/services/horizon/internal/db2/schema"
	"github.com/stellar/go/services/horizon/internal/ingest"
	"github.com/stellar/go/support/db"
	hlog "github.com/stellar/go/support/log"
)

var dbCmd = &cobra.Command{
	Use:   "db [command]",
	Short: "commands to manage horizon's postgres db",
}

var dbBackfillCmd = &cobra.Command{
	Use:   "backfill [COUNT]",
	Short: "backfills horizon history for COUNT ledgers",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Println("Missing COUNT. Usage: backfill [COUNT].")
			return
		}

		initApp().UpdateLedgerState()

		i := ingestSystem(ingest.Config{})
		i.SkipCursorUpdate = true
		parsed, err := strconv.ParseUint(args[0], 10, 32)
		if err != nil {
			log.Fatal(err)
		}

		err = i.Backfill(uint(parsed))
		if err != nil {
			log.Fatal(err)
		}
	},
}

var dbInitAssetStatsCmd = &cobra.Command{
	Use:   "init-asset-stats",
	Short: "initializes values for assets stats",
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()

		hdb, err := db.Open("postgres", config.DatabaseURL)
		if err != nil {
			log.Fatal(err)
		}

		cdb, err := db.Open("postgres", config.StellarCoreDatabaseURL)
		if err != nil {
			log.Fatal(err)
		}

		assetStats := ingest.AssetStats{
			CoreSession:    cdb,
			HistorySession: hdb,
		}

		log.Println("Getting assets from core DB...")

		count, err := assetStats.AddAllAssetsFromCore()
		if err != nil {
			log.Fatal(err)
		}

		log.Println(fmt.Sprintf("Updating %d assets...", count))

		err = assetStats.UpdateAssetStats()
		if err != nil {
			log.Fatal(err)
		}

		log.Println(fmt.Sprintf("Added stats for %d assets...", count))
	},
}

var dbClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "clears all imported historical data",
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()

		err := ingestSystem(ingest.Config{}).ClearAll()
		if err != nil {
			log.Fatal(err)
		}
	},
}

var dbInitCmd = &cobra.Command{
	Use:   "init",
	Short: "install schema",
	Long:  "init initializes the postgres database used by horizon.",
	Run: func(cmd *cobra.Command, args []string) {
		dbConn, err := db.Open("postgres", viper.GetString("db-url"))
		if err != nil {
			log.Fatal(err)
		}

		err = schema.Init(dbConn)
		if err != nil {
			log.Fatal(err)
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

		db, err := sql.Open("postgres", viper.GetString("db-url"))
		if err != nil {
			log.Fatal(err)
		}

		_, err = schema.Migrate(db, dir, count)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var dbReapCmd = &cobra.Command{
	Use:   "reap",
	Short: "reaps (i.e. removes) any reapable history data",
	Long:  "reap removes any historical data that is earlier than the configured retention cutoff",
	Run: func(cmd *cobra.Command, args []string) {
		err := initApp().DeleteUnretainedHistory()
		if err != nil {
			log.Fatal(err)
		}
	},
}

var dbRebaseCmd = &cobra.Command{
	Use:   "rebase",
	Short: "rebases clears the horizon db and ingests the latest ledger segment from stellar-core",
	Long:  "...",
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()

		i := ingestSystem(ingest.Config{})
		i.SkipCursorUpdate = true

		err := i.RebaseHistory()
		if err != nil {
			log.Fatal(err)
		}
	},
}

var dbReingestCmd = &cobra.Command{
	Use:   "reingest",
	Short: "imports all data",
	Long:  "reingest runs the ingestion pipeline over every ledger",
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()

		i := ingestSystem(ingest.Config{})
		i.SkipCursorUpdate = true
		logStatus := func(stage string) {
			count := i.Metrics.IngestLedgerTimer.Count()
			rate := i.Metrics.IngestLedgerTimer.RateMean()
			loadMean := time.Duration(i.Metrics.LoadLedgerTimer.Mean())
			ingestMean := time.Duration(i.Metrics.IngestLedgerTimer.Mean())
			clearMean := time.Duration(i.Metrics.IngestLedgerTimer.Mean())
			hlog.WithField("count", count).
				WithField("rate", rate).
				WithField("means", fmt.Sprintf("load: %s clear: %s ingest: %s", loadMean, clearMean, ingestMean)).
				Infof("reingest: %s", stage)
		}

		done := make(chan error, 1)

		// run ingestion in separate goroutine
		go func() {
			_, err := reingest(i, args)
			done <- err
			logStatus("complete")
		}()

		// output metrics
		metrics := time.Tick(2 * time.Second)
		for {
			select {
			case <-metrics:
				logStatus("status")

			case err := <-done:
				if err != nil {
					log.Fatal(err)
				}
				os.Exit(0)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(dbCmd)
	dbCmd.AddCommand(
		dbInitCmd,
		dbInitAssetStatsCmd,
		dbBackfillCmd,
		dbClearCmd,
		dbMigrateCmd,
		dbReapCmd,
		dbReingestCmd,
		dbRebaseCmd,
	)
}

func ingestSystem(ingestConfig ingest.Config) *ingest.System {
	hdb, err := db.Open("postgres", config.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	cdb, err := db.Open("postgres", config.StellarCoreDatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	passphrase := viper.GetString("network-passphrase")
	if passphrase == "" {
		log.Fatal("network-passphrase is blank: reingestion requires manually setting passphrase")
	}

	return ingest.New(passphrase, config.StellarCoreURL, cdb, hdb, ingestConfig)
}

func reingest(i *ingest.System, args []string) (int, error) {
	if len(args) == 0 {
		return i.ReingestAll()
	}

	if len(args) == 1 && args[0] == "outdated" {
		return i.ReingestOutdated()
	}

	for idx, arg := range args {
		seq, err := strconv.Atoi(arg)
		if err != nil {
			return idx, err
		}

		err = i.ReingestSingle(int32(seq))
		if err != nil {
			return idx, err
		}
	}
	return len(args), nil
}
