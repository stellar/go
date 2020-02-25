package dbtest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPostgres(t *testing.T) {
	db := Postgres(t)
	t.Log("tempdb url", db.DSN)

	conn := db.Open()
	_, err := conn.Exec("SELECT 1")
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	db.Close()

	conn = db.Open()
	_, err = conn.Exec("SELECT 1")
	require.EqualError(t, err, fmt.Sprintf("pq: database \"%s\" does not exist", db.dbName))
	require.Contains(t, err.Error(), "data")

	// ensure Close() can be called multiple times
	db.Close()
}
