package ioutil

import (
	"bytes"
	"crypto/rand"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReadAllSafe_lessThanMax tests if the size of the contents of the reader
// is less than the max size that it returns a byte slice containing the full
// contents of the reader.
func TestReadAllSafe_lessThanMax(t *testing.T) {
	bOrig := [50]byte{}
	_, err := rand.Read(bOrig[:])
	require.NoError(t, err)

	// Copy the original bytes to ensure we don't pollute it.
	rBytes := bOrig
	r := bytes.NewReader(rBytes[:])

	bRead, err := ReadAllSafe(r, 100)
	require.NoError(t, err)
	assert.Equal(t, bOrig[:], bRead)
}

// TestReadAllSafe_eqToMax tests if the size of the contents of the reader is
// equal to the max size that it returns a byte slice containing the full
// contents of the reader.
func TestReadAllSafe_eqToMax(t *testing.T) {
	bOrig := [50]byte{}
	_, err := rand.Read(bOrig[:])
	require.NoError(t, err)

	// Copy the original bytes to ensure we don't pollute it.
	rBytes := bOrig
	r := bytes.NewReader(rBytes[:])

	bRead, err := ReadAllSafe(r, 50)
	require.NoError(t, err)
	assert.Equal(t, bOrig[:], bRead)
}

// TestReadAllSafe_greaterThanMax tests if the size of the contents of the
// reader is greater than the max size that it returns a byte slice containing
// the full contents of the reader.
func TestReadAllSafe_greaterThanMax(t *testing.T) {
	bOrig := [50]byte{}
	_, err := rand.Read(bOrig[:])
	require.NoError(t, err)

	// Copy the original bytes to ensure we don't pollute it.
	rBytes := bOrig
	r := bytes.NewReader(rBytes[:])

	bRead, err := ReadAllSafe(r, 40)
	require.NoError(t, err)
	assert.Equal(t, bOrig[:40], bRead)
}

// TestReadAllSafe_passThroughErr tests if the reader returns an error it is
// passed through.
func TestReadAllSafe_passThroughErr(t *testing.T) {
	wantErr := errors.New("an error occurred")
	r := errReader{Err: wantErr}
	_, err := ReadAllSafe(r, 40)
	assert.Error(t, err, wantErr)
}

type errReader struct {
	Err error
}

func (er errReader) Read(_ []byte) (n int, err error) {
	return 0, er.Err
}
