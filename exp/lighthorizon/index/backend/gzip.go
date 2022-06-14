package index

import (
	"bytes"
	"compress/gzip"
	"io"

	types "github.com/stellar/go/exp/lighthorizon/index/types"
)

func writeGzippedTo(w io.Writer, indexes types.NamedIndices) (int64, error) {
	zw := gzip.NewWriter(w)

	var n int64
	for id, index := range indexes {
		zw.Name = id
		nRead, err := io.Copy(zw, index.Buffer())
		n += nRead
		if err != nil {
			return n, err
		}

		if err := zw.Close(); err != nil {
			return n, err
		}

		zw.Reset(w)
	}

	return n, nil
}

func readGzippedFrom(r io.Reader) (types.NamedIndices, int64, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, 0, err
	}
	indexes := types.NamedIndices{}
	var buf bytes.Buffer
	var n int64
	for {
		zr.Multistream(false)

		nRead, err := io.Copy(&buf, zr)
		n += nRead
		if err != nil {
			return nil, n, err
		}

		ind, err := types.NewCheckpointIndexFromBytes(buf.Bytes())
		if err != nil {
			return nil, n, err
		}
		indexes[zr.Name] = ind

		buf.Reset()
		err = zr.Reset(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, n, err
		}
	}
	if err := zr.Close(); err != nil {
		return nil, n, err
	}
	return indexes, n, nil
}
