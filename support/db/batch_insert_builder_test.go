package db

import (
	"testing"

	"github.com/stellar/go/support/db/dbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchInsertBuilder(t *testing.T) {
	db := dbtest.Postgres(t).Load(testSchema)
	defer db.Close()
	sess := &Session{DB: db.Open()}
	defer sess.DB.Close()

	insertBuilder := &BatchInsertBuilder{
		Table: sess.GetTable("people"),
	}

	var err error

	err = insertBuilder.Row(map[string]interface{}{
		"name":         "bubba",
		"hunger_level": "120",
	})
	assert.NoError(t, err)

	err = insertBuilder.Row(map[string]interface{}{
		"name":         "bubba2",
		"hunger_level": "1202",
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

	err = insertBuilder.Exec()
	assert.NoError(t, err)

	query, args, err := insertBuilder.sql.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "INSERT INTO people (hunger_level,name) VALUES (?,?),(?,?)", query)
	assert.Equal(t, []interface{}{
		"120", "bubba",
		"1202", "bubba2",
	}, args)

	// Check rows
	var found []person
	err = sess.SelectRaw(&found, `SELECT * FROM people WHERE name like 'bubba%'`)

	require.NoError(t, err)
	if assert.Len(t, found, 2) {
		assert.Equal(t, "bubba", found[0].Name)
		assert.Equal(t, "120", found[0].HungerLevel)

		assert.Equal(t, "bubba2", found[1].Name)
		assert.Equal(t, "1202", found[1].HungerLevel)
	}
}
