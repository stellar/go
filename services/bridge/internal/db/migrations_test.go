package db

import (
	"net/http"
	"os"
	"strings"
	"testing"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/shurcooL/httpfs/filter"

	supportHttp "github.com/stellar/go/support/http"
)

func TestGeneratedAssets(t *testing.T) {
	localAssets := filter.Keep(http.Dir("."), func(path string, fi os.FileInfo) bool {
		return fi.IsDir() || strings.HasSuffix(path, ".sql")
	})
	generatedAssets := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	if !supportHttp.EqualFileSystems(localAssets, generatedAssets, "/") {
		t.Fatalf("generated migrations does not match local migrations")
	}
}
