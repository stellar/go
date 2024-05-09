package compressxdr

import (
	"github.com/klauspost/compress/zstd"
	"io"
)

var DefaultCompressor = &ZstdCompressor{}

// Compressor represents a compression algorithm.
type Compressor interface {
	NewWriter(w io.Writer) (io.WriteCloser, error)
	NewReader(r io.Reader) (io.ReadCloser, error)
	GetName() string
}

// ZstdCompressor is an implementation of the Compressor interface for Zstd compression.
type ZstdCompressor struct{}

// GetName returns the name of the compression algorithm.
func (z *ZstdCompressor) GetName() string {
	return "zstd"
}

// NewWriter creates a new Zstd writer.
func (z *ZstdCompressor) NewWriter(w io.Writer) (io.WriteCloser, error) {
	return zstd.NewWriter(w)
}

// NewReader creates a new Zstd reader.
func (z *ZstdCompressor) NewReader(r io.Reader) (io.ReadCloser, error) {
	zr, err := zstd.NewReader(r)
	if err != nil {
		return nil, err
	}
	return zr.IOReadCloser(), err
}
