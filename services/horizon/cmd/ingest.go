package cmd

import (
	"fmt"
	"go/types"
	"net/http"
	_ "net/http/pprof"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/services/horizon/internal/expingest"
	support "github.com/stellar/go/support/config"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/historyarchive"
	"github.com/stellar/go/support/log"
)

var ingestCmd = &cobra.Command{
	Use:   "expingest",
	Short: "ingestion related commands",
}

var ingestVerifyFrom, ingestVerifyTo, ingestVerifyDebugServerPort uint32
var ingestVerifyState bool

var ingestVerifyRangeCmdOpts = []*support.ConfigOption{
	&support.ConfigOption{
		Name:        "from",
		ConfigKey:   &ingestVerifyFrom,
		OptType:     types.Uint32,
		Required:    true,
		FlagDefault: uint32(0),
		Usage:       "first ledger of the range to ingest",
	},
	&support.ConfigOption{
		Name:        "to",
		ConfigKey:   &ingestVerifyTo,
		OptType:     types.Uint32,
		Required:    true,
		FlagDefault: uint32(0),
		Usage:       "last ledger of the range to ingest",
	},
	&support.ConfigOption{
		Name:        "verify-state",
		ConfigKey:   &ingestVerifyState,
		OptType:     types.Bool,
		Required:    false,
		FlagDefault: false,
		Usage:       "[optional] verifies state at the last ledger of the range when true",
	},
	&support.ConfigOption{
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
	Short: "runs ingestion pipeline within a range",
	Long:  "runs ingestion pipeline between X and Y sequence number (inclusive)",
	Run: func(cmd *cobra.Command, args []string) {
		for _, co := range ingestVerifyRangeCmdOpts {
			co.Require()
			co.SetValue()
		}

		initRootConfig()

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

		coreSession, err := db.Open("postgres", config.StellarCoreDatabaseURL)
		if err != nil {
			log.Fatalf("cannot open Core DB: %v", err)
		}

		horizonSession, err := db.Open("postgres", config.DatabaseURL)
		if err != nil {
			log.Fatalf("cannot open Horizon DB: %v", err)
		}

		if !historyarchive.IsCheckpoint(ingestVerifyFrom) {
			log.Fatal("`--from` must be a checkpoint ledger")
		}

		if ingestVerifyState && !historyarchive.IsCheckpoint(ingestVerifyTo) {
			log.Fatal("`--to` must be a checkpoint ledger when `--verify-state` is set.")
		}

		ingestConfig := expingest.Config{
			CoreSession:       coreSession,
			HistorySession:    horizonSession,
			HistoryArchiveURL: config.HistoryArchiveURLs[0],
			OrderBookGraph:    orderbook.NewOrderBookGraph(),
		}

		err = expingest.VerifyPipelineRange(
			ingestVerifyFrom,
			ingestVerifyTo,
			ingestConfig,
			ingestVerifyState,
		)
		if err != nil {
			log.Fatal(err)
		}

		log.Info("Range run successfully!")
	},
}

func init() {
	for _, co := range ingestVerifyRangeCmdOpts {
		err := co.Init(ingestVerifyRangeCmd)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	viper.BindPFlags(ingestVerifyRangeCmd.PersistentFlags())

	rootCmd.AddCommand(ingestCmd)
	ingestCmd.AddCommand(ingestVerifyRangeCmd)
}
