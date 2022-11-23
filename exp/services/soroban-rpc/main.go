package main

import (
	"fmt"
	"go/types"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest/ledgerbackend"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/clients/stellarcore"
	"github.com/stellar/go/exp/services/soroban-rpc/internal"
	"github.com/stellar/go/exp/services/soroban-rpc/internal/methods"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/config"
	supporthttp "github.com/stellar/go/support/http"
	supportlog "github.com/stellar/go/support/log"
)

func main() {
	var endpoint, horizonURL, binaryPath, configPath, networkPassphrase string
	var captiveCoreTomlParams ledgerbackend.CaptiveCoreTomlParams
	var historyArchiveURLs []string
	var checkpointFrequency uint32
	var txConcurrency, txQueueSize int
	var logLevel logrus.Level
	logger := supportlog.New()

	configOpts := config.ConfigOptions{
		{
			Name:        "endpoint",
			Usage:       "Endpoint to listen and serve on",
			OptType:     types.String,
			ConfigKey:   &endpoint,
			FlagDefault: "localhost:8000",
			Required:    false,
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
			Name:           "stellar-captive-core-http-port",
			ConfigKey:      &captiveCoreTomlParams.HTTPPort,
			OptType:        types.Uint,
			CustomSetValue: config.SetOptionalUint,
			Required:       false,
			FlagDefault:    uint(11626),
			Usage:          "HTTP port for Captive Core to listen on (0 disables the HTTP server)",
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
		&config.ConfigOption{
			Name:        "stellar-core-binary-path",
			OptType:     types.String,
			FlagDefault: "",
			Required:    true,
			Usage:       "path to stellar core binary",
			ConfigKey:   &binaryPath,
		},
		&config.ConfigOption{
			Name:        "captive-core-config-path",
			OptType:     types.String,
			FlagDefault: "",
			Required:    true,
			Usage:       "path to additional configuration for the Stellar Core configuration file used by captive core. It must, at least, include enough details to define a quorum set",
			ConfigKey:   &configPath,
		},
		&config.ConfigOption{
			Name:        "history-archive-urls",
			ConfigKey:   &historyArchiveURLs,
			OptType:     types.String,
			Required:    true,
			FlagDefault: "",
			CustomSetValue: func(co *config.ConfigOption) error {
				stringOfUrls := viper.GetString(co.Name)
				urlStrings := strings.Split(stringOfUrls, ",")

				*(co.ConfigKey.(*[]string)) = urlStrings
				return nil
			},
			Usage: "comma-separated list of stellar history archives to connect with",
		},
		{
			Name:        "network-passphrase",
			Usage:       "Network passphrase of the Stellar network transactions should be signed for",
			OptType:     types.String,
			ConfigKey:   &networkPassphrase,
			FlagDefault: network.FutureNetworkPassphrase,
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
				5*time.Minute,
			)

			captiveCoreTomlParams.HistoryArchiveURLs = historyArchiveURLs
			captiveCoreTomlParams.NetworkPassphrase = networkPassphrase
			captiveCoreTomlParams.Strict = true
			captiveCoreToml, err := ledgerbackend.NewCaptiveCoreTomlFromFile(configPath, captiveCoreTomlParams)
			if err != nil {
				logger.WithError(err).Fatal("Invalid captive core toml")
			}

			captiveConfig := ledgerbackend.CaptiveCoreConfig{
				BinaryPath:          binaryPath,
				NetworkPassphrase:   networkPassphrase,
				HistoryArchiveURLs:  historyArchiveURLs,
				CheckpointFrequency: checkpointFrequency,
				Log:                 logger.WithField("subservice", "stellar-core"),
				Toml:                captiveCoreToml,
				UserAgent:           "captivecore",
			}
			core, err := ledgerbackend.NewCaptive(captiveConfig)
			if err != nil {
				logger.Fatalf("could not create captive core: %v", err)
			}

			defer core.Close()

			historyArchive, err := historyarchive.Connect(
				historyArchiveURLs[0],
				historyarchive.ConnectOptions{},
			)

			storage, err := internal.NewLedgerEntryStorage(networkPassphrase, historyArchive, core)
			defer storage.Close()

			handler, err := internal.NewJSONRPCHandler(internal.HandlerParams{
				AccountStore:     methods.AccountStore{Client: hc},
				Logger:           logger,
				TransactionProxy: transactionProxy,
				CoreClient:       &stellarcore.Client{URL: fmt.Sprintf("http://localhost:%d/", captiveCoreTomlParams.HTTPPort)},
			})
			if err != nil {
				logger.Fatalf("could not create handler: %v", err)
			}
			supporthttp.Run(supporthttp.Config{
				ListenAddr: endpoint,
				Handler:    handler,
				OnStarting: func() {
					logger.Infof("Starting Soroban JSON RPC server on %v", endpoint)
					handler.Start()
				},
				OnStopping: func() {
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
