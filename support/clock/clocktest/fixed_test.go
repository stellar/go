package clocktest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewFixed tests that the time returned from the fixed clocks Now()
// function is equal to the time given when creating the clock.
func TestNewFixed(t *testing.T) {
	timeNow := time.Date(2015, 9, 30, 17, 15, 54, 0, time.UTC)
	c := NewFixed(timeNow)
	assert.Equal(t, timeNow, c.Now())
}
