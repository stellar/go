package db

import (
	"context"
	"testing"

	"github.com/stellar/go/support/db/dbtest"
	"github.com/stretchr/testify/assert"
)

type person struct {
	Name        string `db:"name"`
	HungerLevel string `db:"hunger_level"`

	SomethingIgnored int `db:"-"`
}

func TestGetTable(t *testing.T) {
	db := dbtest.Postgres(t).Load(testSchema)
	defer db.Close()
	sess := &Session{DB: db.Open(), Ctx: context.Background()}
	defer sess.DB.Close()

	tbl := sess.GetTable("users")
	if assert.NotNil(t, tbl) {
		assert.Equal(t, "users", tbl.Name)
		assert.Equal(t, sess, tbl.Session)
	}

}
