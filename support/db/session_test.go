package db

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stellar/go/support/db/dbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrentQueriesTransaction(t *testing.T) {
	db := dbtest.Postgres(t).Load(testSchema)
	defer db.Close()

	sess := &Session{
		Ctx: context.Background(),
		DB:  db.Open(),
		// This test would fail for `Synchronized: false`
		Synchronized: true,
	}
	defer sess.DB.Close()

	err := sess.Begin()
	assert.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			istr := fmt.Sprintf("%d", i)
			var err2 error
			if i%3 == 0 {
				var names []string
				err2 = sess.SelectRaw(&names, "SELECT name FROM people")
			} else if i%3 == 1 {
				var name string
				err2 = sess.GetRaw(&name, "SELECT name FROM people LIMIT 1")
			} else {
				_, err2 = sess.ExecRaw(
					"INSERT INTO people (name, hunger_level) VALUES ('bartek" + istr + "', " + istr + ")",
				)
			}
			assert.NoError(t, err2)
			wg.Done()
		}(i)
	}

	wg.Wait()
	err = sess.Rollback()
	assert.NoError(t, err)
}

func TestSession(t *testing.T) {
	db := dbtest.Postgres(t).Load(testSchema)
	defer db.Close()

	assert := assert.New(t)
	require := require.New(t)
	sess := &Session{DB: db.Open(), Ctx: context.Background()}
	defer sess.DB.Close()

	assert.Equal("postgres", sess.Dialect())

	var count int
	err := sess.GetRaw(&count, "SELECT COUNT(*) FROM people")
	assert.NoError(err)
	assert.Equal(3, count)

	var names []string
	err = sess.SelectRaw(&names, "SELECT name FROM people")
	assert.NoError(err)
	assert.Len(names, 3)

	ret, err := sess.ExecRaw("DELETE FROM people")
	assert.NoError(err)
	deleted, err := ret.RowsAffected()
	assert.NoError(err)
	assert.Equal(int64(3), deleted)

	// Test args (NOTE: there is a simple escaped arg to ensure no error is raised
	// during execution)
	db.Load(testSchema)
	var name string
	err = sess.GetRaw(
		&name,
		"SELECT name FROM people WHERE hunger_level = ? AND name != '??'",
		1000000,
	)
	assert.NoError(err)
	assert.Equal("scott", name)

	// Test NoRows
	err = sess.GetRaw(
		&name,
		"SELECT name FROM people WHERE hunger_level = ?",
		1234,
	)
	assert.True(sess.NoRows(err))

	// Test transactions
	db.Load(testSchema)
	require.NoError(sess.Begin(), "begin failed")
	err = sess.GetRaw(&count, "SELECT COUNT(*) FROM people")
	assert.NoError(err)
	assert.Equal(3, count)
	_, err = sess.ExecRaw("DELETE FROM people")
	assert.NoError(err)
	err = sess.GetRaw(&count, "SELECT COUNT(*) FROM people")
	assert.NoError(err)
	assert.Equal(0, count, "people did not appear deleted inside transaction")
	assert.NoError(sess.Rollback(), "rollback failed")

	// Ensure commit works
	require.NoError(sess.Begin(), "begin failed")
	sess.ExecRaw("DELETE FROM people")
	assert.NoError(sess.Commit(), "commit failed")
	err = sess.GetRaw(&count, "SELECT COUNT(*) FROM people")
	assert.NoError(err)
	assert.Equal(0, count)

	// ensure that selecting into a populated slice clears the slice first
	db.Load(testSchema)
	require.Len(names, 3, "ids slice was not preloaded with data")
	err = sess.SelectRaw(&names, "SELECT name FROM people limit 2")
	assert.NoError(err)
	assert.Len(names, 2)

	// Test ReplacePlaceholders
	out, err := sess.ReplacePlaceholders("? = ? = ? = ??")
	if assert.NoError(err) {
		assert.Equal("$1 = $2 = $3 = ?", out)
	}
}
