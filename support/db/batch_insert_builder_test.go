package db

import (
	"context"
	"testing"

	"github.com/stellar/go/support/db/dbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type hungerRow struct {
	Name        string `db:"name"`
	HungerLevel string `db:"hunger_level"`
}

type invalidHungerRow struct {
	Name        string `db:"name"`
	HungerLevel string `db:"hunger_level"`
	LastName    string `db:"last_name"`
}

func TestBatchInsertBuilder(t *testing.T) {
	db := dbtest.Postgres(t).Load(testSchema)
	defer db.Close()
	sess := &Session{DB: db.Open(), Ctx: context.Background()}
	defer sess.DB.Close()

	insertBuilder := &BatchInsertBuilder{
		Table: sess.GetTable("people"),
	}

	// exec on the empty set should produce no errors
	assert.NoError(t, insertBuilder.Exec())

	var err error

	err = insertBuilder.Row(map[string]interface{}{
		"name":         "bubba",
		"hunger_level": "120",
	})
	assert.NoError(t, err)

	err = insertBuilder.RowStruct(hungerRow{
		Name:        "bubba2",
		HungerLevel: "1202",
	})
	assert.NoError(t, err)

	// Extra column
	err = insertBuilder.Row(map[string]interface{}{
		"name":         "bubba",
		"hunger_level": "120",
		"abc":          "def",
	})
	assert.EqualError(t, err, "invalid number of columns (expected=2, actual=3)")

	// Not enough columns
	err = insertBuilder.Row(map[string]interface{}{
		"name": "bubba",
	})
	assert.EqualError(t, err, "invalid number of columns (expected=2, actual=1)")

	// Invalid column
	err = insertBuilder.Row(map[string]interface{}{
		"name":  "bubba",
		"hello": "120",
	})
	assert.EqualError(t, err, `column "hunger_level" does not exist`)

	err = insertBuilder.RowStruct(invalidHungerRow{
		Name:        "Max",
		HungerLevel: "500",
	})
	assert.EqualError(t, err, `expected value of type "db.hungerRow" but got "db.invalidHungerRow" value`)

	err = insertBuilder.Exec()
	assert.NoError(t, err)

	// Check rows
	var found []person
	err = sess.SelectRaw(&found, `SELECT * FROM people WHERE name like 'bubba%'`)

	require.NoError(t, err)
	assert.Equal(
		t,
		found,
		[]person{
			person{Name: "bubba", HungerLevel: "120"},
			person{Name: "bubba2", HungerLevel: "1202"},
		},
	)

	err = insertBuilder.Row(map[string]interface{}{
		"name":         "bubba",
		"hunger_level": "1",
	})
	assert.NoError(t, err)

	err = insertBuilder.Exec()
	assert.EqualError(
		t, err, "error adding values while inserting to people: exec failed: pq:"+
			" duplicate key value violates unique constraint \"people_pkey\"",
	)

	insertBuilder.Suffix = "ON CONFLICT (name) DO NOTHING"

	err = insertBuilder.Row(map[string]interface{}{
		"name":         "bubba",
		"hunger_level": "1",
	})
	assert.NoError(t, err)

	err = insertBuilder.Exec()
	assert.NoError(t, err)

	err = sess.SelectRaw(&found, `SELECT * FROM people WHERE name like 'bubba%'`)

	require.NoError(t, err)
	assert.Equal(
		t,
		found,
		[]person{
			person{Name: "bubba", HungerLevel: "120"},
			person{Name: "bubba2", HungerLevel: "1202"},
		},
	)

	insertBuilder.Suffix = "ON CONFLICT (name) DO UPDATE SET hunger_level = EXCLUDED.hunger_level"

	err = insertBuilder.Row(map[string]interface{}{
		"name":         "bubba",
		"hunger_level": "1",
	})
	assert.NoError(t, err)

	err = insertBuilder.Exec()
	assert.NoError(t, err)

	err = sess.SelectRaw(&found, `SELECT * FROM people WHERE name like 'bubba%'`)

	require.NoError(t, err)
	assert.Equal(
		t,
		found,
		[]person{
			person{Name: "bubba2", HungerLevel: "1202"},
			person{Name: "bubba", HungerLevel: "1"},
		},
	)
}
