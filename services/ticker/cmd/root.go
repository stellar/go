package cmd

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	horizonclient "github.com/stellar/go/clients/horizonclient"
	hlog "github.com/stellar/go/support/log"
)

var DatabaseURL string
var Client *horizonclient.Client
var UseTestNet bool
var Logger = hlog.New()

var defaultDatabaseURL = getEnv("DB_URL", "postgres://localhost:5432/stellarticker01?sslmode=disable")

var rootCmd = &cobra.Command{
	Use:   "ticker",
	Short: "Stellar Development Foundation Ticker.",
	Long:  `A tool to provide Stellar Asset and Market data.`,
}

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(
		&DatabaseURL,
		"db-url",
		"d",
		defaultDatabaseURL,
		"database URL, such as: postgres://user:pass@localhost:5432/ticker",
	)
	rootCmd.PersistentFlags().BoolVar(
		&UseTestNet,
		"testnet",
		false,
		"use the Stellar Test Network, instead of the Stellar Public Network",
	)

	Logger.SetLevel(logrus.DebugLevel)
}

func initConfig() {
	if UseTestNet {
		Logger.Debug("Using Stellar Default Test Network")
		Client = horizonclient.DefaultTestNetClient
	} else {
		Logger.Debug("Using Stellar Default Public Network")
		Client = horizonclient.DefaultPublicNetClient
	}
}

// Execute executes the root command and panics upon error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
