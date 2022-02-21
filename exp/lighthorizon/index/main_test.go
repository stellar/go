package index

import (
	"fmt"
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
