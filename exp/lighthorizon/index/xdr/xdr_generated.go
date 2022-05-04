//lint:file-ignore S1005 The issue should be fixed in xdrgen. Unfortunately, there's no way to ignore a single file in staticcheck.
//lint:file-ignore U1000 fmtTest is not needed anywhere, should be removed in xdrgen.

// Package xdr is generated from:
//
//  xdr/LightHorizon-types.x
//
// DO NOT EDIT or your changes may be overwritten
package xdr

import (
	"bytes"
	"encoding"
	"fmt"
	"io"

	"github.com/stellar/go-xdr/xdr3"
)

type xdrType interface {
	xdrType()
}

type decoderFrom interface {
	DecodeFrom(d *xdr.Decoder) (int, error)
}

// Unmarshal reads an xdr element from `r` into `v`.
func Unmarshal(r io.Reader, v interface{}) (int, error) {
	if decodable, ok := v.(decoderFrom); ok {
		d := xdr.NewDecoder(r)
		return decodable.DecodeFrom(d)
	}
	// delegate to xdr package's Unmarshal
	return xdr.Unmarshal(r, v)
}

// Marshal writes an xdr element `v` into `w`.
func Marshal(w io.Writer, v interface{}) (int, error) {
	if _, ok := v.(xdrType); ok {
		if bm, ok := v.(encoding.BinaryMarshaler); ok {
			b, err := bm.MarshalBinary()
			if err != nil {
				return 0, err
			}
			return w.Write(b)
		}
	}
	// delegate to xdr package's Marshal
	return xdr.Marshal(w, v)
}

// Uint32 is an XDR Typedef defines as:
//
//   typedef unsigned int uint32;
//
type Uint32 uint32

// EncodeTo encodes this value using the Encoder.
func (s Uint32) EncodeTo(e *xdr.Encoder) error {
	var err error
	if _, err = e.EncodeUint(uint32(s)); err != nil {
		return err
	}
	return nil
}

var _ decoderFrom = (*Uint32)(nil)

// DecodeFrom decodes this value using the Decoder.
func (s *Uint32) DecodeFrom(d *xdr.Decoder) (int, error) {
	var err error
	var n, nTmp int
	var v uint32
	v, nTmp, err = d.DecodeUint()
	n += nTmp
	if err != nil {
		return n, fmt.Errorf("decoding Unsigned int: %s", err)
	}
	*s = Uint32(v)
	return n, nil
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (s Uint32) MarshalBinary() ([]byte, error) {
	b := bytes.Buffer{}
	e := xdr.NewEncoder(&b)
	err := s.EncodeTo(e)
	return b.Bytes(), err
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (s *Uint32) UnmarshalBinary(inp []byte) error {
	r := bytes.NewReader(inp)
	d := xdr.NewDecoder(r)
	_, err := s.DecodeFrom(d)
	return err
}

var (
	_ encoding.BinaryMarshaler   = (*Uint32)(nil)
	_ encoding.BinaryUnmarshaler = (*Uint32)(nil)
)

// xdrType signals that this type is an type representing
// representing XDR values defined by this package.
func (s Uint32) xdrType() {}

var _ xdrType = (*Uint32)(nil)

// Value is an XDR Typedef defines as:
//
//   typedef opaque Value<>;
//
type Value []byte

// EncodeTo encodes this value using the Encoder.
func (s Value) EncodeTo(e *xdr.Encoder) error {
	var err error
	if _, err = e.EncodeOpaque(s[:]); err != nil {
		return err
	}
	return nil
}

var _ decoderFrom = (*Value)(nil)

// DecodeFrom decodes this value using the Decoder.
func (s *Value) DecodeFrom(d *xdr.Decoder) (int, error) {
	var err error
	var n, nTmp int
	(*s), nTmp, err = d.DecodeOpaque(0)
	n += nTmp
	if err != nil {
		return n, fmt.Errorf("decoding Value: %s", err)
	}
	return n, nil
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (s Value) MarshalBinary() ([]byte, error) {
	b := bytes.Buffer{}
	e := xdr.NewEncoder(&b)
	err := s.EncodeTo(e)
	return b.Bytes(), err
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (s *Value) UnmarshalBinary(inp []byte) error {
	r := bytes.NewReader(inp)
	d := xdr.NewDecoder(r)
	_, err := s.DecodeFrom(d)
	return err
}

var (
	_ encoding.BinaryMarshaler   = (*Value)(nil)
	_ encoding.BinaryUnmarshaler = (*Value)(nil)
)

// xdrType signals that this type is an type representing
// representing XDR values defined by this package.
func (s Value) xdrType() {}

var _ xdrType = (*Value)(nil)

// CheckpointIndex is an XDR Struct defines as:
//
//   struct CheckpointIndex {
//        uint32 firstCheckpoint;
//        uint32 lastCheckpoint;
//        Value bitmap;
//    };
//
type CheckpointIndex struct {
	FirstCheckpoint Uint32
	LastCheckpoint  Uint32
	Bitmap          Value
}

// EncodeTo encodes this value using the Encoder.
func (s *CheckpointIndex) EncodeTo(e *xdr.Encoder) error {
	var err error
	if err = s.FirstCheckpoint.EncodeTo(e); err != nil {
		return err
	}
	if err = s.LastCheckpoint.EncodeTo(e); err != nil {
		return err
	}
	if err = s.Bitmap.EncodeTo(e); err != nil {
		return err
	}
	return nil
}

var _ decoderFrom = (*CheckpointIndex)(nil)

// DecodeFrom decodes this value using the Decoder.
func (s *CheckpointIndex) DecodeFrom(d *xdr.Decoder) (int, error) {
	var err error
	var n, nTmp int
	nTmp, err = s.FirstCheckpoint.DecodeFrom(d)
	n += nTmp
	if err != nil {
		return n, fmt.Errorf("decoding Uint32: %s", err)
	}
	nTmp, err = s.LastCheckpoint.DecodeFrom(d)
	n += nTmp
	if err != nil {
		return n, fmt.Errorf("decoding Uint32: %s", err)
	}
	nTmp, err = s.Bitmap.DecodeFrom(d)
	n += nTmp
	if err != nil {
		return n, fmt.Errorf("decoding Value: %s", err)
	}
	return n, nil
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (s CheckpointIndex) MarshalBinary() ([]byte, error) {
	b := bytes.Buffer{}
	e := xdr.NewEncoder(&b)
	err := s.EncodeTo(e)
	return b.Bytes(), err
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (s *CheckpointIndex) UnmarshalBinary(inp []byte) error {
	r := bytes.NewReader(inp)
	d := xdr.NewDecoder(r)
	_, err := s.DecodeFrom(d)
	return err
}

var (
	_ encoding.BinaryMarshaler   = (*CheckpointIndex)(nil)
	_ encoding.BinaryUnmarshaler = (*CheckpointIndex)(nil)
)

// xdrType signals that this type is an type representing
// representing XDR values defined by this package.
func (s CheckpointIndex) xdrType() {}

var _ xdrType = (*CheckpointIndex)(nil)

var fmtTest = fmt.Sprint("this is a dummy usage of fmt")
