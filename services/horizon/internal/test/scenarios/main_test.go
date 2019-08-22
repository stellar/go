package scenarios

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/shurcooL/httpfs/filter"
	supportHttp "github.com/stellar/go/support/http"
)

func TestGeneratedAssets(t *testing.T) {
	var localAssets http.FileSystem = filter.Keep(http.Dir("assets"), func(path string, fi os.FileInfo) bool {
		return fi.IsDir() || strings.HasSuffix(path, ".sql")
	})
	if !supportHttp.EqualFileSystems(localAssets, assets, "/") {
		t.Fatalf("generated migrations does not match local migrations")
	}
}
