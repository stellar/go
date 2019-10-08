// Package ioutil contains a collection of utilities for working with io.Readers.
package ioutil

import (
	"io"
	"io/ioutil"
)

// ReadAllSafe reads the io.Reader to it's EOF, or until the maxSizeToRead
// bytes have been consumed.
func ReadAllSafe(r io.Reader, maxSizeToRead int64) ([]byte, error) {
	return ioutil.ReadAll(io.LimitReader(r, maxSizeToRead))
}
