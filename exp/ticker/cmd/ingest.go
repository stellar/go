package cmd

import (
	"github.com/lib/pq"
	"github.com/spf13/cobra"
	ticker "github.com/stellar/go/exp/ticker/internal"
	"github.com/stellar/go/exp/ticker/internal/tickerdb"
)

func init() {
	rootCmd.AddCommand(cmdIngest)
	cmdIngest.AddCommand(cmdIngestAssets)
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
