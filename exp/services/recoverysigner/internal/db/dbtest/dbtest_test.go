package dbtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	db := Open(t)
	session := db.Open()

	result := struct {
		Count int `db:"count"`
	}{}
	err := session.Get(&result, `SELECT COUNT(*) FROM gorp_migrations`)
	require.NoError(t, err)
	assert.Greater(t, result.Count, 0)
}
