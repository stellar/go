package index

import (
	"bytes"
	"testing"

	types "github.com/stellar/go/exp/lighthorizon/index/types"
	"github.com/stretchr/testify/require"
)

func TestGzipWriteReadRoundtrip(t *testing.T) {
	indexes := types.NamedIndices{}
	index := &types.CheckpointIndex{}
	index.SetActive(5)
	indexes["A"] = index

	var buf bytes.Buffer
	_, err := writeGzippedTo(&buf, indexes)
	require.NoError(t, err)

	read, _, err := readGzippedFrom(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	require.Equal(t, indexes, read)
}
