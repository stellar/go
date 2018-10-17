package horizon

import (
	"net/url"

	"github.com/PuerkitoBio/throttled"
	"github.com/sirupsen/logrus"
)

// Config is the configuration for horizon.  It get's populated by the
// app's main function and is provided to NewApp.
type Config struct {
	DatabaseURL            string
	StellarCoreDatabaseURL string
	StellarCoreURL         string
	Port                   int
	RateLimit              throttled.Quota
	RedisURL               string
	FriendbotURL           *url.URL
	LogLevel               logrus.Level
	LogFile                string
	SentryDSN              string
	LogglyTag              string
	LogglyToken            string
	// Maximum length of the path returned by `/paths` endpoint.
	MaxPathLength uint
	// TLSCert is a path to a certificate file to use for horizon's TLS config
	TLSCert string
	// TLSKey is the path to a private key file to use for horizon's TLS config
	TLSKey string
	// Ingest is a boolean that indicates whether or not this horizon instance
	// should run the data ingestion subsystem.
	Ingest bool
	// HistoryRetentionCount represents the minimum number of ledgers worth of
	// history data to retain in the horizon database. For the purposes of
	// determining a "retention duration", each ledger roughly corresponds to 10
	// seconds of real time.
	HistoryRetentionCount uint
	// StaleThreshold represents the number of ledgers a history database may be
	// out-of-date by before horizon begins to respond with an error to history
	// requests.
	StaleThreshold uint
	// SkipCursorUpdate causes the ingestor to skip reporting the "last imported
	// ledger" state to stellar-core.
	SkipCursorUpdate bool
	// DisableAssetStats is a feature flag that determines whether to calculate
	// asset stats during the ingestion and expose `/assets` endpoint.
	// Disabling it will save CPU when ingesting ledgers full of many different
	// assets related operations.
	DisableAssetStats bool
	// AllowEmptyLedgerDataResponses is a feature flag that sets unavailable
	// ledger data (like `close_time`) to `nil` instead of returning 500 error
	// response.
	AllowEmptyLedgerDataResponses bool
}
