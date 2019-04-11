package cmd

import (
	"context"

	"github.com/lib/pq"
	"github.com/spf13/cobra"
	ticker "github.com/stellar/go/exp/ticker/internal"
	"github.com/stellar/go/exp/ticker/internal/tickerdb"
)

var ShouldStream bool
var BackfillDays int

func init() {
	rootCmd.AddCommand(cmdIngest)
	cmdIngest.AddCommand(cmdIngestAssets)
	cmdIngest.AddCommand(cmdIngestTrades)

	cmdIngestTrades.Flags().BoolVar(
		&ShouldStream,
		"stream",
		false,
		"Continuously stream new trades from the Horizon Stream API as a daemon",
	)

	cmdIngestTrades.Flags().IntVar(
		&BackfillDays,
		"num-days",
		7,
		"Number of past days to backfill trade data",
	)
}

var cmdIngest = &cobra.Command{
	Use:   "ingest [data type]",
	Short: "Ingests new data from data type into the database.",
}

var cmdIngestAssets = &cobra.Command{
	Use:   "assets",
	Short: "Refreshes the asset database with new data retrieved from Horizon.",
	Run: func(cmd *cobra.Command, args []string) {
		Logger.Warn("Refreshing the asset database")
		dbInfo, err := pq.ParseURL(DatabaseURL)
		if err != nil {
			Logger.Fatal("could not parse db-url:", err)
		}

		session, err := tickerdb.CreateSession("postgres", dbInfo)
		if err != nil {
			Logger.Fatal("could not connect to db:", err)
		}
		defer session.DB.Close()

		err = ticker.RefreshAssets(&session, Client, Logger)
		if err != nil {
			Logger.Fatal("could not refresh error database:", err)
		}
	},
}

var cmdIngestTrades = &cobra.Command{
	Use:   "trades",
	Short: "Fills the trade database with data retrieved form Horizon.",
	Run: func(cmd *cobra.Command, args []string) {
		dbInfo, err := pq.ParseURL(DatabaseURL)
		if err != nil {
			Logger.Fatal("could not parse db-url:", err)
		}

		session, err := tickerdb.CreateSession("postgres", dbInfo)
		if err != nil {
			Logger.Fatal("could not connect to db:", err)
		}
		defer session.DB.Close()

		Logger.Warnf("Backfilling Trade data for the past %d day(s)\n", BackfillDays)
		err = ticker.BackfillTrades(&session, Client, Logger, BackfillDays, 0)
		if err != nil {
			Logger.Fatal("could not refresh error database:", err)
		}

		if ShouldStream {
			Logger.Warn("Streaming new data (this is a continuous process)")
			ctx := context.Background()
			err = ticker.StreamTrades(ctx, &session, Client, Logger)
			if err != nil {
				Logger.Fatal("could not refresh error database:", err)
			}
		}
	},
}
