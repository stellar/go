package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/facebookgo/inject"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/services/bifrost/config"
	"github.com/stellar/go/services/bifrost/database"
	"github.com/stellar/go/services/bifrost/ethereum"
	"github.com/stellar/go/services/bifrost/server"
	"github.com/stellar/go/services/bifrost/stellar"
	supportConfig "github.com/stellar/go/support/config"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

var rootCmd = &cobra.Command{
	Use:   "bifrost",
	Short: "Bridge server to allow participating in Stellar based ICOs using Ethereum",
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Starts backend server",
	Run: func(cmd *cobra.Command, args []string) {
		var (
			cfg     config.Config
			cfgPath = cmd.PersistentFlags().Lookup("config").Value.String()
		)

		err := supportConfig.Read(cfgPath, &cfg)
		if err != nil {
			switch cause := errors.Cause(err).(type) {
			case *supportConfig.InvalidConfigError:
				log.Error("config file: ", cause)
			default:
				log.Error(err)
			}
			os.Exit(-1)
		}

		db := &database.PostgresDatabase{}
		err = db.Open(cfg.Database.DSN)
		if err != nil {
			log.WithField("err", err).Error("Error connecting to database")
			os.Exit(-1)
		}

		ethereumListener := &ethereum.Listener{}

		stellarAccountConfigurator := &stellar.AccountConfigurator{
			NetworkPassphrase: cfg.Stellar.NetworkPassphrase,
			IssuerSecretKey:   cfg.Stellar.IssuerSecretKey,
		}

		horizonClient := &horizon.Client{
			URL: cfg.Stellar.Horizon,
			HTTP: &http.Client{
				Timeout: 10 * time.Second,
			},
		}

		addressGenerator, err := ethereum.NewAddressGenerator(cfg.Ethereum.MasterPublicKey)
		if err != nil {
			log.Error(err)
			os.Exit(-1)
		}

		server := &server.Server{}

		var g inject.Graph
		err = g.Provide(
			&inject.Object{Value: addressGenerator},
			&inject.Object{Value: &cfg},
			&inject.Object{Value: db},
			&inject.Object{Value: ethereumListener},
			&inject.Object{Value: horizonClient},
			&inject.Object{Value: server},
			&inject.Object{Value: stellarAccountConfigurator},
		)
		if err != nil {
			log.WithField("err", err).Error("Error providing objects to injector")
			os.Exit(-1)
		}

		if err := g.Populate(); err != nil {
			log.WithField("err", err).Error("Error injecting objects")
			os.Exit(-1)
		}

		err = server.Start()
		if err != nil {
			log.WithField("err", err).Error("Error starting the server")
			os.Exit(-1)
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("version")
	},
}

func init() {
	log.SetLevel(log.InfoLevel)
	log.DefaultLogger.Logger.Formatter.(*logrus.TextFormatter).FullTimestamp = true

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(serverCmd)

	serverCmd.PersistentFlags().StringP("config", "c", "bifrost.cfg", "config file path")
}

func main() {
	rootCmd.Execute()
}
