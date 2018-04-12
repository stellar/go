package test

import (
	"github.com/sirupsen/logrus"
	"github.com/stellar/go/services/horizon/internal/log"
)

var testLogger *log.Entry

func init() {
	testLogger, _ = log.New()
	testLogger.Entry.Logger.Formatter.(*logrus.TextFormatter).DisableColors = true
	testLogger.Entry.Logger.Level = logrus.DebugLevel
}
