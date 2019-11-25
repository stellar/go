package db

import (
	"context"
	"testing"

	"github.com/stellar/go/support/db/dbtest"
	"github.com/stretchr/testify/assert"
)

func TestGetBuilder_Exec(t *testing.T) {
	db := dbtest.Postgres(t).Load(testSchema)
	defer db.Close()
	sess := &Session{DB: db.Open(), Ctx: context.Background()}
	defer sess.DB.Close()

	var found person

	tbl := sess.GetTable("people")
	err := tbl.Get(&found, "name = ?", "scott").Exec()

	if assert.NoError(t, err, "query error") {
		assert.Equal(t, "scott", found.Name)
		assert.Equal(t, "1000000", found.HungerLevel)
	}
}
