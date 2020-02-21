package dbtest

import (
	"fmt"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func execStatement(t *testing.T, pguser, query string) {
	db, err := sqlx.Open("postgres", fmt.Sprintf("postgres://%s@localhost/?sslmode=disable", pguser))
	require.NoError(t, err)
	_, err = db.Exec(query)
	require.NoError(t, err)
	require.NoError(t, db.Close())
}

// Postgres provisions a new, blank database with a random name on the localhost
// of the running process.  It assumes that you have postgres running on the
// default port, have the command line postgres tools installed, and that the
// current user has access to the server.  It panics on the event of a failure.
func Postgres(t *testing.T) *DB {
	var result DB
	result.databaseName = randomName()
	result.Dialect = "postgres"
	result.t = t

	pguser := os.Getenv("PGUSER")
	if len(pguser) == 0 {
		pguser = "postgres"
	}

	// create the db
	execStatement(t, pguser, "CREATE DATABASE "+pq.QuoteIdentifier(result.databaseName))

	result.DSN = fmt.Sprintf("postgres://%s@localhost/%s?sslmode=disable", pguser, result.databaseName)

	result.closer = func() {
		execStatement(t, pguser, "DROP DATABASE "+pq.QuoteIdentifier(result.databaseName))
	}

	return &result
}
