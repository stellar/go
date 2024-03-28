package ledgerexporter

import (
	"compress/gzip"
	"io"

	xdr3 "github.com/stellar/go-xdr/xdr3"
)

type XDRGzipEncoder struct {
	XdrPayload interface{}
}

func (g *XDRGzipEncoder) WriteTo(w io.Writer) (int64, error) {
	gw := gzip.NewWriter(w)
	n, err := xdr3.Marshal(gw, g.XdrPayload)
	if err != nil {
		return int64(n), err
	}
	return int64(n), gw.Close()
}

type XDRGzipDecoder struct {
	XdrPayload interface{}
}

func (d *XDRGzipDecoder) ReadFrom(r io.Reader) (int64, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return 0, err
	}
	defer gr.Close()

	n, err := xdr3.Unmarshal(gr, d.XdrPayload)
	if err != nil {
		return int64(n), err
	}
	return int64(n), nil
}
