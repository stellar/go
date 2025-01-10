package xdr

import (
	"encoding/base64"
	"encoding/hex"
)

func (h Hash) HexString() string {
	return hex.EncodeToString(h[:])
}

func (s Hash) Equals(o Hash) bool {
	if len(s) != len(o) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] != o[i] {
			return false
		}
	}
	return true
}

// MarshalBinaryBase64 marshals XDR into a binary form and then encodes it
// using base64.
func (h Hash) MarshalBinaryBase64() (string, error) {
	b, err := h.MarshalBinary()
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}
