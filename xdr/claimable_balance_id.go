package xdr

import (
	"errors"
	"fmt"
)

// MarshalBinaryCompress marshals ClaimableBalanceId to []byte but unlike
// MarshalBinary() it removes all unnecessary bytes, exploting the fact
// that XDR is padding data to 4 bytes in union discriminants etc.
// It's primary use is in ingest/io.StateReader that keep LedgerKeys in
// memory so this function decrease memory requirements.
//
// Warning, do not use UnmarshalBinary() on data encoded using this method!
func (cb ClaimableBalanceId) MarshalBinaryCompress() ([]byte, error) {
	m := []byte{byte(cb.Type)}

	switch cb.Type {
	case ClaimableBalanceIdTypeClaimableBalanceIdTypeV0:
		hash, err := cb.V0.MarshalBinary()
		if err != nil {
			return nil, err
		}
		m = append(m, hash...)
	default:
		panic("Unknown type")
	}

	return m, nil
}

// String returns a display friendly version of the balanceID.
// The first digit represents the union discriminant and the rest is the hex
// encoded value of the union's body.
func (cb ClaimableBalanceId) String() (string, error) {
	var balanceID string
	switch cb.Type {
	case ClaimableBalanceIdTypeClaimableBalanceIdTypeV0:
		body, err := MarshalHex(cb.V0)
		if err != nil {
			return "", err
		}

		balanceID = fmt.Sprintf("%x%s", int(ClaimableBalanceIdTypeClaimableBalanceIdTypeV0), body)
	default:
		return "", errors.New("Unknown type")
	}

	return balanceID, nil
}
