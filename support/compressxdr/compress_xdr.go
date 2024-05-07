package compressxdr

import (
	"io"

	"github.com/stellar/go/support/errors"
)

const (
	GZIP = "gzip"
)

type XDREncoder interface {
	WriteTo(w io.Writer) (int64, error)
}

type XDRDecoder interface {
	ReadFrom(r io.Reader) (int64, error)
	Unzip(r io.Reader) ([]byte, error)
}

func NewXDREncoder(compressionType string, xdrPayload interface{}) (XDREncoder, error) {
	switch compressionType {
	case GZIP:
		return &XDRGzipEncoder{XdrPayload: xdrPayload}, nil
	default:
		return nil, errors.Errorf("invalid compression type %s, not supported", compressionType)
	}
}

func NewXDRDecoder(compressionType string, xdrPayload interface{}) (XDRDecoder, error) {
	switch compressionType {
	case GZIP:
		return &XDRGzipDecoder{XdrPayload: xdrPayload}, nil
	default:
		return nil, errors.Errorf("invalid compression type %s, not supported", compressionType)
	}
}
