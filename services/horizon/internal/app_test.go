package horizon

import (
	"net/http"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stellar/go/services/horizon/internal/test"
)

func TestGenericHTTPFeatures(t *testing.T) {
	ht := StartHTTPTest(t, "base")
	defer ht.Finish()

	// CORS
	w := ht.Get("/")
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.Empty(w.HeaderMap.Get("Access-Control-Allow-Origin"))
	}

	w = ht.Get("/", func(r *http.Request) {
		r.Header.Set("Origin", "somewhere.com")
	})

	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.Equal(
			"somewhere.com",
			w.HeaderMap.Get("Access-Control-Allow-Origin"),
		)
	}

	// Trailing slash is stripped
	w = ht.Get("/ledgers")
	ht.Assert.Equal(200, w.Code)
	w = ht.Get("/ledgers/")
	ht.Assert.Equal(200, w.Code)
}
func TestMetrics(t *testing.T) {
	ht := StartHTTPTest(t, "base")
	defer ht.Finish()

	adminRouterRH := test.NewRequestHelper(ht.App.webServer.Router.Internal)
	w := adminRouterRH.Get("/metrics")
	ht.Assert.Equal(200, w.Code)

	hl := ht.App.historyLatestLedgerCounter
	he := ht.App.historyElderLedgerCounter
	cl := ht.App.coreLatestLedgerCounter

	ht.Require.EqualValues(3, getMetricValue(hl).GetCounter().GetValue())
	ht.Require.EqualValues(1, getMetricValue(he).GetCounter().GetValue())
	ht.Require.EqualValues(64, getMetricValue(cl).GetCounter().GetValue())
}

func getMetricValue(metric prometheus.Metric) *dto.Metric {
	value := &dto.Metric{}
	err := metric.Write(value)
	if err != nil {
		panic(err)
	}
	return value
}
