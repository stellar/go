package clocktest

import (
	"time"

	"github.com/stellar/go/support/clock"
)

type fixedSource time.Time

func (s fixedSource) Now() time.Time {
	return time.Time(s)
}

// NewFixed creates a Clock with its time stopped and fixed to the time
// given.
func NewFixed(t time.Time) clock.Clock {
	return clock.Clock{
		Source: fixedSource(t),
	}
}
