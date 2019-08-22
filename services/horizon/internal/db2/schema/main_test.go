package schema

import (
	"net/http"
	"testing"

	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/db/dbtest"
	supportHttp "github.com/stellar/go/support/http"
	"github.com/stretchr/testify/assert"
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
	var localAssets http.FileSystem = http.Dir("assets")
	if !supportHttp.EqualFileSystems(localAssets, assets, "/") {
		t.Fatalf("generated migrations does not match local migrations")
	}
}
