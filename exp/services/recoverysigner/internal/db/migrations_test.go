package db

import (
	"testing"

	migrate "github.com/rubenv/sql-migrate"
	"github.com/stellar/go/exp/services/recoverysigner/internal/db/dbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanMigration_noneToApply(t *testing.T) {
	db := dbtest.Open(t)
	session := db.Open()

	migrations, err := PlanMigration(session, migrate.Up, 0)
	require.NoError(t, err)
	wantMigrations := []string{}
	assert.Equal(t, wantMigrations, migrations)
}

func TestPlanMigration_someToApply(t *testing.T) {
	db := dbtest.Open(t)
	session := db.Open()

	migrations, err := PlanMigration(session, migrate.Down, 0)
	require.NoError(t, err)
	wantMigrations := []string{"20200309000000-initial.sql"}
	assert.Equal(t, wantMigrations, migrations)
}

func TestMigrate_noneToApply(t *testing.T) {
	db := dbtest.Open(t)
	session := db.Open()

	n, err := Migrate(session, migrate.Up, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestMigrate_someToApply(t *testing.T) {
	db := dbtest.Open(t)
	session := db.Open()

	n, err := Migrate(session, migrate.Down, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}
