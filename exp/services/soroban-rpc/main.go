package main

import (
	"context"
	"fmt"
	"github.com/stellar/go/network"
	"go/types"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/exp/services/soroban-rpc/internal"
	"github.com/stellar/go/exp/services/soroban-rpc/internal/methods"
	"github.com/stellar/go/support/config"
	supporthttp "github.com/stellar/go/support/http"
	supportlog "github.com/stellar/go/support/log"
)

func main() {
	var port int
	var horizonURL, networkPassphrase string
	var txConcurrency, txQueueSize int
	var logLevel logrus.Level
	logger := supportlog.New()

	configOpts := config.ConfigOptions{
		{
			Name:        "port",
			Usage:       "Port to listen and serve on",
			OptType:     types.Int,
			ConfigKey:   &port,
			FlagDefault: 8000,
			Required:    true,
		},
		&config.ConfigOption{
			Name:        "horizon-url",
			ConfigKey:   &horizonURL,
			OptType:     types.String,
			Required:    true,
			FlagDefault: "",
			Usage:       "URL used to query Horizon",
		},
		&config.ConfigOption{
			Name:        "log-level",
			ConfigKey:   &logLevel,
			OptType:     types.String,
			FlagDefault: "info",
			CustomSetValue: func(co *config.ConfigOption) error {
				ll, err := logrus.ParseLevel(viper.GetString(co.Name))
				if err != nil {
					return fmt.Errorf("Could not parse log-level: %v", viper.GetString(co.Name))
				}
				*(co.ConfigKey.(*logrus.Level)) = ll
				return nil
			},
			Usage: "minimum log severity (debug, info, warn, error) to log",
		},
		{
			Name:        "network-passphrase",
			Usage:       "Network passphrase of the Stellar network transactions should be signed for",
			OptType:     types.String,
			ConfigKey:   &networkPassphrase,
			FlagDefault: network.TestNetworkPassphrase,
			Required:    true,
		},
		{
			Name:        "tx-concurrency",
			Usage:       "Maximum number of concurrent transaction submissions",
			OptType:     types.Int,
			ConfigKey:   &txConcurrency,
			FlagDefault: 10,
			Required:    false,
		},
		{
			Name:        "tx-queue",
			Usage:       "Maximum length of pending transactions queue",
			OptType:     types.Int,
			ConfigKey:   &txQueueSize,
			FlagDefault: 10,
			Required:    false,
		},
	}
	cmd := &cobra.Command{
		Use:   "soroban-rpc",
		Short: "Run the remote soroban-rpc server",
		Run: func(_ *cobra.Command, _ []string) {
			configOpts.Require()
			configOpts.SetValues()
			logger.SetLevel(logLevel)

			hc := &horizonclient.Client{
				HorizonURL: horizonURL,
				HTTP: &http.Client{
					Timeout: horizonclient.HorizonTimeout,
				},
				AppName: "Soroban RPC",
			}
			hc.SetHorizonTimeout(horizonclient.HorizonTimeout)

			transactionProxy := methods.NewTransactionProxy(
				hc,
				txConcurrency,
				txQueueSize,
				networkPassphrase,
				10*time.Minute,
				5*time.Minute,
			)

			handler, err := internal.NewJSONRPCHandler(internal.HandlerParams{
				AccountStore:     methods.AccountStore{Client: hc},
				Logger:           logger,
				TransactionProxy: transactionProxy,
			})
			if err != nil {
				logger.Fatalf("could not create handler: %v", err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			supporthttp.Run(supporthttp.Config{
				ListenAddr: fmt.Sprintf(":%d", port),
				Handler:    handler,
				OnStarting: func() {
					logger.Infof("Starting Soroban JSON RPC server on %v", port)
					transactionProxy.Start(ctx)
				},
				OnStopping: func() {
					cancel()
					handler.Close()
				},
			})
		},
	}

	if err := configOpts.Init(cmd); err != nil {
		logger.WithError(err).Fatal("could not parse config options")
	}

	if err := cmd.Execute(); err != nil {
		logger.WithError(err).Fatal("could not run")
	}
}
