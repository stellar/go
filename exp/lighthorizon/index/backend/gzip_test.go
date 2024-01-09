package index

import (
	"bufio"
	"bytes"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	types "github.com/stellar/go/exp/lighthorizon/index/types"
	"github.com/stretchr/testify/require"
)

func TestGzipRoundtrip(t *testing.T) {
	index := &types.BitmapIndex{}
	anotherIndex := &types.BitmapIndex{}
	for i := 0; i < 100+rand.Intn(1000); i++ {
		index.SetActive(uint32(rand.Intn(10_000)))
		anotherIndex.SetActive(uint32(rand.Intn(10_000)))
	}

	indices := types.NamedIndices{
		"a":                   index,
		"short/name":          anotherIndex,
		"slightlyLonger/name": index,
	}

	var buf bytes.Buffer
	wroteBytes, err := writeGzippedTo(&buf, indices)
	require.NoError(t, err)
	require.Greater(t, wroteBytes, int64(0))

	gz := filepath.Join(t.TempDir(), "test.gzip")
	require.NoError(t, os.WriteFile(gz, buf.Bytes(), 0644))
	f, err := os.Open(gz)
	require.NoError(t, err)
	defer f.Close()

	// Ensure that reading directly from a file errors out.
	_, _, err = readGzippedFrom(f)
	require.Error(t, err)

	read, readBytes, err := readGzippedFrom(bufio.NewReader(f))
	require.NoError(t, err)
	require.Greater(t, readBytes, int64(0))

	require.Equal(t, indices, read)
	require.Equal(t, wroteBytes, readBytes)
	require.Len(t, read, len(indices))

	for name, index := range indices {
		raw1, err := index.ToXDR().MarshalBinary()
		require.NoError(t, err)

		raw2, err := read[name].ToXDR().MarshalBinary()
		require.NoError(t, err)

		require.Equal(t, raw1, raw2)
	}
}
