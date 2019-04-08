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

	publicKey := "ASOKDASDKMAKSD19023ASDSAD0912309"
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
		Code:        code,
		IssuerID:    dbIssuer.ID,
		LastValid:   firstTime,
		LastChecked: firstTime,
	}
	err = session.InsertOrUpdateAsset(&a, []string{"code", "issuer_id"})
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
	assert.Equal(t, dbIssuer.ID, dbAsset1.IssuerID)
	assert.Equal(t, firstTime.Round(0).Local(), dbAsset1.LastValid.Local())
	assert.Equal(t, firstTime.Round(0).Local(), dbAsset1.LastChecked.Local())

	// Creating Seconde Asset:
	secondTime := time.Now()
	a.LastValid = secondTime
	a.LastChecked = secondTime
	err = session.InsertOrUpdateAsset(&a, []string{"code", "issuer_id"})
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
	assert.Equal(t, dbIssuer.ID, dbAsset2.IssuerID)
	assert.NotEqual(t, firstTime.Round(0).Local(), dbAsset2.LastValid.Local())
	assert.NotEqual(t, firstTime.Round(0).Local(), dbAsset2.LastChecked.Local())
	assert.Equal(t, secondTime.Round(0).Local(), dbAsset2.LastValid.Local())
	assert.Equal(t, secondTime.Round(0).Local(), dbAsset2.LastChecked.Local())

	// Creating Third Asset:
	thirdTime := time.Now()
	a.LastValid = thirdTime
	a.LastChecked = thirdTime
	err = session.InsertOrUpdateAsset(&a, []string{"code", "issuer_id", "last_valid", "last_checked"})
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
	assert.Equal(t, dbIssuer.ID, dbAsset3.IssuerID)
	assert.NotEqual(t, thirdTime.Round(0).Local(), dbAsset3.LastValid.Local())
	assert.NotEqual(t, thirdTime.Round(0).Local(), dbAsset3.LastChecked.Local())
	assert.Equal(t, dbAsset2.LastValid.Local(), dbAsset3.LastValid.Local())
	assert.Equal(t, dbAsset2.LastValid.Local(), dbAsset3.LastChecked.Local())
}
