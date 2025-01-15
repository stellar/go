package xdr

import "github.com/stellar/go/strkey"

func (n NodeId) GetAddress() (string, bool) {
	switch n.Type {
	case PublicKeyTypePublicKeyTypeEd25519:
		ed, ok := n.GetEd25519()
		if !ok {
			return "", false
		}
		raw := make([]byte, 32)
		copy(raw, ed[:])
		encodedAddress, err := strkey.Encode(strkey.VersionByteAccountID, raw)
		if err != nil {
			return "", false
		}
		return encodedAddress, true
	default:
		return "", false
	}
}
