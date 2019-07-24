package io

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryStateReaderTempStoreOpen(t *testing.T) {
	s := MemoryStateReaderTempStore{}
	assert.Nil(t, s.m)
	err := s.Open()
	assert.NoError(t, err)
	assert.NotNil(t, s.m)

	err = s.Set("a", true)
	assert.NoError(t, err)

	err = s.Set("b", false)
	assert.NoError(t, err)

	v, err := s.Get("a")
	assert.NoError(t, err)
	assert.True(t, v)

	v, err = s.Get("b")
	assert.NoError(t, err)
	assert.False(t, v)

	// Get for not-set key should return false
	v, err = s.Get("c")
	assert.NoError(t, err)
	assert.False(t, v)

	err = s.Close()
	assert.NoError(t, err)
	assert.Nil(t, s.m)
}
