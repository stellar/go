package tickerdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type exampleDBModel struct {
	ID      int    `db:"id"`
	Name    string `db:"name"`
	Counter int    `db:"counter"`
}

func TestGetDBFieldTags(t *testing.T) {
	m := exampleDBModel{
		ID:      10,
		Name:    "John Doe",
		Counter: 15,
	}

	fieldTags := getDBFieldTags(m, true)
	assert.Contains(t, fieldTags, "\"name\"")
	assert.Contains(t, fieldTags, "\"counter\"")
	assert.NotContains(t, fieldTags, "\"id\"")
	assert.Equal(t, 2, len(fieldTags))

	fieldTagsWithID := getDBFieldTags(m, false)
	assert.Contains(t, fieldTagsWithID, "\"name\"")
	assert.Contains(t, fieldTagsWithID, "\"counter\"")
	assert.Contains(t, fieldTagsWithID, "\"id\"")
	assert.Equal(t, 3, len(fieldTagsWithID))
}

func TestGetDBFieldValues(t *testing.T) {
	m := exampleDBModel{
		ID:      10,
		Name:    "John Doe",
		Counter: 15,
	}

	fieldValues := getDBFieldValues(m, true)
	assert.Contains(t, fieldValues, 15)
	assert.Contains(t, fieldValues, "John Doe")
	assert.NotContains(t, fieldValues, 10)
	assert.Equal(t, 2, len(fieldValues))

	fieldTagsWithID := getDBFieldValues(m, false)
	assert.Contains(t, fieldTagsWithID, 15)
	assert.Contains(t, fieldTagsWithID, "John Doe")
	assert.Contains(t, fieldTagsWithID, 10)
	assert.Equal(t, 3, len(fieldTagsWithID))
}

func TestGeneratePlaceholders(t *testing.T) {
	var p []interface{}
	p = append(p, 1)
	p = append(p, 2)
	p = append(p, 3)
	placeholder := generatePlaceholders(p)
	assert.Equal(t, "?, ?, ?", placeholder)
}
