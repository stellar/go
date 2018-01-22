package database

import (
	"database/sql"
	"os"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/require"
)

const defaultTestDSN = "postgres://root:mysecretpassword@127.0.0.1:5432/circle_test?sslmode=disable"

var once sync.Once

func OpenTestDB(t *testing.T) *sqlx.DB {
	dsn := os.Getenv("bifrostTestDBAddress")
	if dsn == "" {
		dsn = defaultTestDSN
	}
	db, err := sqlx.Open("postgres", dsn)
	require.NoError(t, err)
	once.Do(migrateSchema(t, db.DB))
	return db
}

func migrateSchema(t *testing.T, db *sql.DB) func() {
	return func() {
		migrations := &migrate.FileMigrationSource{
			Dir: "./migrations",
		}
		_, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
		require.NoError(t, err)
	}
}
