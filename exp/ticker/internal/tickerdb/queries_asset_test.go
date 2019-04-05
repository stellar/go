package tickerdb

import (
	"testing"
	"time"

	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stellar/go/support/db/dbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsertOrUpdateAsset(t *testing.T) {
	db := dbtest.Postgres(t)
	defer db.Close()

	var session TickerSession
	session.DB = db.Open()
	defer session.DB.Close()

	// Run migrations to make sure the tests are run
	// on the most updated schema version
	migrations := &migrate.FileMigrationSource{
		Dir: "./migrations",
	}
	_, err := migrate.Exec(session.DB.DB, "postgres", migrations, migrate.Up)
	require.NoError(t, err)

	firstTime := time.Now()
	a := Asset{
		Code:        "XLM",
		Issuer:      "STELLAR DEVELOPMENT FOUNDATION",
		LastValid:   firstTime,
		LastChecked: firstTime,
	}
	err = session.InsertOrUpdateAsset(&a)
	require.NoError(t, err)

	var dbAsset1 Asset
	err = session.GetRaw(&dbAsset1, `
		SELECT *
		FROM assets
		ORDER BY id DESC
		LIMIT 1`,
	)
	require.NoError(t, err)
	assert.Equal(t, "XLM", dbAsset1.Code)
	assert.Equal(t, "STELLAR DEVELOPMENT FOUNDATION", dbAsset1.Issuer)
	assert.Equal(t, firstTime.Round(0).Local(), dbAsset1.LastValid.Local())
	assert.Equal(t, firstTime.Round(0).Local(), dbAsset1.LastChecked.Local())

	secondTime := time.Now()
	a.LastValid = secondTime
	a.LastChecked = secondTime
	err = session.InsertOrUpdateAsset(&a)
	require.NoError(t, err)

	var dbAsset2 Asset
	err = session.GetRaw(&dbAsset2, `
		SELECT *
		FROM assets
		ORDER BY id DESC
		LIMIT 1`,
	)
	require.NoError(t, err)
	assert.Equal(t, dbAsset1.ID, dbAsset2.ID)
	assert.Equal(t, "XLM", dbAsset1.Code)
	assert.Equal(t, "STELLAR DEVELOPMENT FOUNDATION", dbAsset1.Issuer)
	assert.NotEqual(t, firstTime.Round(0).Local(), dbAsset2.LastValid.Local())
	assert.NotEqual(t, firstTime.Round(0).Local(), dbAsset2.LastChecked.Local())
	assert.Equal(t, secondTime.Round(0).Local(), dbAsset2.LastValid.Local())
	assert.Equal(t, secondTime.Round(0).Local(), dbAsset2.LastChecked.Local())
}
