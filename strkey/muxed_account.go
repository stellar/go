package strkey

import (
	"bytes"
	"fmt"

	xdr "github.com/stellar/go-xdr/xdr3"
	"github.com/stellar/go/support/errors"
)

type MuxedAccount struct {
	id      uint64
	ed25519 [32]byte
}

// SetID populates the muxed account ID.
func (m *MuxedAccount) SetID(id uint64) {
	m.id = id
}

// SetAddress populates the muxed account ed25519 address.
func (m *MuxedAccount) SetAddress(address string) error {
	if !IsValidEd25519PublicKey(address) {
		return errors.New("invalid ed25519 public key")
	}

	raw, err := Decode(VersionByteAccountID, address)
	if err != nil {
		return errors.Wrap(err, "decoding ed25519 address")
	}
	if len(raw) != 32 {
		return fmt.Errorf("invalid binary length: %d", len(raw))
	}

	copy(m.ed25519[:], raw)

	return nil
}

// ID returns the muxed account id according with SEP-23 definition for
// multiplexed accounts.
func (m *MuxedAccount) ID() uint64 {
	return m.id
}

// Address returns the muxed account G-address according with SEP-23 definition
// for multiplexed accounts.
func (m *MuxedAccount) Address() (string, error) {
	if m.ed25519 == [32]byte{} {
		return "", errors.New("muxed account has no ed25519 key")
	}

	return Encode(VersionByteAccountID, m.ed25519[:])
}

// MuxedAddress returns the muxed account M-address according with SEP-23
// definition for multiplexed accounts.
func (m *MuxedAccount) MuxedAddress() (string, error) {
	if m.ed25519 == [32]byte{} {
		return "", errors.New("muxed account has no ed25519 key")
	}

	b := new(bytes.Buffer)
	_, err := xdr.Marshal(b, m.id)
	if err != nil {
		return "", errors.Wrap(err, "marshaling muxed address id")
	}

	raw := make([]byte, 0, 40)
	raw = append(raw, m.ed25519[:]...)
	raw = append(raw, b.Bytes()...)

	return Encode(VersionByteMuxedAccount, raw)
}
