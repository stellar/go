package ticker

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi"

	"github.com/stellar/go/services/ticker/internal/gql"
	"github.com/stellar/go/services/ticker/internal/tickerdb/tickerdbtest"
	hlog "github.com/stellar/go/support/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTicker(t *testing.T) {
	session := tickerdbtest.SetupTickerTestSession(t, "./tickerdb/migrations")
	defer session.DB.Close()

	logger := hlog.New()
	resolver := gql.New(&session, logger)
	h := resolver.SetupHandler()
	m := chi.NewMux()
	m.Post("/graphql", h.ServeHTTP)

	type test struct {
		queryField string
		queryVal   string
		respField  string
		wantBody   string
	}

	tests := []test{
		// All response fields, with a single queried pair.
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\"]`, respField: "tradePair", wantBody: `{"tradePair": "BTC_ETH"}`},
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\"]`, respField: "baseAssetCode", wantBody: `{"baseAssetCode": "BTC"}`},
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\"]`, respField: "counterAssetCode", wantBody: `{"counterAssetCode": "ETH"}`},
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\"]`, respField: "baseVolume", wantBody: `{"baseVolume": 174}`},
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\"]`, respField: "counterVolume", wantBody: `{"counterVolume": 86}`},
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\"]`, respField: "tradeCount", wantBody: `{"tradeCount": 3}`},
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\"]`, respField: "open", wantBody: `{"open": 1}`},
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\"]`, respField: "close", wantBody: `{"close": 0.92}`},
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\"]`, respField: "high", wantBody: `{"high": 1.0}`},
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\"]`, respField: "low", wantBody: `{"low": 0.1}`},
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\"]`, respField: "orderbookStats { bidCount }", wantBody: `{"orderbookStats": {"bidCount": 16}}`},
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\"]`, respField: "orderbookStats { bidVolume }", wantBody: `{"orderbookStats": {"bidVolume": 0.25}}`},
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\"]`, respField: "orderbookStats { bidMax }", wantBody: `{"orderbookStats": {"bidMax": 200}}`},
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\"]`, respField: "orderbookStats { askCount }", wantBody: `{"orderbookStats": {"askCount": 18}}`},
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\"]`, respField: "orderbookStats { askVolume }", wantBody: `{"orderbookStats": {"askVolume": 45}}`},
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\"]`, respField: "orderbookStats { askMin }", wantBody: `{"orderbookStats": {"askMin": 0.1}}`},
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\"]`, respField: "orderbookStats { spread }", wantBody: `{"orderbookStats": {"spread": -1999}}`},
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\"]`, respField: "orderbookStats { spreadMidPoint }", wantBody: `{"orderbookStats": {"spreadMidPoint": -799.5}}`},
		// Check reversing base and counter.
		{queryField: "pairNames", queryVal: `[\"ETH_BTC\"]`, respField: "tradePair", wantBody: `{"tradePair": "ETH_BTC"}`},
		{queryField: "pairNames", queryVal: `[\"ETH_BTC\"]`, respField: "baseAssetCode", wantBody: `{"baseAssetCode": "ETH"}`},
		{queryField: "pairNames", queryVal: `[\"ETH_BTC\"]`, respField: "counterAssetCode", wantBody: `{"counterAssetCode": "BTC"}`},
		{queryField: "pairNames", queryVal: `[\"ETH_BTC\"]`, respField: "baseVolume", wantBody: `{"baseVolume": 86}`},
		{queryField: "pairNames", queryVal: `[\"ETH_BTC\"]`, respField: "counterVolume", wantBody: `{"counterVolume": 174}`},
		{queryField: "pairNames", queryVal: `[\"ETH_BTC\"]`, respField: "tradeCount", wantBody: `{"tradeCount": 3}`},
		// Other input cases: multiple pairs, code.
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\", \"BTC_XLM\"]`, respField: "tradePair", wantBody: `{"tradePair": "BTC_ETH"},{"tradePair": "BTC_XLM"}`},
		{queryField: "pairNames", queryVal: `[\"BTC_ETH\", \"BTC_XLM\"]`, respField: "tradeCount", wantBody: `{"tradeCount": 3},{"tradeCount": 2}`},
		{queryField: "code", queryVal: `\"BTC\"`, respField: "tradePair", wantBody: `{"tradePair": "BTC_ETH"},{"tradePair": "BTC_XLM"}`},
	}

	for _, tc := range tests {
		req := fmt.Sprintf(`{
			"query": "query getTicker() {ticker(%s: %s, numHoursAgo: 24) {%s}}",
			"operationName": "getTicker",
			"variables": {}
		}`, tc.queryField, tc.queryVal, tc.respField)

		fmt.Println(req)
		r := httptest.NewRequest("POST", "/graphql", strings.NewReader(req))
		w := httptest.NewRecorder()

		m.ServeHTTP(w, r)
		resp := w.Result()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		wantBody := fmt.Sprintf(`{"data":{"ticker":[%s]}}`, tc.wantBody)
		assert.JSONEq(t, wantBody, string(body))
	}
}

func TestMarkets(t *testing.T) {
	session := tickerdbtest.SetupTickerTestSession(t, "./tickerdb/migrations")
	defer session.DB.Close()

	logger := hlog.New()
	resolver := gql.New(&session, logger)
	h := resolver.SetupHandler()
	m := chi.NewMux()
	m.Post("/graphql", h.ServeHTTP)

	issuerPK := "GCF3TQXKZJNFJK7HCMNE2O2CUNKCJH2Y2ROISTBPLC7C5EIA5NNG2XZB"

	type test struct {
		field    string
		wantBody string
	}
	tests := []test{
		{field: "tradePair", wantBody: fmt.Sprintf(`{"tradePair":"BTC:%s / ETH:%s"}`, issuerPK, issuerPK)},
		{field: "baseAssetCode", wantBody: `{"baseAssetCode":"BTC"}`},
		{field: "baseAssetIssuer", wantBody: fmt.Sprintf(`{"baseAssetIssuer":"%s"}`, issuerPK)},
		{field: "counterAssetCode", wantBody: `{"counterAssetCode":"ETH"}`},
		{field: "counterAssetIssuer", wantBody: fmt.Sprintf(`{"counterAssetIssuer":"%s"}`, issuerPK)},
		{field: "baseVolume", wantBody: `{"baseVolume":150}`},
		{field: "counterVolume", wantBody: `{"counterVolume":60}`},
		{field: "tradeCount", wantBody: `{"tradeCount":2}`},
		{field: "open", wantBody: `{"open":1}`},
		{field: "low", wantBody: `{"low":0.1}`},
		{field: "high", wantBody: `{"high":1}`},
		{field: "close", wantBody: `{"close":0.1}`},
		{field: "change", wantBody: `{"change":-0.9}`},
		{field: "orderbookStats { bidCount }", wantBody: `{"orderbookStats": {"bidCount": 15}}`},
		{field: "orderbookStats { bidVolume }", wantBody: `{"orderbookStats": {"bidVolume": 0.15}}`},
		{field: "orderbookStats { bidMax }", wantBody: `{"orderbookStats": {"bidMax": 200}}`},
		{field: "orderbookStats { askCount }", wantBody: `{"orderbookStats": {"askCount": 17}}`},
		{field: "orderbookStats { askVolume }", wantBody: `{"orderbookStats": {"askVolume": 30}}`},
		{field: "orderbookStats { askMin }", wantBody: `{"orderbookStats": {"askMin": 0.1}}`},
		{field: "orderbookStats { askMin }", wantBody: `{"orderbookStats": {"askMin": 0.1}}`},
	}

	for _, tc := range tests {
		queryStr := fmt.Sprintf(
			`baseAssetCode: \"BTC\", baseAssetIssuer: \"%s\", counterAssetCode: \"ETH\", counterAssetIssuer: \"%s\"`,
			issuerPK, issuerPK,
		)
		req := fmt.Sprintf(`
		{
			"query": "query getMarkets() {markets(%s, numHoursAgo: 24) {%s}}",
			"operationName": "getMarkets",
			"variables": {}
		}`, queryStr, tc.field)

		fmt.Println(req)
		r := httptest.NewRequest("POST", "/graphql", strings.NewReader(req))
		w := httptest.NewRecorder()

		m.ServeHTTP(w, r)
		resp := w.Result()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		wantBody := fmt.Sprintf(`{"data":{"markets":[%s]}}`, tc.wantBody)
		assert.JSONEq(t, wantBody, string(body))
	}
}
