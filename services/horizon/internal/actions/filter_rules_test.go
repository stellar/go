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

	fc1 := history.FilterConfig{
		Rules:   `{"whitelist": ["1","2","3"]}`,
		Name:    "xyz",
		Enabled: false,
	}

	q.UpsertFilterConfig(tt.Ctx, fc1)

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
	tt.Assert.Equal(http.StatusOK, resp.StatusCode)

	raw, err := ioutil.ReadAll(resp.Body)
	tt.Assert.NoError(err)

	var filterCfgResource filterResource
	json.Unmarshal(raw, &filterCfgResource)
	tt.Assert.NoError(err)

	tt.Assert.Equal(filterCfgResource.Name, "xyz")
	tt.Assert.Equal(len(filterCfgResource.Rules["whitelist"].([]interface{})), 3)
	tt.Assert.Equal(filterCfgResource.Enabled, false)
	tt.Assert.True(filterCfgResource.LastModified > 0)
}

func TestGetFilterConfigListResult(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{SessionInterface: tt.HorizonSession()}

	fc1 := history.FilterConfig{
		Rules:   `{"whitelist": ["1","2","3"]}`,
		Name:    "xyz",
		Enabled: false,
	}

	q.UpsertFilterConfig(tt.Ctx, fc1)

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

	var filterCfgResourceList []filterResource
	json.Unmarshal(raw, &filterCfgResourceList)
	tt.Assert.NoError(err)

	tt.Assert.Len(filterCfgResourceList, 1)
	tt.Assert.Equal(filterCfgResourceList[0].Name, "xyz")
	tt.Assert.Equal(len(filterCfgResourceList[0].Rules["whitelist"].([]interface{})), 3)
	tt.Assert.Equal(filterCfgResourceList[0].Enabled, false)
	tt.Assert.True(filterCfgResourceList[0].LastModified > 0)
}

func TestUpdateUnsupportedFilterConfig(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{SessionInterface: tt.HorizonSession()}

	fc1 := history.FilterConfig{
		Rules:   `{"whitelist": ["1","2","3"]}`,
		Name:    "unsupported",
		Enabled: false,
	}

	q.UpsertFilterConfig(tt.Ctx, fc1)

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
	tt.Assert.Equal(http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateFilterConfig(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{SessionInterface: tt.HorizonSession()}

	fc1 := history.FilterConfig{
		Rules:   `{"whitelist": ["1","2","3"]}`,
		Name:    filters.FilterAssetFilterName,
		Enabled: false,
	}

	q.UpsertFilterConfig(tt.Ctx, fc1)

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
	tt.Assert.Equal(filterRules["whitelist"].([]interface{})[0], "4")
}
