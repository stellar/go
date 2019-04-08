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
		PublicKey:   "STELLAR DEVELOPMENT FOUNDATION",
		LastValid:   firstTime,
		LastChecked: firstTime,
	}
	err = session.InsertOrUpdateAsset(&a, []string{"code", "public_key"})
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
	assert.Equal(t, "STELLAR DEVELOPMENT FOUNDATION", dbAsset1.PublicKey)
	assert.Equal(
		t,
		firstTime.Local().Truncate(time.Millisecond),
		dbAsset1.LastValid.Local().Truncate(time.Millisecond),
	)
	assert.Equal(
		t,
		firstTime.Local().Truncate(time.Millisecond),
		dbAsset1.LastChecked.Local().Truncate(time.Millisecond),
	)

	secondTime := time.Now()
	a.LastValid = secondTime
	a.LastChecked = secondTime
	err = session.InsertOrUpdateAsset(&a, []string{"code", "public_key"})
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
	assert.Equal(t, "XLM", dbAsset2.Code)
	assert.Equal(t, "STELLAR DEVELOPMENT FOUNDATION", dbAsset2.PublicKey)
	assert.NotEqual(
		t,
		firstTime.Local().Truncate(time.Millisecond),
		dbAsset2.LastValid.Local().Truncate(time.Millisecond),
	)
	assert.NotEqual(t,
		firstTime.Local().Truncate(time.Millisecond),
		dbAsset2.LastChecked.Local().Truncate(time.Millisecond),
	)
	assert.Equal(
		t,
		secondTime.Local().Truncate(time.Millisecond),
		dbAsset2.LastValid.Local().Truncate(time.Millisecond),
	)
	assert.Equal(
		t,
		secondTime.Local().Truncate(time.Millisecond),
		dbAsset2.LastChecked.Local().Truncate(time.Millisecond),
	)

	thirdTime := time.Now()
	a.LastValid = thirdTime
	a.LastChecked = thirdTime
	err = session.InsertOrUpdateAsset(&a, []string{"code", "public_key", "last_valid", "last_checked"})
	require.NoError(t, err)
	var dbAsset3 Asset
	err = session.GetRaw(&dbAsset3, `
		SELECT *
		FROM assets
		ORDER BY id DESC
		LIMIT 1`,
	)
	require.NoError(t, err)
	assert.Equal(t, dbAsset2.ID, dbAsset3.ID)
	assert.Equal(t, "XLM", dbAsset3.Code)
	assert.Equal(t, "STELLAR DEVELOPMENT FOUNDATION", dbAsset3.PublicKey)
	assert.NotEqual(
		t,
		thirdTime.Local().Truncate(time.Millisecond),
		dbAsset3.LastValid.Local().Truncate(time.Millisecond),
	)
	assert.NotEqual(
		t,
		thirdTime.Local().Truncate(time.Millisecond),
		dbAsset3.LastChecked.Local().Truncate(time.Millisecond),
	)
	assert.Equal(
		t,
		dbAsset2.LastValid.Local().Truncate(time.Millisecond),
		dbAsset3.LastValid.Local().Truncate(time.Millisecond),
	)
	assert.Equal(
		t, dbAsset2.LastValid.Local().Truncate(time.Millisecond),
		dbAsset3.LastChecked.Local().Truncate(time.Millisecond),
	)
}
