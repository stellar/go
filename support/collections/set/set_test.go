package set

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSet(t *testing.T) {
	s := Set[string]{}
	s.Add("sanity")
	require.True(t, s.Contains("sanity"))
	require.False(t, s.Contains("check"))
}

func TestSafeSet(t *testing.T) {
	s := NewSafeSet[string](0)
	s.Add("sanity")
	require.True(t, s.Contains("sanity"))
	require.False(t, s.Contains("check"))
}
