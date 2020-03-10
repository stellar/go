package db

import (
	"testing"

	"github.com/stellar/go/support/db/dbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen_openAndPingSucceeds(t *testing.T) {
	db := dbtest.Postgres(t)

	sqlxDB, err := Open(db.DSN)
	require.NoError(t, err)
	assert.Equal(t, "postgres", sqlxDB.DriverName())

	err = sqlxDB.Ping()
	require.NoError(t, err)
}

func TestOpen_openAndPingFails(t *testing.T) {
	sqlxDB, err := Open("postgres://localhost:0")
	require.NoError(t, err)
	assert.Equal(t, "postgres", sqlxDB.DriverName())

	err = sqlxDB.Ping()
	require.EqualError(t, err, "dial tcp 127.0.0.1:0: connect: connection refused")
}
