package schema

import (
	"net/http"
	"os"
	"strings"
	"testing"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/shurcooL/httpfs/filter"
	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/db/dbtest"
	supportHttp "github.com/stellar/go/support/http"
)

func TestInit(t *testing.T) {
	tdb := dbtest.Postgres(t)
	defer tdb.Close()
	sess := &db.Session{DB: tdb.Open()}

	defer sess.DB.Close()

	err := Init(sess)

	assert.NoError(t, err)
}

func TestGeneratedAssets(t *testing.T) {
	localAssets := filter.Keep(http.Dir("."), func(path string, fi os.FileInfo) bool {
		return fi.IsDir() || strings.HasSuffix(path, ".sql")
	})
	generatedAssets := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	if !supportHttp.EqualFileSystems(localAssets, generatedAssets, "/") {
		t.Fatalf("generated migrations does not match local migrations")
	}
}
