package time

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRoundUp(t *testing.T) {
	assert.Equal(t, MillisFromInt64(20), MillisFromInt64(13).RoundUp(10))
	assert.Equal(t, MillisFromInt64(10), MillisFromInt64(10).RoundUp(10))
}

func TestRoundDown(t *testing.T) {
	assert.Equal(t, MillisFromInt64(10), MillisFromInt64(13).RoundDown(10))
	assert.Equal(t, MillisFromInt64(10), MillisFromInt64(10).RoundDown(10))
}

func TestParsing(t *testing.T) {
	expected := Now()
	actual, err := MillisFromString(expected.String())
	assert.NoError(t, err)
	assert.Equal(t, actual, expected)
}