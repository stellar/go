package main

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/stellar/go/exp/lighthorizon/actions"
	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/exp/lighthorizon/services"
	"github.com/stellar/go/exp/lighthorizon/tools"

	"github.com/stellar/go/network"
	"github.com/stellar/go/support/log"
)

const (
	HorizonLiteVersion = "0.0.1-alpha"
	defaultCacheSize   = (60 * 60 * 24) / 6 // 1 day of ledgers @ 6s each
)

func main() {
	log.SetLevel(logrus.InfoLevel) // default for subcommands

	cmd := &cobra.Command{
		Use:  "lighthorizon <subcommand>",
		Long: "Horizon Lite command suite",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Usage() // require a subcommand
		},
	}

	serve := &cobra.Command{
		Use: "serve <txmeta source> <index source>",
		Long: `Starts the Horizon Lite server, binding it to port 8080 on all 
local interfaces of the host. You can refer to the OpenAPI documentation located
at the /api endpoint to see what endpoints are supported.

The <txmeta source> should be a URL to meta archives from which to read unpacked
ledger files, while the <index source> should be a URL containing indices that
break down accounts by active ledgers.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				cmd.Usage()
				return
			}

			sourceUrl, indexStoreUrl := args[0], args[1]

			networkPassphrase, _ := cmd.Flags().GetString("network-passphrase")
			switch networkPassphrase {
			case "testnet":
				networkPassphrase = network.TestNetworkPassphrase
			case "pubnet":
				networkPassphrase = network.PublicNetworkPassphrase
			}

			cacheDir, _ := cmd.Flags().GetString("ledger-cache")
			cacheSize, _ := cmd.Flags().GetUint("ledger-cache-size")
			logLevelParam, _ := cmd.Flags().GetString("log-level")

			L := log.WithField("service", "horizon-lite")
			logLevel, err := logrus.ParseLevel(logLevelParam)
			if err != nil {
				log.Warnf("Failed to parse log level '%s', defaulting to 'info'.", logLevelParam)
				logLevel = log.InfoLevel
			}
			L.SetLevel(logLevel)
			L.Info("Starting lighthorizon!")

			registry := prometheus.NewRegistry()
			indexStore, err := index.ConnectWithConfig(index.StoreConfig{
				URL:     indexStoreUrl,
				Log:     L.WithField("service", "index"),
				Metrics: registry,
			})
			if err != nil {
				log.Fatal(err)
				return
			}

			ingester, err := archive.NewIngestArchive(archive.ArchiveConfig{
				SourceUrl:         sourceUrl,
				NetworkPassphrase: networkPassphrase,
				CacheDir:          cacheDir,
				CacheSize:         int(cacheSize),
			})
			if err != nil {
				log.Fatal(err)
				return
			}

			Config := services.Config{
				Archive:    ingester,
				Passphrase: networkPassphrase,
				IndexStore: indexStore,
				Metrics:    services.NewMetrics(registry),
			}

			lightHorizon := services.LightHorizon{
				Transactions: &services.TransactionRepository{
					Config: Config,
				},
				Operations: &services.OperationRepository{
					Config: Config,
				},
			}

			// Inject our config into the root response.
			router := lightHorizonHTTPHandler(registry, lightHorizon).(*chi.Mux)
			router.MethodFunc(http.MethodGet, "/", actions.Root(actions.RootResponse{
				Version:      HorizonLiteVersion,
				LedgerSource: sourceUrl,
				IndexSource:  indexStoreUrl,
			}))

			log.Fatal(http.ListenAndServe(":8080", router))
		},
	}

	serve.Flags().String("log-level", "info",
		"logging level: 'info', 'debug', 'warn', 'error', 'panic', 'fatal', or 'trace'")
	serve.Flags().String("network-passphrase", "pubnet", "network passphrase")
	serve.Flags().String("ledger-cache", "", "path to cache frequently-used ledgers; "+
		"if left empty, uses a temporary directory")
	serve.Flags().Uint("ledger-cache-size", defaultCacheSize,
		"number of ledgers to store in the cache")

	cmd.AddCommand(serve)
	tools.AddCacheCommands(cmd)
	tools.AddIndexCommands(cmd)
	cmd.Execute()
}
