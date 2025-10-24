package xdr

import (
	"encoding/hex"

	"github.com/stellar/go/strkey"
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

// ToContractAddress converts a Hash to a Stellar contract address string.
// Returns the contract address in strkey format (C...) and an error if encoding fails.
func (h Hash) ToContractAddress() (string, error) {
	return strkey.Encode(strkey.VersionByteContract, h[:])
}
