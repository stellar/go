package cmd

import (
	"github.com/lib/pq"
	"github.com/spf13/cobra"
	ticker "github.com/stellar/go/exp/ticker/internal"
	"github.com/stellar/go/exp/ticker/internal/tickerdb"
)

var OutFile string

func init() {
	rootCmd.AddCommand(cmdGenerate)
	cmdGenerate.AddCommand(cmdGenerateMarketData)

	cmdGenerateMarketData.Flags().StringVarP(
		&OutFile,
		"out-file",
		"o",
		"markets.json",
		"Set the name of the output file",
	)
}

var cmdGenerate = &cobra.Command{
	Use:   "generate [data type]",
	Short: "Generates reports about assets and markets",
}

var cmdGenerateMarketData = &cobra.Command{
	Use:   "market-data",
	Short: "Generate the aggregated market data (for 24h and 7d) and outputs to a file.",
	Run: func(cmd *cobra.Command, args []string) {
		dbInfo, err := pq.ParseURL(DatabaseURL)
		if err != nil {
			Logger.Fatal("could not parse db-url:", err)
		}

		session, err := tickerdb.CreateSession("postgres", dbInfo)
		if err != nil {
			Logger.Fatal("could not connect to db:", err)
		}

		Logger.Infof("Starting market data generation, outputting to: %s\n", OutFile)
		err = ticker.GenerateMarketSummaryFile(&session, Logger, OutFile)
		if err != nil {
			Logger.Fatal("could not generate market data:", err)
		}
	},
}
