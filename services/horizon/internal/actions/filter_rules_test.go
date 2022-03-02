package actions

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/filters"
	"github.com/stellar/go/services/horizon/internal/test"
)

func TestGetFilterConfigNotFound(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{SessionInterface: tt.HorizonSession()}
	handler := &FilterRuleHandler{}
	recorder := httptest.NewRecorder()
	handler.Get(
		recorder,
		makeRequest(
			t,
			map[string]string{},
			map[string]string{"filter_name": "xyz"},
			q,
		),
	)

	resp := recorder.Result()
	tt.Assert.Equal(http.StatusNotFound, resp.StatusCode)
}

func TestGetFilterConfigOneResult(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{SessionInterface: tt.HorizonSession()}

	// put some more values into the config for resource validation after retrieval
	fc1 := history.FilterConfig{
		Rules:   `{"whitelist": ["1","2","3"]}`,
		Name:    filters.FilterAssetFilterName,
		Enabled: true,
	}

	q.UpdateFilterConfig(tt.Ctx, fc1)

	handler := &FilterRuleHandler{}
	recorder := httptest.NewRecorder()
	handler.Get(
		recorder,
		makeRequest(
			t,
			map[string]string{},
			map[string]string{"filter_name": filters.FilterAssetFilterName},
			q,
		),
	)

	resp := recorder.Result()
	tt.Assert.Equal(http.StatusOK, resp.StatusCode)

	raw, err := ioutil.ReadAll(resp.Body)
	tt.Assert.NoError(err)

	var filterCfgResource filterResource
	json.Unmarshal(raw, &filterCfgResource)
	tt.Assert.NoError(err)

	tt.Assert.Equal(filterCfgResource.Name, filters.FilterAssetFilterName)
	tt.Assert.Equal(len(filterCfgResource.Rules["whitelist"].([]interface{})), 3)
	tt.Assert.Equal(filterCfgResource.Rules["whitelist"].([]interface{})[0], "1")
	tt.Assert.Equal(filterCfgResource.Rules["whitelist"].([]interface{})[1], "2")
	tt.Assert.Equal(filterCfgResource.Rules["whitelist"].([]interface{})[2], "3")
	tt.Assert.Equal(filterCfgResource.Enabled, true)
	tt.Assert.True(filterCfgResource.LastModified > 0)
}

func TestGetFilterConfigListResult(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{SessionInterface: tt.HorizonSession()}

	handler := &FilterRuleHandler{}
	recorder := httptest.NewRecorder()
	handler.Get(
		recorder,
		makeRequest(
			t,
			map[string]string{},
			map[string]string{},
			q,
		),
	)

	resp := recorder.Result()
	tt.Assert.Equal(http.StatusOK, resp.StatusCode)

	raw, err := ioutil.ReadAll(resp.Body)
	tt.Assert.NoError(err)

	var filterCfgResourceResponse []filterResource
	json.Unmarshal(raw, &filterCfgResourceResponse)
	tt.Assert.NoError(err)

	// these are from the pre-defined default config rows seeded/created by scheam migrations file
	filterCfgResourceList := []filterResource{
		{Name: "asset", LastModified: 0, Rules: map[string]interface{}{}, Enabled: false},
		{Name: "account", LastModified: 0, Rules: map[string]interface{}{}, Enabled: false},
	}

	tt.Assert.Len(filterCfgResourceResponse, 2)
	tt.Assert.ElementsMatchf(filterCfgResourceList, filterCfgResourceResponse, "filter resource list does not match")
}

func TestMalFormedUpdateFilterConfig(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{SessionInterface: tt.HorizonSession()}

	handler := &FilterRuleHandler{}
	recorder := httptest.NewRecorder()
	request := makeRequest(
		t,
		map[string]string{},
		map[string]string{"filter_name": "asset"},
		q,
	)

	request.Body = ioutil.NopCloser(strings.NewReader(`
	    {
			"rules": {
			    "whitelist": ["4","5","6"]
			},
			"enabled": true,
			"name": "unsupported"
		}
		`))

	handler.Update(
		recorder,
		request,
	)

	resp := recorder.Result()
	// can't update a filter with a name that doens't match up to an existing implemented filter
	tt.Assert.Equal(http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateUnsupportedFilterConfig(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{SessionInterface: tt.HorizonSession()}

	handler := &FilterRuleHandler{}
	recorder := httptest.NewRecorder()
	request := makeRequest(
		t,
		map[string]string{},
		map[string]string{"filter_name": "unsupported"},
		q,
	)

	request.Body = ioutil.NopCloser(strings.NewReader(`
	    {
			"rules": {
			    "whitelist": ["4","5","6"]
			},
			"enabled": true,
			"name": "unsupported"
		}
		`))

	handler.Update(
		recorder,
		request,
	)

	resp := recorder.Result()
	// can't update a filter with a name that doens't match up to an existing implemented filter
	tt.Assert.Equal(http.StatusNotFound, resp.StatusCode)
}

func TestUpdateFilterConfig(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{SessionInterface: tt.HorizonSession()}

	handler := &FilterRuleHandler{}
	recorder := httptest.NewRecorder()
	request := makeRequest(
		t,
		map[string]string{},
		map[string]string{"filter_name": filters.FilterAssetFilterName},
		q,
	)

	request.Body = ioutil.NopCloser(strings.NewReader(`
	    {
			"rules": {
			    "whitelist": ["4","5","6"]
			},
			"enabled": true,
			"name": "` + filters.FilterAssetFilterName + `"
		}`))

	handler.Update(
		recorder,
		request,
	)

	resp := recorder.Result()
	tt.Assert.Equal(http.StatusOK, resp.StatusCode)

	fcUpdated, err := q.GetFilterByName(tt.Ctx, filters.FilterAssetFilterName)
	tt.Assert.NoError(err)

	tt.Assert.Equal(fcUpdated.Name, filters.FilterAssetFilterName)
	tt.Assert.Equal(fcUpdated.Enabled, true)
	tt.Assert.True(fcUpdated.LastModified > 0)

	var filterRules map[string]interface{}
	err = json.Unmarshal([]byte(fcUpdated.Rules), &filterRules)
	tt.Assert.NoError(err)
	tt.Assert.Len(filterRules["whitelist"].([]interface{}), 3)
	tt.Assert.Equal(filterRules["whitelist"].([]interface{})[0], "4")
	tt.Assert.Equal(filterRules["whitelist"].([]interface{})[1], "5")
	tt.Assert.Equal(filterRules["whitelist"].([]interface{})[2], "6")
}
