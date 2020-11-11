// Package xdr contains the generated code for parsing the xdr structures used
// for stellar.
package xdr

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	xdr "github.com/stellar/go-xdr/xdr3"
	"github.com/stellar/go/support/errors"
)

// Keyer represents a type that can be converted into a LedgerKey
type Keyer interface {
	LedgerKey() LedgerKey
}

var _ = LedgerEntry{}
var _ = LedgerKey{}

func Uint32Ptr(val uint32) *Uint32 {
	pval := Uint32(val)
	return &pval
}

func safeUnmarshalString(decoder func(reader io.Reader) io.Reader, data string, dest interface{}) error {
	count := &countWriter{}
	l := len(data)

	_, err := Unmarshal(decoder(io.TeeReader(strings.NewReader(data), count)), dest)
	if err != nil {
		return err
	}

	if count.Count != l {
		return fmt.Errorf("input not fully consumed. expected to read: %d, actual: %d", l, count.Count)
	}

	return nil
}

// SafeUnmarshalBase64 first decodes the provided reader from base64 before
// decoding the xdr into the provided destination.  Also ensures that the reader
// is fully consumed.
func SafeUnmarshalBase64(data string, dest interface{}) error {
	return safeUnmarshalString(
		func(r io.Reader) io.Reader {
			return base64.NewDecoder(base64.StdEncoding, r)
		},
		data,
		dest,
	)
}

// SafeUnmarshalHex first decodes the provided reader from hex before
// decoding the xdr into the provided destination.  Also ensures that the reader
// is fully consumed.
func SafeUnmarshalHex(data string, dest interface{}) error {
	return safeUnmarshalString(hex.NewDecoder, data, dest)
}

// SafeUnmarshal decodes the provided reader into the destination and verifies
// that provided bytes are all consumed by the unmarshalling process.
func SafeUnmarshal(data []byte, dest interface{}) error {
	r := bytes.NewReader(data)
	n, err := Unmarshal(r, dest)

	if err != nil {
		return err
	}

	if n != len(data) {
		return fmt.Errorf("input not fully consumed. expected to read: %d, actual: %d", len(data), n)
	}

	return nil
}

func marshalString(encoder func([]byte) string, v interface{}) (string, error) {
	var raw bytes.Buffer

	_, err := Marshal(&raw, v)

	if err != nil {
		return "", err
	}

	return encoder(raw.Bytes()), nil
}

func MarshalBase64(v interface{}) (string, error) {
	return marshalString(base64.StdEncoding.EncodeToString, v)
}

func MarshalHex(v interface{}) (string, error) {
	return marshalString(hex.EncodeToString, v)
}

func MarshalFramed(w io.Writer, v interface{}) error {
	var tmp bytes.Buffer
	n, err := Marshal(&tmp, v)
	if err != nil {
		return err
	}
	un := uint32(n)
	if un > 0x7fffffff {
		return fmt.Errorf("Overlong write: %d bytes", n)
	}

	un = un | 0x80000000
	err = binary.Write(w, binary.BigEndian, &un)
	if err != nil {
		return errors.Wrap(err, "error in binary.Write")
	}
	k, err := tmp.WriteTo(w)
	if int64(n) != k {
		return fmt.Errorf("Mismatched write length: %d vs. %d", n, k)
	}
	return err
}

// ReadFrameLength returns a length of a framed XDR object.
func ReadFrameLength(r io.Reader) (uint32, error) {
	var frameLen uint32
	n, e := Unmarshal(r, &frameLen)
	if e != nil {
		return 0, errors.Wrap(e, "unmarshalling XDR frame header")
	}
	if n != 4 {
		return 0, errors.New("bad length of XDR frame header")
	}
	if (frameLen & 0x80000000) != 0x80000000 {
		return 0, errors.New("malformed XDR frame header")
	}
	frameLen &= 0x7fffffff
	return frameLen, nil
}

// XDR and RPC define a (minimal) framing format which our metadata arrives in: a 4-byte
// big-endian length header that has the high bit set, followed by that length worth of
// XDR data. Decoding this involves just a little more work than xdr.Unmarshal.
func UnmarshalFramed(r io.Reader, v interface{}) (int, error) {
	frameLen, err := ReadFrameLength(r)
	if err != nil {
		return 0, errors.Wrap(err, "unmarshalling XDR frame header")
	}
	m, err := xdr.Unmarshal(r, v)
	if err != nil {
		return 0, errors.Wrap(err, "unmarshalling framed XDR")
	}
	if int64(m) != int64(frameLen) {
		return 0, errors.New("bad length of XDR frame body")
	}
	return m + 4 /* frame size: uint32 */, nil
}

type countWriter struct {
	Count int
}

func (w *countWriter) Write(d []byte) (int, error) {
	l := len(d)
	w.Count += l
	return l, nil
}
