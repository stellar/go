package log

import (
	metrics "github.com/rcrowley/go-metrics"
	"github.com/sirupsen/logrus"
)

// Metrics is a logrus hook-compliant struct that records metrics about logging
// when added to a logrus.Logger
type Metrics map[logrus.Level]metrics.Meter

// NewMetrics creates a new hook for recording metrics.
func NewMetrics() *Metrics {
	return &Metrics{
		logrus.DebugLevel: metrics.NewMeter(),
		logrus.InfoLevel:  metrics.NewMeter(),
		logrus.WarnLevel:  metrics.NewMeter(),
		logrus.ErrorLevel: metrics.NewMeter(),
		logrus.PanicLevel: metrics.NewMeter(),
	}
}

// Fire is triggered by logrus, in response to a logging event
func (m *Metrics) Fire(e *logrus.Entry) error {
	(*m)[e.Level].Mark(1)
	return nil
}

// Levels returns the logging levels that will trigger this hook to run.  In
// this case, all of them.
func (m *Metrics) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.DebugLevel,
		logrus.InfoLevel,
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.PanicLevel,
	}
}
