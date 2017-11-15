package time

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMillisRoundUp(t *testing.T) {
	assert.Equal(t, MillisFromInt64(20), MillisFromInt64(13).RoundUp(10))
	assert.Equal(t, MillisFromInt64(10), MillisFromInt64(10).RoundUp(10))
}

func TestMillisRoundDown(t *testing.T) {
	assert.Equal(t, MillisFromInt64(10), MillisFromInt64(13).RoundDown(10))
	assert.Equal(t, MillisFromInt64(10), MillisFromInt64(10).RoundDown(10))
}

func TestMillisParsing(t *testing.T) {
	expected := Now()
	actual, err := MillisFromString(expected.String())
	assert.NoError(t, err)
	assert.Equal(t, actual, expected)
}