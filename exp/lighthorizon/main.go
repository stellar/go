package main

import (
	"context"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/stellar/go/exp/lighthorizon/actions"
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/exp/lighthorizon/ingester"
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
			downloadCount, _ := cmd.Flags().GetUint("parallel-downloads")

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

			ingester, err := ingester.NewIngester(ingester.IngesterConfig{
				SourceUrl:         sourceUrl,
				NetworkPassphrase: networkPassphrase,
				CacheDir:          cacheDir,
				CacheSize:         int(cacheSize),
				ParallelDownloads: downloadCount,
			})
			if err != nil {
				log.Fatal(err)
				return
			}

			latestLedger, err := ingester.GetLatestLedgerSequence(context.Background())
			if err != nil {
				log.Fatalf("Failed to retrieve latest ledger from %s: %v", sourceUrl, err)
				return
			}
			log.Infof("The latest ledger stored at %s is %d.", sourceUrl, latestLedger)

			cachePreloadCount, _ := cmd.Flags().GetUint32("ledger-cache-preload")
			cachePreloadStart, _ := cmd.Flags().GetUint32("ledger-cache-preload-start")
			if cachePreloadCount > 0 {
				if cacheDir == "" {
					log.Fatalf("--ledger-cache-preload=%d specified but no "+
						"--ledger-cache directory provided.",
						cachePreloadCount)
					return
				} else {
					startLedger := int(latestLedger) - int(cachePreloadCount)
					if cachePreloadStart > 0 {
						startLedger = int(cachePreloadStart)
					}
					if startLedger <= 0 {
						log.Warnf("Starting ledger invalid (%d), defaulting to 2.",
							startLedger)
						startLedger = 2
					}

					log.Infof("Preloading cache at %s with %d ledgers, starting at ledger %d.",
						cacheDir, startLedger, cachePreloadCount)
					go func() {
						tools.BuildCache(sourceUrl, cacheDir,
							uint32(startLedger), cachePreloadCount, false)
					}()
				}
			}

			Config := services.Config{
				Ingester:   ingester,
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

				LatestLedger: latestLedger,
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
	serve.Flags().Uint32("ledger-cache-preload", 0,
		"should the cache come preloaded with the latest <n> ledgers?")
	serve.Flags().Uint32("ledger-cache-preload-start", 0,
		"the preload should start at ledger <n>")
	serve.Flags().Uint("parallel-downloads", 1,
		"how many workers should download ledgers in parallel?")

	cmd.AddCommand(serve)
	tools.AddCacheCommands(cmd)
	tools.AddIndexCommands(cmd)
	cmd.Execute()
}
