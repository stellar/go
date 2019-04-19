package tickerdb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stellar/go/support/db/dbtest"
	"github.com/stretchr/testify/require"
)

func TestBulkInsertTrades(t *testing.T) {
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

	// Adding a seed issuer to be used later:
	tbl := session.GetTable("issuers")
	_, err = tbl.Insert(Issuer{
		PublicKey: "GCF3TQXKZJNFJK7HCMNE2O2CUNKCJH2Y2ROISTBPLC7C5EIA5NNG2XZB",
		Name:      "FOO BAR",
	}).IgnoreCols("id").Exec()
	require.NoError(t, err)
	var issuer Issuer
	err = session.GetRaw(&issuer, `
		SELECT *
		FROM issuers
		ORDER BY id DESC
		LIMIT 1`,
	)
	require.NoError(t, err)

	// Adding a seed asset to be used later:
	err = session.InsertOrUpdateAsset(&Asset{
		Code:     "XLM",
		IssuerID: issuer.ID,
	}, []string{"code", "issuer_id"})
	require.NoError(t, err)
	var asset1 Asset
	err = session.GetRaw(&asset1, `
		SELECT *
		FROM assets
		ORDER BY id DESC
		LIMIT 1`,
	)
	require.NoError(t, err)

	// Adding another asset to be used later:
	err = session.InsertOrUpdateAsset(&Asset{
		Code:     "BTC",
		IssuerID: issuer.ID,
	}, []string{"code", "issuer_id"})
	require.NoError(t, err)
	var asset2 Asset
	err = session.GetRaw(&asset2, `
		SELECT *
		FROM assets
		ORDER BY id DESC
		LIMIT 1`,
	)
	require.NoError(t, err)

	// Verify that we actually have two assets:
	assert.NotEqual(t, asset1.ID, asset2.ID)

	// Now let's create the trades:
	trades := []Trade{
		Trade{
			HorizonID:       "hrzid1",
			BaseAssetID:     asset1.ID,
			CounterAssetID:  asset2.ID,
			LedgerCloseTime: time.Now(),
		},
		Trade{
			HorizonID:       "hrzid2",
			BaseAssetID:     asset2.ID,
			CounterAssetID:  asset1.ID,
			LedgerCloseTime: time.Now(),
		},
	}
	err = session.BulkInsertTrades(trades)
	require.NoError(t, err)

	// Ensure only two were created:
	rows, err := session.QueryRaw("SELECT * FROM trades")
	require.NoError(t, err)
	rowsCount := 0
	for rows.Next() {
		rowsCount++
	}
	assert.Equal(t, 2, rowsCount)

	// Re-insert the same trades and check if count remains = 2:
	err = session.BulkInsertTrades(trades)
	require.NoError(t, err)

	rows, err = session.QueryRaw("SELECT * FROM trades")
	require.NoError(t, err)
	rowsCount2 := 0
	for rows.Next() {
		rowsCount2++
	}
	assert.Equal(t, 2, rowsCount2)
}

func TestGetLastTrade(t *testing.T) {
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

	// Sanity Check (there are no trades in the database)
	_, err = session.GetLastTrade()
	require.Error(t, err)

	// Adding a seed issuer to be used later:
	tbl := session.GetTable("issuers")
	_, err = tbl.Insert(Issuer{
		PublicKey: "GCF3TQXKZJNFJK7HCMNE2O2CUNKCJH2Y2ROISTBPLC7C5EIA5NNG2XZB",
		Name:      "FOO BAR",
	}).IgnoreCols("id").Exec()
	require.NoError(t, err)
	var issuer Issuer
	err = session.GetRaw(&issuer, `
		SELECT *
		FROM issuers
		ORDER BY id DESC
		LIMIT 1`,
	)
	require.NoError(t, err)

	// Adding a seed asset to be used later:
	err = session.InsertOrUpdateAsset(&Asset{
		Code:     "XLM",
		IssuerID: issuer.ID,
	}, []string{"code", "issuer_id"})
	require.NoError(t, err)
	var asset1 Asset
	err = session.GetRaw(&asset1, `
		SELECT *
		FROM assets
		ORDER BY id DESC
		LIMIT 1`,
	)
	require.NoError(t, err)

	// Adding another asset to be used later:
	err = session.InsertOrUpdateAsset(&Asset{
		Code:     "BTC",
		IssuerID: issuer.ID,
	}, []string{"code", "issuer_id"})
	require.NoError(t, err)
	var asset2 Asset
	err = session.GetRaw(&asset2, `
		SELECT *
		FROM assets
		ORDER BY id DESC
		LIMIT 1`,
	)
	require.NoError(t, err)

	// Verify that we actually have two assets:
	assert.NotEqual(t, asset1.ID, asset2.ID)

	now := time.Now()
	oneYearBefore := now.AddDate(-1, 0, 0)

	// Now let's create the trades:
	trades := []Trade{
		Trade{
			HorizonID:       "hrzid2",
			BaseAssetID:     asset2.ID,
			CounterAssetID:  asset1.ID,
			LedgerCloseTime: oneYearBefore,
		},
		Trade{
			HorizonID:       "hrzid1",
			BaseAssetID:     asset1.ID,
			CounterAssetID:  asset2.ID,
			LedgerCloseTime: now,
		},
		Trade{
			HorizonID:       "hrzid2",
			BaseAssetID:     asset2.ID,
			CounterAssetID:  asset1.ID,
			LedgerCloseTime: oneYearBefore,
		},
	}

	// Re-insert the same trades and check if count remains = 2:
	err = session.BulkInsertTrades(trades)
	require.NoError(t, err)

	lastTrade, err := session.GetLastTrade()
	require.NoError(t, err)
	assert.Equal(
		t,
		now.Local().Truncate(time.Millisecond),
		lastTrade.LedgerCloseTime.Local().Truncate(time.Millisecond),
	)
}
