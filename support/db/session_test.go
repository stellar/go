package db

import (
	"context"
	"testing"

	"github.com/stellar/go/support/db/dbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSession(t *testing.T) {
	db := dbtest.Postgres(t).Load(testSchema)
	defer db.Close()

	ctx := context.Background()
	assert := assert.New(t)
	require := require.New(t)
	sess := &Session{DB: db.Open()}
	defer sess.DB.Close()

	assert.Equal("postgres", sess.Dialect())

	var count int
	err := sess.GetRaw(ctx, &count, "SELECT COUNT(*) FROM people")
	assert.NoError(err)
	assert.Equal(3, count)

	var names []string
	err = sess.SelectRaw(ctx, &names, "SELECT name FROM people")
	assert.NoError(err)
	assert.Len(names, 3)

	ret, err := sess.ExecRaw(ctx, "DELETE FROM people")
	assert.NoError(err)
	deleted, err := ret.RowsAffected()
	assert.NoError(err)
	assert.Equal(int64(3), deleted)

	// Test args (NOTE: there is a simple escaped arg to ensure no error is raised
	// during execution)
	db.Load(testSchema)
	var name string
	err = sess.GetRaw(ctx,
		&name,
		"SELECT name FROM people WHERE hunger_level = ? AND name != '??'",
		1000000,
	)
	assert.NoError(err)
	assert.Equal("scott", name)

	// Test NoRows
	err = sess.GetRaw(ctx,
		&name,
		"SELECT name FROM people WHERE hunger_level = ?",
		1234,
	)
	assert.True(sess.NoRows(err))

	// Test transactions
	db.Load(testSchema)
	require.NoError(sess.Begin(ctx), "begin failed")
	err = sess.GetRaw(ctx, &count, "SELECT COUNT(*) FROM people")
	assert.NoError(err)
	assert.Equal(3, count)
	_, err = sess.ExecRaw(ctx, "DELETE FROM people")
	assert.NoError(err)
	err = sess.GetRaw(ctx, &count, "SELECT COUNT(*) FROM people")
	assert.NoError(err)
	assert.Equal(0, count, "people did not appear deleted inside transaction")
	assert.NoError(sess.Rollback(ctx), "rollback failed")

	// Ensure commit works
	require.NoError(sess.Begin(ctx), "begin failed")
	sess.ExecRaw(ctx, "DELETE FROM people")
	assert.NoError(sess.Commit(ctx), "commit failed")
	err = sess.GetRaw(ctx, &count, "SELECT COUNT(*) FROM people")
	assert.NoError(err)
	assert.Equal(0, count)

	// ensure that selecting into a populated slice clears the slice first
	db.Load(testSchema)
	require.Len(names, 3, "ids slice was not preloaded with data")
	err = sess.SelectRaw(ctx, &names, "SELECT name FROM people limit 2")
	assert.NoError(err)
	assert.Len(names, 2)

	// Test ReplacePlaceholders
	out, err := sess.ReplacePlaceholders("? = ? = ? = ??")
	if assert.NoError(err) {
		assert.Equal("$1 = $2 = $3 = ?", out)
	}
}
