package index

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGzipWriteReadRoundtrip(t *testing.T) {
	indexes := map[string]*CheckpointIndex{}
	index := &CheckpointIndex{}
	index.SetActive(5)
	indexes["A"] = index

	var buf bytes.Buffer
	_, err := writeGzippedTo(&buf, indexes)
	require.NoError(t, err)

	read, _, err := readGzippedFrom(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	require.Equal(t, indexes, read)
}
