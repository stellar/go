package horizon

import (
	"fmt"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/support/render/hal"
)

// Interface verification
var _ actions.JSONer = (*MetricsAction)(nil)

// MetricsAction collects and renders a snapshot from the metrics system that
// will inlude any previously registered metrics.
type MetricsAction struct {
	Action
	Snapshot map[string]interface{}
}

// JSON is a method for actions.JSON
func (action *MetricsAction) JSON() error {
	action.LoadSnapshot()
	action.Snapshot["_links"] = map[string]interface{}{
		"self": hal.NewLink("/metrics"),
	}

	hal.Render(action.W, action.Snapshot)
	return action.Err
}

// PrometheusFormat is a method for actions.PrometheusResponder
func (action *MetricsAction) PrometheusFormat() error {
	action.App.metrics.Each(func(name string, i interface{}) {
		switch metric := i.(type) {
		case metrics.Counter:
			fmt.Fprintf(action.W, "%s %d\n", name, metric.Count())
		case metrics.Gauge:
			fmt.Fprintf(action.W, "%s %d\n", name, metric.Value())
		case metrics.GaugeFloat64:
			fmt.Fprintf(action.W, "%s %f\n", name, metric.Value())
		case metrics.Histogram:
			h := metric.Snapshot()
			ps := h.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})

			fmt.Fprintf(action.W, "%s_count %d\n", name, h.Count())
			fmt.Fprintf(action.W, "%s_min %d\n", name, h.Min())
			fmt.Fprintf(action.W, "%s_max %d\n", name, h.Max())
			fmt.Fprintf(action.W, "%s_mean %f\n", name, h.Mean())
			fmt.Fprintf(action.W, "%s_stddev %f\n", name, h.StdDev())
			fmt.Fprintf(action.W, "%s_bucket{quantile=\"0.50\"} %f\n", name, ps[0])
			fmt.Fprintf(action.W, "%s_bucket{quantile=\"0.75\"} %f\n", name, ps[1])
			fmt.Fprintf(action.W, "%s_bucket{quantile=\"0.95\"} %f\n", name, ps[2])
			fmt.Fprintf(action.W, "%s_bucket{quantile=\"0.99\"} %f\n", name, ps[3])
			fmt.Fprintf(action.W, "%s_bucket{quantile=\"0.999\"} %f\n", name, ps[4])
		case metrics.Meter:
			m := metric.Snapshot()

			fmt.Fprintf(action.W, "%s_count %d\n", name, m.Count())
			fmt.Fprintf(action.W, "%s_1m_rate %f\n", name, m.Rate1())
			fmt.Fprintf(action.W, "%s_5m_rate %f\n", name, m.Rate5())
			fmt.Fprintf(action.W, "%s_15m_rate %f\n", name, m.Rate15())
			fmt.Fprintf(action.W, "%s_mean_rate %f\n", name, m.RateMean())
		case metrics.Timer:
			t := metric.Snapshot()
			ps := t.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})

			fmt.Fprintf(action.W, "%s_count %d\n", name, t.Count())
			fmt.Fprintf(action.W, "%s_min %f\n", name, time.Duration(t.Min()).Seconds())
			fmt.Fprintf(action.W, "%s_max %f\n", name, time.Duration(t.Max()).Seconds())
			fmt.Fprintf(action.W, "%s_mean %f\n", name, time.Duration(t.Mean()).Seconds())
			fmt.Fprintf(action.W, "%s_stddev %f\n", name, time.Duration(t.StdDev()).Seconds())
			fmt.Fprintf(action.W, "%s_bucket{quantile=\"0.50\"} %f\n", name, time.Duration(ps[0]).Seconds())
			fmt.Fprintf(action.W, "%s_bucket{quantile=\"0.75\"} %f\n", name, time.Duration(ps[1]).Seconds())
			fmt.Fprintf(action.W, "%s_bucket{quantile=\"0.95\"} %f\n", name, time.Duration(ps[2]).Seconds())
			fmt.Fprintf(action.W, "%s_bucket{quantile=\"0.99\"} %f\n", name, time.Duration(ps[3]).Seconds())
			fmt.Fprintf(action.W, "%s_bucket{quantile=\"0.999\"} %f\n", name, time.Duration(ps[4]).Seconds())
			fmt.Fprintf(action.W, "%s_1m_rate %f\n", name, t.Rate1())
			fmt.Fprintf(action.W, "%s_5m_rate %f\n", name, t.Rate5())
			fmt.Fprintf(action.W, "%s_15m_rate %f\n", name, t.Rate15())
			fmt.Fprintf(action.W, "%s_mean_rate %f\n", name, t.RateMean())
		}
		fmt.Fprintf(action.W, "\n")
	})

	return action.Err
}

// LoadSnapshot populates action.Snapshot
//
// Original code copied from github.com/rcrowley/go-metrics MarshalJSON
func (action *MetricsAction) LoadSnapshot() {
	action.Snapshot = make(map[string]interface{})

	action.App.metrics.Each(func(name string, i interface{}) {
		values := make(map[string]interface{})
		switch metric := i.(type) {
		case metrics.Counter:
			values["count"] = metric.Count()
		case metrics.Gauge:
			values["value"] = metric.Value()
		case metrics.GaugeFloat64:
			values["value"] = metric.Value()
		case metrics.Healthcheck:
			values["error"] = nil
			metric.Check()
			if err := metric.Error(); nil != err {
				values["error"] = metric.Error().Error()
			}
		case metrics.Histogram:
			h := metric.Snapshot()
			ps := h.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
			values["count"] = h.Count()
			values["min"] = h.Min()
			values["max"] = h.Max()
			values["mean"] = h.Mean()
			values["stddev"] = h.StdDev()
			values["median"] = ps[0]
			values["75%"] = ps[1]
			values["95%"] = ps[2]
			values["99%"] = ps[3]
			values["99.9%"] = ps[4]
		case metrics.Meter:
			m := metric.Snapshot()
			values["count"] = m.Count()
			values["1m.rate"] = m.Rate1()
			values["5m.rate"] = m.Rate5()
			values["15m.rate"] = m.Rate15()
			values["mean.rate"] = m.RateMean()
		case metrics.Timer:
			t := metric.Snapshot()
			ps := t.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
			values["count"] = t.Count()
			values["min"] = time.Duration(t.Min()).Seconds()
			values["max"] = time.Duration(t.Max()).Seconds()
			values["mean"] = time.Duration(t.Mean()).Seconds()
			values["stddev"] = time.Duration(t.StdDev()).Seconds()
			values["median"] = ps[0] / float64(time.Second)
			values["75%"] = ps[1] / float64(time.Second)
			values["95%"] = ps[2] / float64(time.Second)
			values["99%"] = ps[3] / float64(time.Second)
			values["99.9%"] = ps[4] / float64(time.Second)
			values["1m.rate"] = t.Rate1()
			values["5m.rate"] = t.Rate5()
			values["15m.rate"] = t.Rate15()
			values["mean.rate"] = t.RateMean()
		}
		action.Snapshot[name] = values
	})
}
