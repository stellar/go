package schema

import (
	"net/http"
	"testing"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/support/db/dbtest"
	supportHttp "github.com/stellar/go/support/http"
)

func TestInit(t *testing.T) {
	tdb := dbtest.Postgres(t)
	defer tdb.Close()
	db := tdb.Open()

	defer db.Close()

	_, err := Migrate(db.DB, MigrateUp, 0)

	assert.NoError(t, err)
}

func TestGeneratedAssets(t *testing.T) {
	generatedAssets := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	if !supportHttp.EqualFileSystems(http.Dir("."), generatedAssets, "migrations") {
		t.Fatalf("generated migrations does not match local migrations")
	}
}
