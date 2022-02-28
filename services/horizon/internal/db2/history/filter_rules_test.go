package history

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stretchr/testify/assert"
)

var (
	fc1 = FilterConfig{
		Rules:   "{}",
		Name:    "test data",
		Enabled: false,
	}
)

func TestInsertFilterConfig(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	err := q.UpsertFilterConfig(tt.Ctx, fc1)
	assert.NoError(t, err)
	fc1Result, err := q.GetFilterByName(tt.Ctx, "test data")
	assert.NoError(t, err)
	tt.Assert.True(fc1Result.LastModified > 0)
	tt.Assert.Equal(fc1Result.Name, "test data")
	tt.Assert.Equal(fc1Result.Enabled, false)
	tt.Assert.Equal(fc1Result.Rules, "{}")
}

func TestGetAllFilterConfigs(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	err := q.UpsertFilterConfig(tt.Ctx, fc1)
	assert.NoError(t, err)
	results, err := q.GetAllFilters(tt.Ctx)
	assert.NoError(t, err)
	tt.Assert.Len(results, 1)

	tt.Assert.Equal(results[0].Name, "test data")
}
func TestRemoveFilterConfig(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	err := q.UpsertFilterConfig(tt.Ctx, fc1)
	assert.NoError(t, err)

	err = q.DeleteFilterByName(tt.Ctx, "not found")
	assert.Error(t, err)

	err = q.DeleteFilterByName(tt.Ctx, "test data")
	assert.NoError(t, err)

	fc1, err = q.GetFilterByName(tt.Ctx, "test data")
	assert.Error(t, err)
}

func TestUpdateExistingFilterConfig(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	err := q.UpsertFilterConfig(tt.Ctx, fc1)
	assert.NoError(t, err)
	fc1Result, err := q.GetFilterByName(tt.Ctx, "test data")
	assert.NoError(t, err)
	tt.Assert.Equal(fc1Result.Enabled, false)
	tt.Assert.Equal(fc1Result.Rules, "{}")

	fc1Result.Enabled = true
	fc1Result.Rules = `{"abc": "123"}`
	err = q.UpsertFilterConfig(tt.Ctx, fc1Result)
	assert.NoError(t, err)
	fc1Result, err = q.GetFilterByName(tt.Ctx, "test data")
	assert.NoError(t, err)
	tt.Assert.Equal(fc1Result.Name, "test data")
	tt.Assert.Equal(fc1Result.Enabled, true)
	tt.Assert.Equal(fc1Result.Rules, `{"abc": "123"}`)
}
