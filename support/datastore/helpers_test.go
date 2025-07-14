package datastore

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// requireReaderContentEquals is a helper function to assert reader content.
func requireReaderContentEquals(t *testing.T, reader io.ReadCloser, expected []byte) {
	var buf bytes.Buffer
	_, err := io.Copy(&buf, reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Equal(t, expected, buf.Bytes())
}
