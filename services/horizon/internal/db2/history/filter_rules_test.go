package history

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stretchr/testify/assert"
)

func TestGetAllFilterConfigs(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	results, err := q.GetAllFilters(tt.Ctx)
	assert.NoError(t, err)
	tt.Assert.Len(results, 2)

	filters := []FilterConfig{
		{Name: "asset", LastModified: 0, Rules: "{}", Enabled: false},
		{Name: "account", LastModified: 0, Rules: "{}", Enabled: false},
	}

	tt.Assert.ElementsMatchf(results, filters, "filter get all list doesn not match")
}

func TestUpdateExistingFilterConfig(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	fc1Result, err := q.GetFilterByName(tt.Ctx, "asset")
	assert.NoError(t, err)
	tt.Assert.Equal(fc1Result.Enabled, false)
	tt.Assert.Equal(fc1Result.Rules, "{}")

	fc1Result.Enabled = true
	fc1Result.Rules = `{"abc": "123"}`
	err = q.UpdateFilterConfig(tt.Ctx, fc1Result)
	assert.NoError(t, err)
	fc1Result, err = q.GetFilterByName(tt.Ctx, "asset")
	assert.NoError(t, err)
	tt.Assert.Equal(fc1Result.Name, "asset")
	tt.Assert.Equal(fc1Result.Enabled, true)
	tt.Assert.Equal(fc1Result.Rules, `{"abc": "123"}`)
}

func TestUpdateNonExistingFilterConfig(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	fc1 := FilterConfig{}
	fc1.Enabled = true
	fc1.Name = "notfound"
	fc1.Rules = `{"abc": "123"}`
	err := q.UpdateFilterConfig(tt.Ctx, fc1)
	assert.Error(t, err)
}
