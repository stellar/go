package index

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFromBytes(t *testing.T) {
	for i := uint32(1); i < 200; i++ {
		t.Run(fmt.Sprintf("New%d", i), func(t *testing.T) {
			index := &CheckpointIndex{}
			index.SetActive(i)
			b := index.Flush()
			newIndex, err := NewCheckpointIndexFromBytes(b)
			require.NoError(t, err)
			assert.Equal(t, index.firstCheckpoint, newIndex.firstCheckpoint)
			assert.Equal(t, index.bitmap, newIndex.bitmap)
			assert.Equal(t, index.shift, newIndex.shift)
		})
	}
}

func TestNextActive(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		index := &CheckpointIndex{}

		i, err := index.NextActive(0)
		assert.Equal(t, uint32(0), i)
		assert.EqualError(t, err, io.EOF.Error())
	})

	t.Run("one byte", func(t *testing.T) {
		t.Run("after last", func(t *testing.T) {
			index := &CheckpointIndex{}
			index.SetActive(3)

			// 16 is well-past the end
			i, err := index.NextActive(16)
			assert.Equal(t, uint32(0), i)
			assert.EqualError(t, err, io.EOF.Error())
		})

		t.Run("only one bit in the byte", func(t *testing.T) {
			index := &CheckpointIndex{}
			index.SetActive(1)

			i, err := index.NextActive(1)
			assert.NoError(t, err)
			assert.Equal(t, uint32(1), i)
		})

		t.Run("only one bit in the byte (offset)", func(t *testing.T) {
			index := &CheckpointIndex{}
			index.SetActive(9)

			i, err := index.NextActive(1)
			assert.NoError(t, err)
			assert.Equal(t, uint32(9), i)
		})

		severalSet := &CheckpointIndex{}
		severalSet.SetActive(9)
		severalSet.SetActive(11)

		t.Run("several bits set (first)", func(t *testing.T) {
			i, err := severalSet.NextActive(9)
			assert.NoError(t, err)
			assert.Equal(t, uint32(9), i)
		})

		t.Run("several bits set (second)", func(t *testing.T) {
			i, err := severalSet.NextActive(10)
			assert.NoError(t, err)
			assert.Equal(t, uint32(11), i)
		})

		t.Run("several bits set (second, inclusive)", func(t *testing.T) {
			i, err := severalSet.NextActive(11)
			assert.NoError(t, err)
			assert.Equal(t, uint32(11), i)
		})
	})

	t.Run("many bytes", func(t *testing.T) {
		index := &CheckpointIndex{}
		index.SetActive(9)
		index.SetActive(129)

		// Before the first
		i, err := index.NextActive(8)
		assert.NoError(t, err)
		assert.Equal(t, uint32(9), i)

		// at the first
		i, err = index.NextActive(9)
		assert.NoError(t, err)
		assert.Equal(t, uint32(9), i)

		// In the middle
		i, err = index.NextActive(11)
		assert.NoError(t, err)
		assert.Equal(t, uint32(129), i)

		// At the end
		i, err = index.NextActive(129)
		assert.NoError(t, err)
		assert.Equal(t, uint32(129), i)

		// after the end
		i, err = index.NextActive(130)
		assert.EqualError(t, err, io.EOF.Error())
		assert.Equal(t, uint32(0), i)
	})
}

func TestMaxBitAfter(t *testing.T) {
	for _, tc := range []struct {
		b     byte
		after uint32
		shift uint32
		ok    bool
	}{
		{0b0000_0000, 0, 0, false},
		{0b0000_0000, 1, 0, false},
		{0b1000_0000, 0, 0, true},
		{0b0100_0000, 0, 1, true},
		{0b0100_0000, 1, 1, true},
		{0b0010_1000, 0, 2, true},
		{0b0010_1000, 1, 2, true},
		{0b0010_1000, 2, 2, true},
		{0b0010_1000, 3, 4, true},
		{0b0010_1000, 4, 4, true},
	} {
		t.Run(fmt.Sprintf("0b%b,%d", tc.b, tc.after), func(t *testing.T) {
			shift, ok := maxBitAfter(tc.b, tc.after)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.shift, shift)
		})
	}
}
