package horizon

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stellar/go/services/horizon/internal/reap"
	"github.com/stellar/go/services/horizon/internal/test"
	tdb "github.com/stellar/go/services/horizon/internal/test/db"
	"github.com/stellar/go/support/db"
)

type HTTPT struct {
	Assert     *test.Assertions
	App        *App
	RH         test.RequestHelper
	coreServer *test.StaticMockServer
	reaper     *reap.System
	*test.T
}

func startHTTPTest(t *testing.T, scenario string) *HTTPT {
	ret := &HTTPT{T: test.Start(t)}
	if scenario == "" {
		test.ResetHorizonDB(t, ret.HorizonDB)
	} else {
		ret.Scenario(scenario)
	}
	ret.App = NewTestApp(tdb.HorizonURL())
	ret.RH = test.NewRequestHelper(ret.App.webServer.Router.Mux)
	ret.Assert = &test.Assertions{ret.T.Assert}

	ret.coreServer = test.NewStaticMockServer(`{
		"info": {
			"network": "test",
			"build": "test-core",
			"ledger": {
				"version": 18,
				"num": 64
			},
			"protocol_version": 18,
			"network": "Test SDF Network ; September 2015"
		}
	}`)

	ret.App.config.StellarCoreURL = ret.coreServer.URL
	ret.App.UpdateCoreLedgerState(context.Background())
	ret.App.UpdateStellarCoreInfo(context.Background())
	ret.App.UpdateHorizonLedgerState(context.Background())
	ret.reaper = reap.New(
		uint32(ret.App.config.HistoryRetentionCount),
		uint32(ret.App.config.HistoryRetentionReapCount),
		mustNewDBSession(
			db.ReapSubservice, ret.App.config.DatabaseURL, 1, 1, ret.App.prometheusRegistry,
		))

	t.Cleanup(func() {
		ret.Assert.NoError(ret.reaper.Close())
	})

	return ret
}

// StartHTTPTest is a helper function to setup a new test that will make http
// requests. Pair it with a deferred call to FinishHTTPTest.
func StartHTTPTest(t *testing.T, scenario string) *HTTPT {
	if scenario == "" {
		t.Fatal("scenario cannot be empty string")
	}
	return startHTTPTest(t, scenario)
}

// StartHTTPTestWithoutScenario is like StartHTTPTest except it does not use
// a sql scenario
func StartHTTPTestWithoutScenario(t *testing.T) *HTTPT {
	return startHTTPTest(t, "")
}

// Get delegates to the test's request helper
func (ht *HTTPT) Get(
	path string,
	fn ...func(*http.Request),
) *httptest.ResponseRecorder {
	return ht.RH.Get(path, fn...)
}

// GetWithParams delegates to the test's request helper and encodes along with the query params
func (ht *HTTPT) GetWithParams(
	path string,
	queryParams url.Values,
	fn ...func(*http.Request),
) *httptest.ResponseRecorder {
	return ht.RH.Get(path+"?"+queryParams.Encode(), fn...)
}

// Finish closes the test app and finishes the test
func (ht *HTTPT) Finish() {
	ht.T.Finish()
	ht.App.Close()
	ht.coreServer.Close()
}

// Post delegates to the test's request helper
func (ht *HTTPT) Post(
	path string,
	form url.Values,
	mods ...func(*http.Request),
) *httptest.ResponseRecorder {
	return ht.RH.Post(path, form, mods...)
}

// ReapHistory causes the test server to run `DeleteUnretainedHistory`, after
// setting the retention count to the provided number.
func (ht *HTTPT) ReapHistory(retention uint32) {
	ht.reaper.RetentionCount = retention
	ht.reaper.RetentionBatch = 50_000
	ht.Require.NoError(ht.reaper.DeleteUnretainedHistory(context.Background()))
	ht.App.UpdateCoreLedgerState(context.Background())
	ht.App.UpdateHorizonLedgerState(context.Background())
}
