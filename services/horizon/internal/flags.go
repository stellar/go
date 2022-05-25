package horizon

import (
	"fmt"
	"go/types"
	stdLog "log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/network"
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
	// StellarCoreURLFlagName is the command line flag for configuring the URL fore Stellar Core HTTP endpoint
	StellarCoreURLFlagName = "stellar-core-url"
	// StellarCoreBinaryPathName is the command line flag for configuring the path to the stellar core binary
	StellarCoreBinaryPathName = "stellar-core-binary-path"
	// captiveCoreConfigAppendPathName is the command line flag for configuring the path to the captive core additional configuration
	// Note captiveCoreConfigAppendPathName is deprecated in favor of CaptiveCoreConfigPathName
	captiveCoreConfigAppendPathName = "captive-core-config-append-path"
	// CaptiveCoreConfigPathName is the command line flag for configuring the path to the captive core configuration file
	CaptiveCoreConfigPathName = "captive-core-config-path"
	// captive-core-use-db is the command line flag for enabling captive core runtime to use an external db url connection rather than RAM for ledger states
	CaptiveCoreConfigUseDB = "captive-core-use-db"

	captiveCoreMigrationHint = "If you are migrating from Horizon 1.x.y, start with the Migration Guide here: https://developers.stellar.org/docs/run-api-server/migrating/"
)

// validateBothOrNeither ensures that both options are provided, if either is provided.
func validateBothOrNeither(option1, option2 string) error {
	arg1, arg2 := viper.GetString(option1), viper.GetString(option2)
	if arg1 != "" && arg2 == "" {
		return fmt.Errorf("Invalid config: %s = %s, but corresponding option %s is not configured", option1, arg1, option2)
	}
	if arg1 == "" && arg2 != "" {
		return fmt.Errorf("Invalid config: %s = %s, but corresponding option %s is not configured", option2, arg2, option1)
	}
	return nil
}

func applyMigrations(config Config) error {
	dbConn, err := db.Open("postgres", config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("could not connect to horizon db: %v", err)
	}
	defer dbConn.Close()

	numMigrations, err := schema.Migrate(dbConn.DB.DB, schema.MigrateUp, 0)
	if err != nil {
		return fmt.Errorf("could not apply migrations: %v", err)
	}
	if numMigrations > 0 {
		stdLog.Printf("successfully applied %v horizon migrations\n", numMigrations)
	}
	return nil
}

// checkMigrations looks for necessary database migrations and fails with a descriptive error if migrations are needed.
func checkMigrations(config Config) error {
	migrationsToApplyUp := schema.GetMigrationsUp(config.DatabaseURL)
	if len(migrationsToApplyUp) > 0 {
		return fmt.Errorf(
			`There are %v migrations to apply in the "up" direction.
The necessary migrations are: %v
A database migration is required to run this version (%v) of Horizon. Run "horizon db migrate up" to update your DB. Consult the Changelog (https://github.com/stellar/go/blob/master/services/horizon/CHANGELOG.md) for more information.`,
			len(migrationsToApplyUp),
			migrationsToApplyUp,
			apkg.Version(),
		)
	}

	nMigrationsDown := schema.GetNumMigrationsDown(config.DatabaseURL)
	if nMigrationsDown > 0 {
		return fmt.Errorf(
			`A database migration DOWN to an earlier version of the schema is required to run this version (%v) of Horizon. Consult the Changelog (https://github.com/stellar/go/blob/master/services/horizon/CHANGELOG.md) for more information.
In order to migrate the database DOWN, using the HIGHEST version number of Horizon you have installed (not this binary), run "horizon db migrate down %v".`,
			apkg.Version(),
			nMigrationsDown,
		)
	}
	return nil
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
			Name:      "ro-database-url",
			ConfigKey: &config.RoDatabaseURL,
			OptType:   types.String,
			Required:  false,
			Usage:     "horizon postgres read-replica to connect with, when set it will return stale history error when replica is behind primary",
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
			Name:        captiveCoreConfigAppendPathName,
			OptType:     types.String,
			FlagDefault: "",
			Required:    false,
			Usage:       "DEPRECATED in favor of " + CaptiveCoreConfigPathName,
			CustomSetValue: func(opt *support.ConfigOption) error {
				if val := viper.GetString(opt.Name); val != "" {
					if viper.GetString(CaptiveCoreConfigPathName) != "" {
						stdLog.Printf(
							"both --%s and --%s are set. %s is deprecated so %s will be used instead",
							captiveCoreConfigAppendPathName,
							CaptiveCoreConfigPathName,
							captiveCoreConfigAppendPathName,
							CaptiveCoreConfigPathName,
						)
					} else {
						config.CaptiveCoreConfigPath = val
					}
				}
				return nil
			},
		},
		&support.ConfigOption{
			Name:        CaptiveCoreConfigPathName,
			OptType:     types.String,
			FlagDefault: "",
			Required:    false,
			Usage:       "path to the configuration file used by captive core. It must, at least, include enough details to define a quorum set. Any fields in the configuration file which are not supported by captive core will be rejected.",
			CustomSetValue: func(opt *support.ConfigOption) error {
				if val := viper.GetString(opt.Name); val != "" {
					config.CaptiveCoreConfigPath = val
					config.CaptiveCoreTomlParams.Strict = true
				}
				return nil
			},
		},
		&support.ConfigOption{
			Name:        CaptiveCoreConfigUseDB,
			OptType:     types.Bool,
			FlagDefault: false,
			Required:    false,
			Usage: `when enabled, Horizon ingestion will instruct the captive
			              core invocation to use an external db url for ledger states rather than in memory(RAM).\n 
						  Will result in several GB of space shifting out of RAM and to the external db persistence.\n
						  The external db url is determined by the presence of DATABASE parameter in the captive-core-config-path or\n
						  or if absent, the db will default to sqlite and the db file will be stored at location derived from captive-core-storage-path parameter.`,
			CustomSetValue: func(opt *support.ConfigOption) error {
				if val := viper.GetBool(opt.Name); val {
					config.CaptiveCoreConfigUseDB = val
					config.CaptiveCoreTomlParams.UseDB = val
				}
				return nil
			},
			ConfigKey: &config.CaptiveCoreConfigUseDB,
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
			Name:        "exp-enable-ingestion-filtering",
			OptType:     types.Bool,
			FlagDefault: false,
			Required:    false,
			Usage:       "causes Horizon to enable the experimental Ingestion Filtering and the ingestion admin HTTP endpoint at /ingestion/filter",
			ConfigKey:   &config.EnableIngestionFiltering,
		},
		&support.ConfigOption{
			Name:           "captive-core-http-port",
			OptType:        types.Uint,
			CustomSetValue: support.SetOptionalUint,
			Required:       false,
			FlagDefault:    uint(0),
			Usage:          "HTTP port for Captive Core to listen on (0 disables the HTTP server)",
			ConfigKey:      &config.CaptiveCoreTomlParams.HTTPPort,
		},
		&support.ConfigOption{
			Name:    "captive-core-storage-path",
			OptType: types.String,
			CustomSetValue: func(opt *support.ConfigOption) error {
				existingValue := viper.GetString(opt.Name)
				if existingValue == "" || existingValue == "." {
					cwd, err := os.Getwd()
					if err != nil {
						return fmt.Errorf("Unable to determine the current directory: %s", err)
					}
					existingValue = cwd
				}
				*opt.ConfigKey.(*string) = existingValue
				return nil
			},
			Required:  false,
			Usage:     "Storage location for Captive Core bucket data",
			ConfigKey: &config.CaptiveCoreStoragePath,
		},
		&support.ConfigOption{
			Name:           "captive-core-peer-port",
			OptType:        types.Uint,
			FlagDefault:    uint(0),
			CustomSetValue: support.SetOptionalUint,
			Required:       false,
			Usage:          "port for Captive Core to bind to for connecting to the Stellar swarm (0 uses Stellar Core's default)",
			ConfigKey:      &config.CaptiveCoreTomlParams.PeerPort,
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
			CustomSetValue: func(co *support.ConfigOption) error {
				stringOfUrls := viper.GetString(co.Name)
				urlStrings := strings.Split(stringOfUrls, ",")
				*(co.ConfigKey.(*[]string)) = urlStrings
				return nil
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
			CustomSetValue: func(co *support.ConfigOption) error {
				var rateLimit *throttled.RateQuota = nil
				perHourRateLimit := viper.GetInt(co.Name)
				if perHourRateLimit != 0 {
					rateLimit = &throttled.RateQuota{
						MaxRate:  throttled.PerHour(perHourRateLimit),
						MaxBurst: 100,
					}
					*(co.ConfigKey.(**throttled.RateQuota)) = rateLimit
				}
				return nil
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
			CustomSetValue: func(co *support.ConfigOption) error {
				ll, err := logrus.ParseLevel(viper.GetString(co.Name))
				if err != nil {
					return fmt.Errorf("Could not parse log-level: %v", viper.GetString(co.Name))
				}
				*(co.ConfigKey.(*logrus.Level)) = ll
				return nil
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
			Name:           "captive-core-log-path",
			ConfigKey:      &config.CaptiveCoreTomlParams.LogPath,
			OptType:        types.String,
			CustomSetValue: support.SetOptionalString,
			Required:       false,
			Usage:          "name of the path for Core logs (leave empty to log w/ Horizon only)",
		},
		&support.ConfigOption{
			Name:        "max-path-length",
			ConfigKey:   &config.MaxPathLength,
			OptType:     types.Uint,
			FlagDefault: uint(3),
			Usage:       "the maximum number of assets on the path in `/paths` endpoint, warning: increasing this value will increase /paths response time",
		},
		&support.ConfigOption{
			Name:        "max-assets-per-path-request",
			ConfigKey:   &config.MaxAssetsPerPathRequest,
			OptType:     types.Int,
			FlagDefault: int(15),
			Usage:       "the maximum number of assets in '/paths/strict-send' and '/paths/strict-recieve' endpoints",
		},
		&support.ConfigOption{
			Name:        "disable-pool-path-finding",
			ConfigKey:   &config.DisablePoolPathFinding,
			OptType:     types.Bool,
			FlagDefault: false,
			Required:    false,
			Usage:       "excludes liquidity pools from consideration in the `/paths` endpoint",
		},
		&support.ConfigOption{
			Name:        "disable-path-finding",
			ConfigKey:   &config.DisablePathFinding,
			OptType:     types.Bool,
			FlagDefault: false,
			Required:    false,
			Usage:       "disables the path finding endpoints",
		},
		&support.ConfigOption{
			Name:        "max-path-finding-requests",
			ConfigKey:   &config.MaxPathFindingRequests,
			OptType:     types.Uint,
			FlagDefault: uint(0),
			Required:    false,
			Usage: "The maximum number of path finding requests per second horizon will allow." +
				" A value of zero (the default) disables the limit.",
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
			FlagDefault: true,
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
			Name:        "ingest-enable-extended-log-ledger-stats",
			ConfigKey:   &config.IngestEnableExtendedLogLedgerStats,
			OptType:     types.Bool,
			FlagDefault: false,
			Usage:       "enables extended ledger stats in the log (ledger entry changes and operations stats)",
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
		&support.ConfigOption{
			Name:        "rounding-slippage-filter",
			ConfigKey:   &config.RoundingSlippageFilter,
			OptType:     types.Int,
			FlagDefault: 1000,
			Required:    false,
			Usage:       "excludes trades from /trade_aggregations unless their rounding slippage is <x bps",
		},
	}

	return config, flags
}

// NewAppFromFlags constructs a new Horizon App from the given command line flags
func NewAppFromFlags(config *Config, flags support.ConfigOptions) (*App, error) {
	err := ApplyFlags(config, flags, ApplyOptions{RequireCaptiveCoreConfig: true, AlwaysIngest: false})
	if err != nil {
		return nil, err
	}
	// Validate app-specific arguments
	if config.StellarCoreURL == "" {
		return nil, fmt.Errorf("flag --%s cannot be empty", StellarCoreURLFlagName)
	}
	if config.Ingest && !config.EnableCaptiveCoreIngestion && config.StellarCoreDatabaseURL == "" {
		return nil, fmt.Errorf("flag --%s cannot be empty", StellarCoreDBURLFlagName)
	}

	log.Infof("Initializing horizon...")
	app, err := NewApp(*config)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize app: %s", err)
	}
	return app, nil
}

type ApplyOptions struct {
	AlwaysIngest             bool
	RequireCaptiveCoreConfig bool
}

// ApplyFlags applies the command line flags on the given Config instance
func ApplyFlags(config *Config, flags support.ConfigOptions, options ApplyOptions) error {
	// Verify required options and load the config struct
	if err := flags.RequireE(); err != nil {
		return err
	}
	if err := flags.SetValues(); err != nil {
		return err
	}

	// Validate options that should be provided together
	if err := validateBothOrNeither("tls-cert", "tls-key"); err != nil {
		return err
	}

	if options.AlwaysIngest {
		config.Ingest = true
	}

	if config.Ingest {
		// Migrations should be checked as early as possible. Apply and check
		// only on ingesting instances which are required to have write-access
		// to the DB.
		if config.ApplyMigrations {
			stdLog.Println("Applying DB migrations...")
			if err := applyMigrations(*config); err != nil {
				return err
			}
		}
		stdLog.Println("Checking DB migrations...")
		if err := checkMigrations(*config); err != nil {
			return err
		}

		// config.HistoryArchiveURLs contains a single empty value when empty so using
		// viper.GetString is easier.
		if len(config.HistoryArchiveURLs) == 1 && config.HistoryArchiveURLs[0] == "" {
			return fmt.Errorf("--history-archive-urls must be set when --ingest is set")
		}

		if config.EnableCaptiveCoreIngestion {
			stdLog.Println("Preparing captive core...")

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

			// NOTE: If both of these are set (regardless of user- or PATH-supplied
			//       defaults for the binary path), the Remote Captive Core URL
			//       takes precedence.
			if binaryPath == "" && config.RemoteCaptiveCoreURL == "" {
				return fmt.Errorf("Invalid config: captive core requires that either --%s or --remote-captive-core-url is set. %s",
					StellarCoreBinaryPathName, captiveCoreMigrationHint)
			}

			if config.RemoteCaptiveCoreURL == "" && (binaryPath == "" || config.CaptiveCoreConfigPath == "") {
				if options.RequireCaptiveCoreConfig {
					var err error
					errorMessage := fmt.Errorf(
						"Invalid config: captive core requires that both --%s and --%s are set. %s",
						StellarCoreBinaryPathName, CaptiveCoreConfigPathName, captiveCoreMigrationHint,
					)

					var configFileName string
					// Default config files will be located along the binary in the release archive.
					switch config.NetworkPassphrase {
					case network.TestNetworkPassphrase:
						configFileName = "captive-core-testnet.cfg"
						config.HistoryArchiveURLs = []string{"https://history.stellar.org/prd/core-testnet/core_testnet_001/"}
					case network.PublicNetworkPassphrase:
						configFileName = "captive-core-pubnet.cfg"
						config.HistoryArchiveURLs = []string{"https://history.stellar.org/prd/core-live/core_live_001/"}
						config.UsingDefaultPubnetConfig = true
					default:
						return errorMessage
					}

					executablePath, err := os.Executable()
					if err != nil {
						return errorMessage
					}

					config.CaptiveCoreConfigPath = filepath.Join(filepath.Dir(executablePath), configFileName)
					if _, err = os.Stat(config.CaptiveCoreConfigPath); os.IsNotExist(err) {
						return errorMessage
					}

					config.CaptiveCoreTomlParams.NetworkPassphrase = config.NetworkPassphrase
					config.CaptiveCoreToml, err = ledgerbackend.NewCaptiveCoreTomlFromFile(config.CaptiveCoreConfigPath, config.CaptiveCoreTomlParams)
					if err != nil {
						return fmt.Errorf("Invalid captive core toml file %v", err)
					}
				} else {
					var err error
					config.CaptiveCoreTomlParams.HistoryArchiveURLs = config.HistoryArchiveURLs
					config.CaptiveCoreTomlParams.NetworkPassphrase = config.NetworkPassphrase
					config.CaptiveCoreToml, err = ledgerbackend.NewCaptiveCoreToml(config.CaptiveCoreTomlParams)
					if err != nil {
						return fmt.Errorf("Invalid captive core toml file %v", err)
					}
				}
			} else if config.RemoteCaptiveCoreURL == "" {
				var err error
				config.CaptiveCoreTomlParams.HistoryArchiveURLs = config.HistoryArchiveURLs
				config.CaptiveCoreTomlParams.NetworkPassphrase = config.NetworkPassphrase
				config.CaptiveCoreToml, err = ledgerbackend.NewCaptiveCoreTomlFromFile(config.CaptiveCoreConfigPath, config.CaptiveCoreTomlParams)
				if err != nil {
					return fmt.Errorf("Invalid captive core toml file %v", err)
				}
			}

			// If we don't supply an explicit core URL and we are running a local
			// captive core process with the http port enabled, point to it.
			if config.StellarCoreURL == "" && config.RemoteCaptiveCoreURL == "" && config.CaptiveCoreToml.HTTPPort != 0 {
				config.StellarCoreURL = fmt.Sprintf("http://localhost:%d", config.CaptiveCoreToml.HTTPPort)
				viper.Set(StellarCoreURLFlagName, config.StellarCoreURL)
			}
		}
	} else {
		if config.EnableCaptiveCoreIngestion && (config.CaptiveCoreBinaryPath != "" || config.CaptiveCoreConfigPath != "") {
			captiveCoreConfigFlag := captiveCoreConfigAppendPathName
			if viper.GetString(CaptiveCoreConfigPathName) != "" {
				captiveCoreConfigFlag = CaptiveCoreConfigPathName
			}
			return fmt.Errorf("Invalid config: one or more captive core params passed (--%s or --%s) but --ingest not set. "+captiveCoreMigrationHint,
				StellarCoreBinaryPathName, captiveCoreConfigFlag)
		}
		if config.StellarCoreDatabaseURL != "" {
			return fmt.Errorf("Invalid config: --%s passed but --ingest not set. ", StellarCoreDBURLFlagName)
		}
		if config.CaptiveCoreConfigUseDB {
			return fmt.Errorf("Invalid config: --%s has been set, but --ingest not set. ", CaptiveCoreConfigUseDB)
		}
	}

	// Configure log file
	if config.LogFile != "" {
		logFile, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			log.DefaultLogger.SetOutput(logFile)
		} else {
			return fmt.Errorf("Failed to open file to log: %s", err)
		}
	}

	// Configure log level
	log.DefaultLogger.SetLevel(config.LogLevel)

	// Configure DB params. When config.MaxDBConnections is set, set other
	// DB params to that value for backward compatibility.
	if config.MaxDBConnections != 0 {
		config.HorizonDBMaxOpenConnections = config.MaxDBConnections
		config.HorizonDBMaxIdleConnections = config.MaxDBConnections
	}

	if config.BehindCloudflare && config.BehindAWSLoadBalancer {
		return fmt.Errorf("Invalid config: Only one option of --behind-cloudflare and --behind-aws-load-balancer is allowed. If Horizon is behind both, use --behind-cloudflare only.")
	}

	return nil
}
