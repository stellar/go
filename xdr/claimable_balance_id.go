package xdr

// MarshalBinaryCompress marshals ClaimableBalanceId to []byte but unlike
// MarshalBinary() it removes all unnecessary bytes, exploiting the fact
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
