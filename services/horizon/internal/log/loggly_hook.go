package log

import (
	"os"
	"time"

	loggly "github.com/segmentio/go-loggly"
	"github.com/sirupsen/logrus"
)

// LogglyHook sends logs to loggly
type LogglyHook struct {
	client       *loggly.Client
	host         string
	FilteredKeys map[string]bool
}

// NewLogglyHook creates a new hook
func NewLogglyHook(token, tag string) *LogglyHook {
	client := loggly.New(token, tag)
	host, err := os.Hostname()

	if err != nil {
		panic("couldn't get hostname")
	}

	return &LogglyHook{
		client: client,
		host:   host,
	}
}

func (hook *LogglyHook) Fire(entry *logrus.Entry) error {
	logglyMessage := loggly.Message{
		"timestamp": entry.Time.UTC().Format(time.RFC3339Nano),
		"level":     entry.Level.String(),
		"message":   entry.Message,
		"hostname":  hook.host,
	}

	for k, v := range entry.Data {
		//Filter out keys
		if _, ok := hook.FilteredKeys[k]; ok {
			continue
		}

		logglyMessage[k] = v
	}

	return hook.client.Send(logglyMessage)
}

func (hook *LogglyHook) Flush() {
	hook.client.Flush()
}

func (hook *LogglyHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}
