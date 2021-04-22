package horizon

import (
	"fmt"
	"go/types"
	stdLog "log"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stellar/go/services/horizon/internal/db2/schema"
	apkg "github.com/stellar/go/support/app"
	support "github.com/stellar/go/support/config"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/log"
	"github.com/stellar/throttled"
)

const (
	// DatabaseURLFlagName is the command line flag for configuring the Horizon postgres URL
	DatabaseURLFlagName = "db-url"
	// StellarCoreDBURLFlagName is the command line flag for configuring the postgres Stellar Core URL
	StellarCoreDBURLFlagName = "stellar-core-db-url"
	// StellarCoreDBURLFlagName is the command line flag for configuring the URL fore Stellar Core HTTP endpoint
	StellarCoreURLFlagName = "stellar-core-url"
	// StellarCoreBinaryPathName is the command line flag for configuring the path to the stellar core binary
	StellarCoreBinaryPathName = "stellar-core-binary-path"
	// CaptiveCoreConfigAppendPathName is the command line flag for configuring the path to the captive core additional configuration
	CaptiveCoreConfigAppendPathName = "captive-core-config-append-path"

	captiveCoreMigrationHint = "If you are migrating from Horizon 1.x.y read the Migration Guide here: https://github.com/stellar/go/blob/master/services/horizon/internal/docs/captive_core.md"
)

// validateBothOrNeither ensures that both options are provided, if either is provided.
func validateBothOrNeither(option1, option2 string) {
	arg1, arg2 := viper.GetString(option1), viper.GetString(option2)
	if arg1 != "" && arg2 == "" {
		stdLog.Fatalf("Invalid config: %s = %s, but corresponding option %s is not configured", option1, arg1, option2)
	}
	if arg1 == "" && arg2 != "" {
		stdLog.Fatalf("Invalid config: %s = %s, but corresponding option %s is not configured", option2, arg2, option1)
	}
}

func applyMigrations(config Config) {
	dbConn, err := db.Open("postgres", config.DatabaseURL)
	if err != nil {
		stdLog.Fatalf("could not connect to horizon db: %v", err)
	}
	defer dbConn.Close()

	numMigrations, err := schema.Migrate(dbConn.DB.DB, schema.MigrateUp, 0)
	if err != nil {
		stdLog.Fatalf("could not apply migrations: %v", err)
	}
	if numMigrations > 0 {
		stdLog.Printf("successfully applied %v horizon migrations\n", numMigrations)
	}
}

// checkMigrations looks for necessary database migrations and fails with a descriptive error if migrations are needed.
func checkMigrations(config Config) {
	migrationsToApplyUp := schema.GetMigrationsUp(config.DatabaseURL)
	if len(migrationsToApplyUp) > 0 {
		stdLog.Printf(`There are %v migrations to apply in the "up" direction.`, len(migrationsToApplyUp))
		stdLog.Printf("The necessary migrations are: %v", migrationsToApplyUp)
		stdLog.Printf("A database migration is required to run this version (%v) of Horizon. Run \"horizon db migrate up\" to update your DB. Consult the Changelog (https://github.com/stellar/go/blob/master/services/horizon/CHANGELOG.md) for more information.", apkg.Version())
		os.Exit(1)
	}

	nMigrationsDown := schema.GetNumMigrationsDown(config.DatabaseURL)
	if nMigrationsDown > 0 {
		stdLog.Printf("A database migration DOWN to an earlier version of the schema is required to run this version (%v) of Horizon. Consult the Changelog (https://github.com/stellar/go/blob/master/services/horizon/CHANGELOG.md) for more information.", apkg.Version())
		stdLog.Printf("In order to migrate the database DOWN, using the HIGHEST version number of Horizon you have installed (not this binary), run \"horizon db migrate down %v\".", nMigrationsDown)
		os.Exit(1)
	}
}

// Flags returns a Config instance and a list of commandline flags which modify the Config instance
func Flags() (*Config, support.ConfigOptions) {
	config := &Config{}

	// flags defines the complete flag configuration for horizon.
	// Add a new entry here to connect a new field in the horizon.Config struct
	var flags = support.ConfigOptions{
		&support.ConfigOption{
			Name:      DatabaseURLFlagName,
			EnvVar:    "DATABASE_URL",
			ConfigKey: &config.DatabaseURL,
			OptType:   types.String,
			Required:  true,
			Usage:     "horizon postgres database to connect with",
		},
		&support.ConfigOption{
			Name:        StellarCoreBinaryPathName,
			OptType:     types.String,
			FlagDefault: "",
			Required:    false,
			Usage:       "path to stellar core binary (--remote-captive-core-url has higher precedence). If captive core is enabled, look for the stellar-core binary in $PATH by default.",
			ConfigKey:   &config.CaptiveCoreBinaryPath,
		},
		&support.ConfigOption{
			Name:        "remote-captive-core-url",
			OptType:     types.String,
			FlagDefault: "",
			Required:    false,
			Usage:       "url to access the remote captive core server",
			ConfigKey:   &config.RemoteCaptiveCoreURL,
		},
		&support.ConfigOption{
			Name:        CaptiveCoreConfigAppendPathName,
			OptType:     types.String,
			FlagDefault: "",
			Required:    false,
			Usage:       "path to additional configuration for the Stellar Core configuration file used by captive core. It must, at least, include enough details to define a quorum set",
			ConfigKey:   &config.CaptiveCoreConfigAppendPath,
		},
		&support.ConfigOption{
			Name:        "enable-captive-core-ingestion",
			OptType:     types.Bool,
			FlagDefault: true,
			Required:    false,
			Usage:       "causes Horizon to ingest from a Captive Stellar Core process instead of a persistent Stellar Core database",
			ConfigKey:   &config.EnableCaptiveCoreIngestion,
		},
		&support.ConfigOption{
			Name:        "captive-core-http-port",
			OptType:     types.Uint,
			FlagDefault: uint(11626),
			Required:    false,
			Usage:       "HTTP port for Captive Core to listen on (0 disables the HTTP server)",
			ConfigKey:   &config.CaptiveCoreHTTPPort,
		},
		&support.ConfigOption{
			Name:        "captive-core-storage-path",
			OptType:     types.String,
			FlagDefault: "",
			CustomSetValue: func(opt *support.ConfigOption) {
				existingValue := viper.GetString(opt.Name)
				if existingValue == "" || existingValue == "." {
					cwd, err := os.Getwd()
					if err != nil {
						stdLog.Fatalf("Unable to determine the current directory: %s", err)
					}
					existingValue = cwd
				}
				*opt.ConfigKey.(*string) = existingValue
			},
			Required:  false,
			Usage:     "Storage location for Captive Core bucket data",
			ConfigKey: &config.CaptiveCoreStoragePath,
		},
		&support.ConfigOption{
			Name:        "captive-core-peer-port",
			OptType:     types.Uint,
			FlagDefault: uint(0),
			Required:    false,
			Usage:       "port for Captive Core to bind to for connecting to the Stellar swarm (0 uses Stellar Core's default)",
			ConfigKey:   &config.CaptiveCorePeerPort,
		},
		&support.ConfigOption{
			Name:      StellarCoreDBURLFlagName,
			EnvVar:    "STELLAR_CORE_DATABASE_URL",
			ConfigKey: &config.StellarCoreDatabaseURL,
			OptType:   types.String,
			Required:  false,
			Usage:     "stellar-core postgres database to connect with",
		},
		&support.ConfigOption{
			Name:      StellarCoreURLFlagName,
			ConfigKey: &config.StellarCoreURL,
			OptType:   types.String,
			Usage:     "stellar-core to connect with (for http commands). If unset and the local Captive core is enabled, it will use http://localhost:<stellar_captive_core_http_port>",
		},
		&support.ConfigOption{
			Name:        "history-archive-urls",
			ConfigKey:   &config.HistoryArchiveURLs,
			OptType:     types.String,
			Required:    false,
			FlagDefault: "",
			CustomSetValue: func(co *support.ConfigOption) {
				stringOfUrls := viper.GetString(co.Name)
				urlStrings := strings.Split(stringOfUrls, ",")

				*(co.ConfigKey.(*[]string)) = urlStrings
			},
			Usage: "comma-separated list of stellar history archives to connect with",
		},
		&support.ConfigOption{
			Name:        "port",
			ConfigKey:   &config.Port,
			OptType:     types.Uint,
			FlagDefault: uint(8000),
			Usage:       "tcp port to listen on for http requests",
		},
		&support.ConfigOption{
			Name:        "admin-port",
			ConfigKey:   &config.AdminPort,
			OptType:     types.Uint,
			FlagDefault: uint(0),
			Usage:       "WARNING: this should not be accessible from the Internet and does not use TLS, tcp port to listen on for admin http requests, 0 (default) disables the admin server",
		},
		&support.ConfigOption{
			Name:        "max-db-connections",
			ConfigKey:   &config.MaxDBConnections,
			OptType:     types.Int,
			FlagDefault: 0,
			Usage:       "when set has a priority over horizon-db-max-open-connections, horizon-db-max-idle-connections. max horizon database open connections may need to be increased when responses are slow but DB CPU is normal",
		},
		&support.ConfigOption{
			Name:        "horizon-db-max-open-connections",
			ConfigKey:   &config.HorizonDBMaxOpenConnections,
			OptType:     types.Int,
			FlagDefault: 20,
			Usage:       "max horizon database open connections. may need to be increased when responses are slow but DB CPU is normal",
		},
		&support.ConfigOption{
			Name:        "horizon-db-max-idle-connections",
			ConfigKey:   &config.HorizonDBMaxIdleConnections,
			OptType:     types.Int,
			FlagDefault: 20,
			Usage:       "max horizon database idle connections. may need to be set to the same value as horizon-db-max-open-connections when responses are slow and DB CPU is normal, because it may indicate that a lot of time is spent closing/opening idle connections. This can happen in case of high variance in number of requests. must be equal or lower than max open connections",
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
			Usage:          "defines the timeout of connection after which 504 response will be sent or stream will be closed, if Horizon is behind a load balancer with idle connection timeout, this should be set to a few seconds less that idle timeout, does not apply to POST /transactions",
		},
		&support.ConfigOption{
			Name:        "per-hour-rate-limit",
			ConfigKey:   &config.RateQuota,
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
			Name:      "captive-core-log-path",
			ConfigKey: &config.CaptiveCoreLogPath,
			OptType:   types.String,
			Usage:     "name of the path for Core logs (leave empty to log w/ Horizon only)",
		},
		&support.ConfigOption{
			Name:        "max-path-length",
			ConfigKey:   &config.MaxPathLength,
			OptType:     types.Uint,
			FlagDefault: uint(3),
			Usage:       "the maximum number of assets on the path in `/paths` endpoint, warning: increasing this value will increase /paths response time",
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
			Name:        "cursor-name",
			EnvVar:      "CURSOR_NAME",
			ConfigKey:   &config.CursorName,
			OptType:     types.String,
			FlagDefault: "HORIZON",
			Usage:       "ingestor cursor used by horizon to ingest from stellar core. must be uppercase and unique for each horizon instance ingesting from that core instance.",
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
			Name:        "ingest-disable-state-verification",
			ConfigKey:   &config.IngestDisableStateVerification,
			OptType:     types.Bool,
			FlagDefault: false,
			Usage:       "ingestion system runs a verification routing to compare state in local database with history buckets, this can be disabled however it's not recommended",
		},
		&support.ConfigOption{
			Name:        "apply-migrations",
			ConfigKey:   &config.ApplyMigrations,
			OptType:     types.Bool,
			FlagDefault: false,
			Required:    false,
			Usage:       "applies pending migrations before starting horizon",
		},
		&support.ConfigOption{
			Name:        "skip-migrations-check",
			ConfigKey:   &config.SkipMigrationsCheck,
			OptType:     types.Bool,
			FlagDefault: false,
			Required:    false,
			Usage:       "skips checking if there are migrations required",
		},
		&support.ConfigOption{
			Name:        "checkpoint-frequency",
			ConfigKey:   &config.CheckpointFrequency,
			OptType:     types.Uint32,
			FlagDefault: uint32(64),
			Required:    false,
			Usage:       "establishes how many ledgers exist between checkpoints, do NOT change this unless you really know what you are doing",
		},
		&support.ConfigOption{
			Name:        "behind-cloudflare",
			ConfigKey:   &config.BehindCloudflare,
			OptType:     types.Bool,
			FlagDefault: false,
			Required:    false,
			Usage:       "determines if Horizon instance is behind Cloudflare, in such case client IP in the logs will be replaced with Cloudflare header (cannot be used with --behind-aws-load-balancer)",
		},
		&support.ConfigOption{
			Name:        "behind-aws-load-balancer",
			ConfigKey:   &config.BehindAWSLoadBalancer,
			OptType:     types.Bool,
			FlagDefault: false,
			Required:    false,
			Usage:       "determines if Horizon instance is behind AWS load balances like ELB or ALB, in such case client IP in the logs will be replaced with the last IP in X-Forwarded-For header (cannot be used with --behind-cloudflare)",
		},
	}

	return config, flags
}

// NewAppFromFlags constructs a new Horizon App from the given command line flags
func NewAppFromFlags(config *Config, flags support.ConfigOptions) *App {
	ApplyFlags(config, flags)
	// Validate app-specific arguments
	if config.StellarCoreURL == "" {
		log.Fatalf("flag --%s cannot be empty", StellarCoreURLFlagName)
	}
	if config.Ingest && !config.EnableCaptiveCoreIngestion && config.StellarCoreDatabaseURL == "" {
		log.Fatalf("flag --%s cannot be empty", StellarCoreDBURLFlagName)
	}
	app, err := NewApp(*config)
	if err != nil {
		log.Fatalf("cannot initialize app: %s", err)
	}
	return app
}

// ApplyFlags applies the command line flags on the given Config instance
func ApplyFlags(config *Config, flags support.ConfigOptions) {
	// Verify required options and load the config struct
	flags.Require()
	flags.SetValues()

	if config.ApplyMigrations {
		applyMigrations(*config)
	}

	// Migrations should be checked as early as possible
	if !config.SkipMigrationsCheck {
		checkMigrations(*config)
	}

	// Validate options that should be provided together
	validateBothOrNeither("tls-cert", "tls-key")

	if config.Ingest {

		// config.HistoryArchiveURLs contains a single empty value when empty so using
		// viper.GetString is easier.
		if len(config.HistoryArchiveURLs) == 0 {
			stdLog.Fatalf("--history-archive-urls must be set when --ingest is set")
		}

		if config.EnableCaptiveCoreIngestion {
			binaryPath := viper.GetString(StellarCoreBinaryPathName)

			// If the user didn't specify a Stellar Core binary, we can check the
			// $PATH and possibly fill it in for them.
			if binaryPath == "" {
				if result, err := exec.LookPath("stellar-core"); err == nil {
					binaryPath = result
					viper.Set(StellarCoreBinaryPathName, binaryPath)
					config.CaptiveCoreBinaryPath = binaryPath
				}
			}

			// When running live ingestion a config file is required too
			validateBothOrNeither(StellarCoreBinaryPathName, CaptiveCoreConfigAppendPathName)

			// NOTE: If both of these are set (regardless of user- or PATH-supplied
			//       defaults for the binary path), the Remote Captive Core URL
			//       takes precedence.
			if binaryPath == "" && config.RemoteCaptiveCoreURL == "" {
				stdLog.Fatalf("Invalid config: captive core requires that either --%s or --remote-captive-core-url is set. %s",
					StellarCoreBinaryPathName, captiveCoreMigrationHint)
			}

			if binaryPath == "" || config.CaptiveCoreConfigAppendPath == "" {
				stdLog.Fatalf("Invalid config: captive core requires that both --%s and --%s are set. %s",
					StellarCoreBinaryPathName, CaptiveCoreConfigAppendPathName, captiveCoreMigrationHint)
			}

			// If we don't supply an explicit core URL and we are running a local
			// captive core process with the http port enabled, point to it.
			if config.StellarCoreURL == "" && config.RemoteCaptiveCoreURL == "" && config.CaptiveCoreHTTPPort != 0 {
				config.StellarCoreURL = fmt.Sprintf("http://localhost:%d", config.CaptiveCoreHTTPPort)
				viper.Set(StellarCoreURLFlagName, config.StellarCoreURL)
			}
		}
	} else {
		if config.EnableCaptiveCoreIngestion && (config.CaptiveCoreBinaryPath != "" || config.CaptiveCoreConfigAppendPath != "") {
			stdLog.Fatalf("Invalid config: one or more captive core params passed (--%s or --%s) but --ingest not set. "+captiveCoreMigrationHint,
				StellarCoreBinaryPathName, CaptiveCoreConfigAppendPathName)
		}
		if config.StellarCoreDatabaseURL != "" {
			stdLog.Fatalf("Invalid config: --%s passed but --ingest not set. ", StellarCoreDBURLFlagName)
		}
	}

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
	log.DefaultLogger.Logger.SetLevel(config.LogLevel)

	// Configure DB params. When config.MaxDBConnections is set, set other
	// DB params to that value for backward compatibility.
	if config.MaxDBConnections != 0 {
		config.HorizonDBMaxOpenConnections = config.MaxDBConnections
		config.HorizonDBMaxIdleConnections = config.MaxDBConnections
	}

	if config.BehindCloudflare && config.BehindAWSLoadBalancer {
		stdLog.Fatal("Invalid config: Only one option of --behind-cloudflare and --behind-aws-load-balancer is allowed. If Horizon is behind both, use --behind-cloudflare only.")
	}
}
