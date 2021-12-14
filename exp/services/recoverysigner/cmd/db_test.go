package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	dbpkg "github.com/stellar/go/exp/services/recoverysigner/internal/db"
	"github.com/stellar/go/exp/services/recoverysigner/internal/db/dbtest"
	"github.com/stellar/go/support/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBCommand_Migrate_upDownAll(t *testing.T) {
	db := dbtest.OpenWithoutMigrations(t)
	log := log.New()

	dbCommand := DBCommand{
		Logger:      log,
		DatabaseURL: db.DSN,
	}

	// Migrate Up
	{
		dbCommand.Migrate(&cobra.Command{}, []string{"up"})

		session, err := dbpkg.Open(db.DSN)
		require.NoError(t, err)
		ids := []string{}
		err = session.Select(&ids, `SELECT id FROM gorp_migrations`)
		require.NoError(t, err)
		wantIDs := []string{
			"20200309000000-initial-1.sql",
			"20200309000001-initial-2.sql",
			"20200311000000-create-accounts.sql",
			"20200311000001-create-identities.sql",
			"20200311000002-create-auth-methods.sql",
			"20200320000000-create-accounts-audit.sql",
			"20200320000001-create-identities-audit.sql",
			"20200320000002-create-auth-methods-audit.sql",
		}
		assert.Equal(t, wantIDs, ids)
	}

	// Migrate Down
	{
		dbCommand.Migrate(&cobra.Command{}, []string{"down"})

		session, err := dbpkg.Open(db.DSN)
		require.NoError(t, err)
		ids := []string{}
		err = session.Select(&ids, `SELECT id FROM gorp_migrations`)
		require.NoError(t, err)
		assert.Empty(t, ids)
	}
}

func TestDBCommand_Migrate_upTwoDownOne(t *testing.T) {
	db := dbtest.OpenWithoutMigrations(t)
	log := log.New()

	dbCommand := DBCommand{
		Logger:      log,
		DatabaseURL: db.DSN,
	}

	// Migrate Up 2
	{
		dbCommand.Migrate(&cobra.Command{}, []string{"up", "2"})

		session, err := dbpkg.Open(db.DSN)
		require.NoError(t, err)
		ids := []string{}
		err = session.Unsafe().Select(&ids, `SELECT id FROM gorp_migrations`)
		require.NoError(t, err)
		wantIDs := []string{
			"20200309000000-initial-1.sql",
			"20200309000001-initial-2.sql",
		}
		assert.Equal(t, wantIDs, ids)
	}

	// Migrate Down 1
	{
		dbCommand.Migrate(&cobra.Command{}, []string{"down", "1"})

		session, err := dbpkg.Open(db.DSN)
		require.NoError(t, err)
		ids := []string{}
		err = session.Select(&ids, `SELECT id FROM gorp_migrations`)
		require.NoError(t, err)
		wantIDs := []string{
			"20200309000000-initial-1.sql",
		}
		assert.Equal(t, wantIDs, ids)
	}
}

func TestDBCommand_Migrate_invalidDirection(t *testing.T) {
	db := dbtest.OpenWithoutMigrations(t)
	log := log.New()

	dbCommand := DBCommand{
		Logger:      log,
		DatabaseURL: db.DSN,
	}

	dbCommand.Migrate(&cobra.Command{}, []string{"invalid"})

	session, err := dbpkg.Open(db.DSN)
	require.NoError(t, err)
	tables := []string{}
	err = session.Select(&tables, `SELECT table_name FROM information_schema.tables WHERE table_schema='public'`)
	require.NoError(t, err)
	assert.Empty(t, tables)
}

func TestDBCommand_Migrate_invalidCount(t *testing.T) {
	db := dbtest.OpenWithoutMigrations(t)
	log := log.New()

	dbCommand := DBCommand{
		Logger:      log,
		DatabaseURL: db.DSN,
	}

	dbCommand.Migrate(&cobra.Command{}, []string{"down", "invalid"})
	dbCommand.Migrate(&cobra.Command{}, []string{"up", "invalid"})

	session, err := dbpkg.Open(db.DSN)
	require.NoError(t, err)
	tables := []string{}
	err = session.Select(&tables, `SELECT table_name FROM information_schema.tables WHERE table_schema='public'`)
	require.NoError(t, err)
	assert.Empty(t, tables)
}

func TestDBCommand_Migrate_zeroCount(t *testing.T) {
	db := dbtest.OpenWithoutMigrations(t)
	log := log.New()

	dbCommand := DBCommand{
		Logger:      log,
		DatabaseURL: db.DSN,
	}

	dbCommand.Migrate(&cobra.Command{}, []string{"down", "0"})
	dbCommand.Migrate(&cobra.Command{}, []string{"up", "0"})

	session, err := dbpkg.Open(db.DSN)
	require.NoError(t, err)
	tables := []string{}
	err = session.Select(&tables, `SELECT table_name FROM information_schema.tables WHERE table_schema='public'`)
	require.NoError(t, err)
	assert.Empty(t, tables)
}
