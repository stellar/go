// Package test contains simple test helpers that should not
// have any dependencies on horizon's packages.  think constants,
// custom matchers, generic helpers etc.
package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	tdb "github.com/stellar/go/services/horizon/internal/test/db"
	"github.com/stellar/go/support/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// StaticMockServer is a test helper that records it's last request
type StaticMockServer struct {
	*httptest.Server
	LastRequest *http.Request
}

// T provides a common set of functionality for each test in horizon
type T struct {
	T          *testing.T
	Assert     *assert.Assertions
	Require    *require.Assertions
	Ctx        context.Context
	HorizonDB  *sqlx.DB
	CoreDB     *sqlx.DB
	EndLogTest func() []logrus.Entry
}

// Context provides a context suitable for testing in tests that do not create
// a full App instance (in which case your tests should be using the app's
// context).  This context has a logger bound to it suitable for testing.
func Context() context.Context {
	return log.Set(context.Background(), testLogger)
}

// Database returns a connection to the horizon test database
//
// DEPRECATED:  use `Horizon()` from test/db package
func Database(t *testing.T) *sqlx.DB {
	return tdb.Horizon(t)
}

// DatabaseURL returns the database connection the url any test
// use when connecting to the history/horizon database
//
// DEPRECATED:  use `HorizonURL()` from test/db package
func DatabaseURL() string {
	return tdb.HorizonURL()
}

// Start initializes a new test helper object, a new instance of log,
// and conceptually "starts" a new test
func Start(t *testing.T) *T {
	result := &T{}
	result.T = t
	logger := log.New()

	result.Ctx = log.Set(context.Background(), logger)
	result.HorizonDB = Database(t)
	result.CoreDB = StellarCoreDatabase(t)
	result.Assert = assert.New(t)
	result.Require = require.New(t)
	result.EndLogTest = logger.StartTest(log.DebugLevel)

	return result
}

// StellarCoreDatabase returns a connection to the stellar core test database
//
// DEPRECATED:  use `StellarCore()` from test/db package
func StellarCoreDatabase(t *testing.T) *sqlx.DB {
	return tdb.StellarCore(t)
}

// StellarCoreDatabaseURL returns the database connection the url any test
// use when connecting to the stellar-core database
//
// DEPRECATED:  use `StellarCoreURL()` from test/db package
func StellarCoreDatabaseURL() string {
	return tdb.StellarCoreURL()
}
