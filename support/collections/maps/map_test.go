package maps

import (
	"testing"

	"github.com/stellar/go/support/collections/set"
	"github.com/stretchr/testify/require"
)

func TestSanity(t *testing.T) {
	m := map[int]float32{1: 10, 2: 20, 3: 30}
	require.Equal(t, []int{1, 2, 3}, Keys(m))
	require.Equal(t, []float32{10, 20, 30}, Values(m))

	// compatibility with collections/set.Set
	s := set.Set[float32]{}
	s.Add(1)
	s.Add(2)
	s.Add(3)

	require.Equal(t, []float32{1, 2, 3}, Keys(s))
}
