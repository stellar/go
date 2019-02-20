package cmd

import (
	"fmt"
	"go/types"
	stdLog "log"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	horizon "github.com/stellar/go/services/horizon/internal"
	"github.com/stellar/go/services/horizon/internal/db2/schema"
	apkg "github.com/stellar/go/support/app"
	support "github.com/stellar/go/support/config"
	"github.com/stellar/go/support/log"
	"github.com/throttled/throttled"
)

var config horizon.Config

var rootCmd = &cobra.Command{
	Use:   "horizon [db|serve|version]",
	Short: "client-facing api server for the stellar network",
	Long:  "client-facing api server for the stellar network. It acts as the interface between Stellar Core and applications that want to access the Stellar network. It allows you to submit transactions to the network, check the status of accounts, subscribe to event streams and more.",
	Run: func(cmd *cobra.Command, args []string) {
		initApp().Serve()
	},
}

// validateBothOrNeither ensures that both options are provided, if either is provided
func validateBothOrNeither(option1, option2 string) {
	arg1, arg2 := viper.GetString(option1), viper.GetString(option2)
	if arg1 != "" && arg2 == "" {
		stdLog.Fatalf("Invalid config: %s = %s, but corresponding option %s is not configured", option1, arg1, option2)
	}
	if arg1 == "" && arg2 != "" {
		stdLog.Fatalf("Invalid config: %s = %s, but corresponding option %s is not configured", option2, arg2, option1)
	}
}

// checkMigrations looks for necessary database migrations and fails with a descriptive error if migrations are needed
func checkMigrations() {
	migrationsToApplyUp := schema.GetMigrationsUp(viper.GetString("db-url"))
	if len(migrationsToApplyUp) > 0 {
		stdLog.Printf(`There are %v migrations to apply in the "up" direction.`, len(migrationsToApplyUp))
		stdLog.Printf("The necessary migrations are: %v", migrationsToApplyUp)
		stdLog.Printf("A database migration is required to run this version (%v) of Horizon. Run \"horizon db migrate up\" to update your DB. Consult the Changelog (https://github.com/stellar/go/blob/master/services/horizon/CHANGELOG.md) for more information.", apkg.Version())
		os.Exit(1)
	}

	nMigrationsDown := schema.GetNumMigrationsDown(viper.GetString("db-url"))
	if nMigrationsDown > 0 {
		stdLog.Printf("A database migration DOWN to an earlier version of the schema is required to run this version (%v) of Horizon. Consult the Changelog (https://github.com/stellar/go/blob/master/services/horizon/CHANGELOG.md) for more information.", apkg.Version())
		stdLog.Printf("In order to migrate the database DOWN, using the HIGHEST version number of Horizon you have installed (not this binary), run \"horizon db migrate down %v\".", nMigrationsDown)
		os.Exit(1)
	}
}

// configOpts defines the complete flag configuration for horizon
// Add a new entry here to connect a new field in the horizon.Config struct
var configOpts = []*support.ConfigOption{
	&support.ConfigOption{
		Name:      "db-url",
		EnvVar:    "DATABASE_URL",
		ConfigKey: &config.DatabaseURL,
		OptType:   types.String,
		Required:  true,
		Usage:     "horizon postgres database to connect with",
	},
	&support.ConfigOption{
		Name:      "stellar-core-db-url",
		EnvVar:    "STELLAR_CORE_DATABASE_URL",
		ConfigKey: &config.StellarCoreDatabaseURL,
		OptType:   types.String,
		Required:  true,
		Usage:     "stellar-core postgres database to connect with",
	},
	&support.ConfigOption{
		Name:      "stellar-core-url",
		ConfigKey: &config.StellarCoreURL,
		OptType:   types.String,
		Required:  true,
		Usage:     "stellar-core to connect with (for http commands)",
	},
	&support.ConfigOption{
		Name:        "port",
		ConfigKey:   &config.Port,
		OptType:     types.Uint,
		FlagDefault: uint(8000),
		Usage:       "tcp port to listen on for http requests",
	},
	&support.ConfigOption{
		Name:        "max-db-connections",
		ConfigKey:   &config.MaxDBConnections,
		OptType:     types.Int,
		FlagDefault: 20,
		Usage:       "max db connections (per DB), may need to be increased when responses are slow but DB CPU is normal",
	},
	&support.ConfigOption{
		Name:           "sse-update-frequency",
		ConfigKey:      &config.SSEUpdateFrequency,
		OptType:        types.Int,
		FlagDefault:    5,
		CustomSetValue: support.SetDuration,
		Usage:          "defines how often streams should check if there's a new ledger (in seconds), may need to increase in case of big number of streams",
	},
	&support.ConfigOption{
		Name:           "connection-timeout",
		ConfigKey:      &config.ConnectionTimeout,
		OptType:        types.Int,
		FlagDefault:    55,
		CustomSetValue: support.SetDuration,
		Usage:          "defines the timeout of connection after which 504 response will be sent or stream will be closed, if Horizon is behind a load balancer with idle connection timeout, this should be set to a few seconds less that idle timeout",
	},
	&support.ConfigOption{
		Name:        "per-hour-rate-limit",
		ConfigKey:   &config.RateLimit,
		OptType:     types.Int,
		FlagDefault: 3600,
		CustomSetValue: func(co *support.ConfigOption) {
			var rateLimit *throttled.RateQuota = nil
			perHourRateLimit := viper.GetInt(co.Name)
			if perHourRateLimit != 0 {
				rateLimit = &throttled.RateQuota{
					MaxRate:  throttled.PerHour(perHourRateLimit),
					MaxBurst: 100,
				}
				*(co.ConfigKey.(**throttled.RateQuota)) = rateLimit
			}
		},
		Usage: "max count of requests allowed in a one hour period, by remote ip address",
	},
	&support.ConfigOption{
		Name:      "rate-limit-redis-key",
		ConfigKey: &config.RateLimitRedisKey,
		OptType:   types.String,
		Usage:     "redis key for storing rate limit data, useful when deploying a cluster of Horizons, ignored when redis-url is empty",
	},
	&support.ConfigOption{
		Name:      "redis-url",
		ConfigKey: &config.RedisURL,
		OptType:   types.String,
		Usage:     "redis to connect with, for rate limiting",
	},
	&support.ConfigOption{
		Name:           "friendbot-url",
		ConfigKey:      &config.FriendbotURL,
		OptType:        types.String,
		CustomSetValue: support.SetURL,
		Usage:          "friendbot service to redirect to",
	},
	&support.ConfigOption{
		Name:        "log-level",
		ConfigKey:   &config.LogLevel,
		OptType:     types.String,
		FlagDefault: "info",
		CustomSetValue: func(co *support.ConfigOption) {
			ll, err := logrus.ParseLevel(viper.GetString(co.Name))
			if err != nil {
				stdLog.Fatalf("Could not parse log-level: %v", viper.GetString(co.Name))
			}
			*(co.ConfigKey.(*logrus.Level)) = ll
		},
		Usage: "minimum log severity (debug, info, warn, error) to log",
	},
	&support.ConfigOption{
		Name:      "log-file",
		ConfigKey: &config.LogFile,
		OptType:   types.String,
		Usage:     "name of the file where logs will be saved (leave empty to send logs to stdout)",
	},
	&support.ConfigOption{
		Name:        "max-path-length",
		ConfigKey:   &config.MaxPathLength,
		OptType:     types.Uint,
		FlagDefault: uint(4),
		Usage:       "the maximum number of assets on the path in `/paths` endpoint",
	},
	&support.ConfigOption{
		Name:      "network-passphrase",
		ConfigKey: &config.NetworkPassphrase,
		OptType:   types.String,
		Required:  true,
		Usage:     "Override the network passphrase",
	},
	&support.ConfigOption{
		Name:      "sentry-dsn",
		ConfigKey: &config.SentryDSN,
		OptType:   types.String,
		Usage:     "Sentry URL to which panics and errors should be reported",
	},
	&support.ConfigOption{
		Name:      "loggly-token",
		ConfigKey: &config.LogglyToken,
		OptType:   types.String,
		Usage:     "Loggly token, used to configure log forwarding to loggly",
	},
	&support.ConfigOption{
		Name:        "loggly-tag",
		ConfigKey:   &config.LogglyTag,
		OptType:     types.String,
		FlagDefault: "horizon",
		Usage:       "Tag to be added to every loggly log event",
	},
	&support.ConfigOption{
		Name:      "tls-cert",
		ConfigKey: &config.TLSCert,
		OptType:   types.String,
		Usage:     "TLS certificate file to use for securing connections to horizon",
	},
	&support.ConfigOption{
		Name:      "tls-key",
		ConfigKey: &config.TLSKey,
		OptType:   types.String,
		Usage:     "TLS private key file to use for securing connections to horizon",
	},
	&support.ConfigOption{
		Name:        "ingest",
		ConfigKey:   &config.Ingest,
		OptType:     types.Bool,
		FlagDefault: false,
		Usage:       "causes this horizon process to ingest data from stellar-core into horizon's db",
	},
	&support.ConfigOption{
		Name:        "history-retention-count",
		ConfigKey:   &config.HistoryRetentionCount,
		OptType:     types.Uint,
		FlagDefault: uint(0),
		Usage:       "the minimum number of ledgers to maintain within horizon's history tables.  0 signifies an unlimited number of ledgers will be retained",
	},
	&support.ConfigOption{
		Name:        "history-stale-threshold",
		ConfigKey:   &config.StaleThreshold,
		OptType:     types.Uint,
		FlagDefault: uint(0),
		Usage:       "the maximum number of ledgers the history db is allowed to be out of date from the connected stellar-core db before horizon considers history stale",
	},
	&support.ConfigOption{
		Name:        "skip-cursor-update",
		ConfigKey:   &config.SkipCursorUpdate,
		OptType:     types.Bool,
		FlagDefault: false,
		Usage:       "causes the ingester to skip reporting the last imported ledger state to stellar-core",
	},
	&support.ConfigOption{
		Name:        "enable-asset-stats",
		ConfigKey:   &config.EnableAssetStats,
		OptType:     types.Bool,
		FlagDefault: false,
		Usage:       "enables asset stats during the ingestion and expose `/assets` endpoint, Enabling it has a negative impact on CPU",
	},
}

func init() {
	for _, co := range configOpts {
		err := co.Init(rootCmd)
		if err != nil {
			stdLog.Fatal(err.Error())
		}
	}

	viper.BindPFlags(rootCmd.PersistentFlags())
}

func initApp() *horizon.App {
	initConfig()
	return horizon.NewApp(config)
}

func initConfig() {
	// Verify required options and load the config struct
	for _, co := range configOpts {
		co.Require()
		co.SetValue()
	}

	// Migrations should be checked as early as possible
	checkMigrations()

	// Validate options that should be provided together
	validateBothOrNeither("tls-cert", "tls-key")
	validateBothOrNeither("rate-limit-redis-key", "redis-url")

	// Configure log file
	if config.LogFile != "" {
		logFile, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			log.DefaultLogger.Logger.Out = logFile
		} else {
			stdLog.Fatalf("Failed to open file to log: %s", err)
		}
	}

	// Configure log level
	log.DefaultLogger.Level = config.LogLevel
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
