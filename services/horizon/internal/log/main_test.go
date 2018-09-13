package log

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	ge "github.com/go-errors/errors"
)

func TestSet(t *testing.T) {
	assert.Nil(t, context.Background().Value(&contextKey))
	l, _ := New()
	ctx := Set(context.Background(), l)
	assert.Equal(t, l, ctx.Value(&contextKey))
}

func TestCtx(t *testing.T) {
	// defaults to the default logger
	assert.Equal(t, DefaultLogger, Ctx(context.Background()))

	// a set value overrides the default
	l, _ := New()

	ctx := Set(context.Background(), l)
	assert.Equal(t, l, Ctx(ctx))

	// the deepest set value is returns
	nested, _ := New()
	nctx := Set(ctx, nested)
	assert.Equal(t, nested, Ctx(nctx))
}

func TestPushContext(t *testing.T) {
	output := new(bytes.Buffer)
	l, _ := New()
	l.Logger.Formatter.(*logrus.TextFormatter).DisableColors = true
	l.Logger.Out = output
	ctx := Set(context.Background(), l.WithField("foo", "bar"))

	Ctx(ctx).Warn("hello")
	assert.Contains(t, output.String(), "foo=bar")
	assert.NotContains(t, output.String(), "foo=baz")

	ctx = PushContext(ctx, func(logger *Entry) *Entry {
		return logger.WithField("foo", "baz")
	})

	Ctx(ctx).Warn("hello")
	assert.Contains(t, output.String(), "foo=baz")
}

type LoggingStatementsTestSuite struct {
	suite.Suite
	output *bytes.Buffer
	l      *Entry
	m      *Metrics
}

func (suite *LoggingStatementsTestSuite) SetupTest() {
	suite.output = new(bytes.Buffer)
	suite.l, suite.m = New()
	suite.l.Logger.Formatter.(*logrus.TextFormatter).DisableColors = true
	suite.l.Logger.Out = suite.output
}

// Tests that the default logging severity level is Warn
func (suite *LoggingStatementsTestSuite) TestDefaultSeverityLevel() {
	suite.l.Debug("debug")
	suite.l.Info("info")
	suite.l.Warn("warn")

	assert.NotContains(suite.T(), suite.output.String(), "level=info")
	assert.NotContains(suite.T(), suite.output.String(), "level=debug")
	assert.Contains(suite.T(), suite.output.String(), "level=warn")

}

// Tests logging statements that fall under the Debug level
func (suite *LoggingStatementsTestSuite) TestSeverityLevelDebug() {
	suite.l.Logger.Level = logrus.InfoLevel
	suite.l.Debug("Debug")
	assert.Equal(suite.T(), "", suite.output.String())

	suite.l.Logger.Level = logrus.DebugLevel
	suite.l.Debug("Debug")
	assert.Contains(suite.T(), suite.output.String(), "level=debug")
	assert.Contains(suite.T(), suite.output.String(), "msg=Debug")
}

// Tests logging statements that fall under the Info level
func (suite *LoggingStatementsTestSuite) TestSeverityLevelInfo() {
	suite.l.Logger.Level = logrus.WarnLevel
	suite.l.Debug("foo")
	suite.l.Info("foo")
	assert.Equal(suite.T(), "", suite.output.String())

	suite.l.Logger.Level = logrus.InfoLevel
	suite.l.Info("foo")
	assert.Contains(suite.T(), suite.output.String(), "level=info")
	assert.Contains(suite.T(), suite.output.String(), "msg=foo")
}

// Tests logging statements that fall under the Warn level
func (suite *LoggingStatementsTestSuite) TestSeverityLevelWarn() {
	suite.l.Logger.Level = logrus.ErrorLevel
	suite.l.Debug("foo")
	suite.l.Info("foo")
	suite.l.Warn("foo")
	assert.Equal(suite.T(), "", suite.output.String())

	suite.l.Logger.Level = logrus.WarnLevel
	suite.l.Warn("foo")
	assert.Contains(suite.T(), suite.output.String(), "level=warn")
	assert.Contains(suite.T(), suite.output.String(), "msg=foo")
}

// Tests logging statements that fall under the Error level
func (suite *LoggingStatementsTestSuite) TestSeverityLevelError() {
	suite.l.Logger.Level = logrus.FatalLevel
	suite.l.Debug("foo")
	suite.l.Info("foo")
	suite.l.Warn("foo")
	suite.l.Error("foo")
	assert.Equal(suite.T(), "", suite.output.String())

	suite.l.Logger.Level = logrus.ErrorLevel
	suite.l.Error("foo")
	assert.Contains(suite.T(), suite.output.String(), "level=error")
	assert.Contains(suite.T(), suite.output.String(), "msg=foo")
}

// Tests logging statements that fall under the Panic level
func (suite *LoggingStatementsTestSuite) TestSeverityLevelPanic() {
	suite.l.Logger.Level = logrus.PanicLevel
	suite.l.Debug("foo")
	suite.l.Info("foo")
	suite.l.Warn("foo")
	suite.l.Error("foo")
	assert.Equal(suite.T(), "", suite.output.String())

	assert.Panics(suite.T(), func() {
		suite.l.Panic("foo")
	})

	assert.Contains(suite.T(), suite.output.String(), "level=panic")
	assert.Contains(suite.T(), suite.output.String(), "msg=foo")
}

// Adds the stack properly if a go-errors.Error is provided
func (suite *LoggingStatementsTestSuite) TestWithStack() {
	err := ge.New("broken")
	suite.l.WithStack(err).Error("test")
	// simply ensure that the line creating the above error is in the log
	assert.Contains(suite.T(), suite.output.String(), "main_test.go:")
}

//Adds stack=unknown when the provided err has not stack info
func (suite *LoggingStatementsTestSuite) TestWithUnknownStack() {
	suite.l.WithStack(errors.New("broken")).Error("test")
	assert.Contains(suite.T(), suite.output.String(), "stack=unknown")
}

// Tests that metric counts are correct
func (suite *LoggingStatementsTestSuite) TestMetrics() {
	suite.l.Logger.Level = logrus.DebugLevel

	for _, meter := range *suite.m {
		assert.Equal(suite.T(), int64(0), meter.Count())
	}

	suite.l.Debug("foo")
	suite.l.Info("foo")
	suite.l.Warn("foo")
	suite.l.Error("foo")
	assert.Panics(suite.T(), func() {
		suite.l.Panic("foo")
	})

	for _, meter := range *suite.m {
		assert.Equal(suite.T(), int64(1), meter.Count())
	}
}

func TestLoggingStatementsTestSuite(t *testing.T) {
	suite.Run(t, new(LoggingStatementsTestSuite))
}
