package xdr

import (
	"errors"
	"fmt"

	"github.com/stellar/go/strkey"
)

// Address returns the strkey encoded form of this MuxedAccount.  This method will
// panic if the MuxedAccount is backed by a public key of an unknown type.
func (m *MuxedAccount) Address() string {
	address, err := m.GetAddress()
	if err != nil {
		panic(err)
	}
	return address
}

// GetAddress returns the strkey encoded form of this MuxedAccount, and an error
// if the MuxedAccount is backed by a public key of an unknown type.
func (m *MuxedAccount) GetAddress() (string, error) {
	if m == nil {
		return "", nil
	}

	raw := make([]byte, 0, 40)
	switch m.Type {
	case CryptoKeyTypeKeyTypeEd25519:
		ed, ok := m.GetEd25519()
		if !ok {
			return "", fmt.Errorf("Could not get Ed25519")
		}
		raw = append(raw, ed[:]...)
		return strkey.Encode(strkey.VersionByteAccountID, raw)
	case CryptoKeyTypeKeyTypeMuxedEd25519:
		ed, ok := m.GetMed25519()
		if !ok {
			return "", fmt.Errorf("Could not get Med25519")
		}
		idBytes, err := ed.Id.MarshalBinary()
		if err != nil {
			return "", fmt.Errorf("Could not marshal ID")
		}
		raw = append(raw, idBytes...)
		raw = append(raw, ed.Ed25519[:]...)
		return strkey.Encode(strkey.VersionByteMuxedAccount, raw)
	default:
		return "", fmt.Errorf("Unknown muxed account type: %v", m.Type)
	}

}

// Equals returns true if `other` is equivalent to `m`
func (m *MuxedAccount) Equals(other MuxedAccount) bool {
	if m.Type != other.Type {
		return false
	}

	switch m.Type {
	case CryptoKeyTypeKeyTypeEd25519:
		l := m.MustEd25519()
		r := other.MustEd25519()
		return l == r
	case CryptoKeyTypeKeyTypeMuxedEd25519:
		l := m.MustMed25519()
		r := other.MustMed25519()
		return l == r
	default:
		panic(fmt.Errorf("Unknown muxed account type: %v", m.Type))
	}
}

func MustMuxedAccountAddress(address string) MuxedAccount {
	m := MuxedAccount{}
	err := m.SetAddress(address)
	if err != nil {
		panic(err)
	}
	return m
}

// AddressToMuxedAccount returns an MuxedAccount for a given address string.
// If the address is not valid the error returned will not be nil
func AddressToMuxedAccount(address string) (MuxedAccount, error) {
	result := MuxedAccount{}
	err := result.SetAddress(address)

	return result, err
}

// SetAddress modifies the receiver, setting it's value to the MuxedAccount form
// of the provided address.
func (m *MuxedAccount) SetAddress(address string) error {
	if m == nil {
		return nil
	}

	switch len(address) {
	case 56:
		raw, err := strkey.Decode(strkey.VersionByteAccountID, address)
		if err != nil {
			return err
		}
		if len(raw) != 32 {
			return errors.New("invalid address")
		}
		var ui Uint256
		copy(ui[:], raw)
		*m, err = NewMuxedAccount(CryptoKeyTypeKeyTypeEd25519, ui)
		return err
	case 69:
		raw, err := strkey.Decode(strkey.VersionByteMuxedAccount, address)
		if err != nil {
			return err
		}
		if len(raw) != 40 {
			return errors.New("invalid muxed address")
		}
		var muxed MuxedAccountMed25519
		if err = muxed.Id.UnmarshalBinary(raw[:8]); err != nil {
			return err
		}
		copy(muxed.Ed25519[:], raw[8:])
		*m, err = NewMuxedAccount(CryptoKeyTypeKeyTypeMuxedEd25519, muxed)
		return err
	default:
		return errors.New("invalid address")
	}

}

// ToAccountId transforms a MuxedAccount to an AccountId, dropping the
// memo Id if necessary
func (m MuxedAccount) ToAccountId() AccountId {
	result := AccountId{Type: PublicKeyTypePublicKeyTypeEd25519}
	switch m.Type {
	case CryptoKeyTypeKeyTypeEd25519:
		ed := m.MustEd25519()
		result.Ed25519 = &ed
	case CryptoKeyTypeKeyTypeMuxedEd25519:
		ed := m.MustMed25519().Ed25519
		result.Ed25519 = &ed
	default:
		panic(fmt.Errorf("Unknown muxed account type: %v", m.Type))
	}
	return result
}
