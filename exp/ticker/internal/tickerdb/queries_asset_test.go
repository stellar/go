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

	publicKey := "GCF3TQXKZJNFJK7HCMNE2O2CUNKCJH2Y2ROISTBPLC7C5EIA5NNG2XZB"
	issuerAccount := "AM2FQXKZJNFJK7HCMNE2O2CUNKCJH2Y2ROISTBPLC7C5EIA5NNG2XZB"
	name := "FOO BAR"
	code := "XLM"

	// Adding a seed issuer to be used later:
	issuer := Issuer{
		PublicKey: publicKey,
		Name:      name,
	}
	tbl := session.GetTable("issuers")
	_, err = tbl.Insert(issuer).IgnoreCols("id").Exec()
	require.NoError(t, err)
	var dbIssuer Issuer
	err = session.GetRaw(&dbIssuer, `
		SELECT *
		FROM issuers
		ORDER BY id DESC
		LIMIT 1`,
	)
	require.NoError(t, err)

	// Creating first asset:
	firstTime := time.Now()
	a := Asset{
		Code:          code,
		IssuerAccount: issuerAccount,
		IssuerID:      dbIssuer.ID,
		LastValid:     firstTime,
		LastChecked:   firstTime,
	}
	err = session.InsertOrUpdateAsset(&a, []string{"code", "issuer_account", "issuer_id"})
	require.NoError(t, err)

	var dbAsset1 Asset
	err = session.GetRaw(&dbAsset1, `
		SELECT *
		FROM assets
		ORDER BY id DESC
		LIMIT 1`,
	)
	require.NoError(t, err)

	assert.Equal(t, code, dbAsset1.Code)
	assert.Equal(t, issuerAccount, dbAsset1.IssuerAccount)
	assert.Equal(t, dbIssuer.ID, dbAsset1.IssuerID)
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

	// Creating Seconde Asset:
	secondTime := time.Now()
	a.LastValid = secondTime
	a.LastChecked = secondTime
	err = session.InsertOrUpdateAsset(&a, []string{"code", "issuer_account", "issuer_id"})
	require.NoError(t, err)

	var dbAsset2 Asset
	err = session.GetRaw(&dbAsset2, `
		SELECT *
		FROM assets
		ORDER BY id DESC
		LIMIT 1`,
	)
	require.NoError(t, err)

	// Validating if changes match what was expected:
	assert.Equal(t, dbAsset1.ID, dbAsset2.ID)
	assert.Equal(t, code, dbAsset2.Code)
	assert.Equal(t, issuerAccount, dbAsset1.IssuerAccount)
	assert.Equal(t, dbIssuer.ID, dbAsset2.IssuerID)
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

	// Creating Third Asset:
	thirdTime := time.Now()
	a.LastValid = thirdTime
	a.LastChecked = thirdTime
	err = session.InsertOrUpdateAsset(&a, []string{"code", "issuer_id", "last_valid", "last_checked", "issuer_account"})
	require.NoError(t, err)
	var dbAsset3 Asset
	err = session.GetRaw(&dbAsset3, `
		SELECT *
		FROM assets
		ORDER BY id DESC
		LIMIT 1`,
	)
	require.NoError(t, err)

	// Validating if changes match what was expected:
	assert.Equal(t, dbAsset2.ID, dbAsset3.ID)
	assert.Equal(t, code, dbAsset3.Code)
	assert.Equal(t, issuerAccount, dbAsset3.IssuerAccount)
	assert.Equal(t, dbIssuer.ID, dbAsset3.IssuerID)
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
