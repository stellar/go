package logmetrics

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
)

func TestLogPackage(t *testing.T) {
	Convey("Metrics", t, func() {
		output := new(bytes.Buffer)
		l, m := New()
		l.Logger.Formatter.(*logrus.TextFormatter).DisableColors = true
		l.Logger.Level = logrus.DebugLevel
		l.Logger.Out = output

		for _, meter := range *m {
			So(meter.Count(), ShouldEqual, 0)
		}

		l.Debug("foo")
		l.Info("foo")
		l.Warn("foo")
		l.Error("foo")
		So(func() {
			l.Panic("foo")
		}, ShouldPanic)

		for _, meter := range *m {
			So(meter.Count(), ShouldEqual, 1)
		}
	})

}
