package httpjson

import "github.com/stellar/go/support/errors"

// ErrNotObject is returned when Object.UnmarshalJSON is called
// with bytes not representing a valid json object.
// A valid json object means it starts with `null` or `{`, not `[`.
var ErrNotJsonObject = errors.New("input is not a json object")

type Object []byte

func (o Object) MarshalJSON() ([]byte, error) {
	if len(o) == 0 {
		return []byte("{}"), nil
	}
	return o, nil
}

func (o *Object) UnmarshalJSON(in []byte) error {
	var first byte
	for _, c := range in {
		if !isSpace(c) {
			first = c
			break
		}
	}
	// input does not start with 'n' ("null") or '{'
	if first != 'n' && first != '{' {
		return ErrNotJsonObject
	}

	*o = in
	return nil
}

// https://github.com/golang/go/blob/9f193fbe31d7ffa5f6e71a6387cbcf4636306660/src/encoding/json/scanner.go#L160-L162
func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}
