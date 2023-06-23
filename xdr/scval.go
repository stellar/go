package xdr

import (
	"bytes"
	"fmt"

	"github.com/stellar/go/strkey"
)

func (address ScAddress) String() (string, error) {
	var result string
	var err error

	switch address.Type {
	case ScAddressTypeScAddressTypeAccount:
		pubkey := address.MustAccountId().Ed25519
		result, err = strkey.Encode(strkey.VersionByteAccountID, pubkey[:])
	case ScAddressTypeScAddressTypeContract:
		contractID := *address.ContractId
		result, err = strkey.Encode(strkey.VersionByteContract, contractID[:])
	default:
		return "", fmt.Errorf("unfamiliar address type: %v", address.Type)
	}

	if err != nil {
		return "", err
	}

	return result, nil
}

func (s ContractExecutable) Equals(o ContractExecutable) bool {
	if s.Type != o.Type {
		return false
	}
	switch s.Type {
	case ContractExecutableTypeContractExecutableToken:
		return true
	case ContractExecutableTypeContractExecutableWasm:
		return s.MustWasmHash().Equals(o.MustWasmHash())
	default:
		panic("unknown ScContractExecutable type: " + s.Type.String())
	}
}

func (s ScError) Equals(o ScError) bool {
	return s.Type == o.Type && s.Code == o.Code
}

func (s ScVal) Equals(o ScVal) bool {
	if s.Type != o.Type {
		return false
	}

	switch s.Type {
	case ScValTypeScvBool:
		return s.MustB() == o.MustB()
	case ScValTypeScvVoid:
		return true
	case ScValTypeScvError:
		return s.MustError().Equals(o.MustError())
	case ScValTypeScvU32:
		return s.MustU32() == o.MustU32()
	case ScValTypeScvI32:
		return s.MustI32() == o.MustI32()
	case ScValTypeScvU64:
		return s.MustU64() == o.MustU64()
	case ScValTypeScvI64:
		return s.MustI64() == o.MustI64()
	case ScValTypeScvTimepoint:
		return s.MustTimepoint() == o.MustTimepoint()
	case ScValTypeScvDuration:
		return s.MustDuration() == o.MustDuration()
	case ScValTypeScvU128:
		return s.MustU128() == o.MustU128()
	case ScValTypeScvI128:
		return s.MustI128() == o.MustI128()
	case ScValTypeScvU256:
		return s.MustU256() == o.MustU256()
	case ScValTypeScvI256:
		return s.MustI256() == o.MustI256()
	case ScValTypeScvBytes:
		return s.MustBytes().Equals(o.MustBytes())
	case ScValTypeScvString:
		return s.MustStr() == o.MustStr()
	case ScValTypeScvSymbol:
		return s.MustSym() == o.MustSym()
	case ScValTypeScvVec:
		return s.MustVec().Equals(o.MustVec())
	case ScValTypeScvMap:
		return s.MustMap().Equals(o.MustMap())
	case ScValTypeScvAddress:
		return s.MustAddress().Equals(o.MustAddress())
	case ScValTypeScvContractInstance:
		return s.MustInstance().Executable.Equals(o.MustInstance().Executable) && s.MustInstance().Storage.Equals(o.MustInstance().Storage)
	case ScValTypeScvLedgerKeyContractInstance:
		return true
	case ScValTypeScvLedgerKeyNonce:
		return s.MustNonceKey().Equals(o.MustNonceKey())

	default:
		panic("unknown ScVal type: " + s.Type.String())
	}
}

func (s ScBytes) Equals(o ScBytes) bool {
	return bytes.Equal([]byte(s), []byte(o))
}

func (s ScAddress) Equals(o ScAddress) bool {
	if s.Type != o.Type {
		return false
	}

	switch s.Type {
	case ScAddressTypeScAddressTypeAccount:
		sAccountID := s.MustAccountId()
		return sAccountID.Equals(o.MustAccountId())
	case ScAddressTypeScAddressTypeContract:
		return s.MustContractId() == o.MustContractId()
	default:
		panic("unknown ScAddress type: " + s.Type.String())
	}
}

// IsBool returns true if the given ScVal is a boolean
func (s ScVal) IsBool() bool {
	return s.Type == ScValTypeScvBool
}

func (s *ScVec) Equals(o *ScVec) bool {
	if s == nil && o == nil {
		return true
	}
	if s == nil || o == nil {
		return false
	}
	if len(*s) != len(*o) {
		return false
	}
	for i := range *s {
		if !(*s)[i].Equals((*o)[i]) {
			return false
		}
	}
	return true
}

func (s *ScMap) Equals(o *ScMap) bool {
	if s == nil && o == nil {
		return true
	}
	if s == nil || o == nil {
		return false
	}
	if len(*s) != len(*o) {
		return false
	}
	for i, entry := range *s {
		if !entry.Equals((*o)[i]) {
			return false
		}
	}
	return true
}

func (s ScMapEntry) Equals(o ScMapEntry) bool {
	return s.Key.Equals(o.Key) && s.Val.Equals(o.Val)
}

func (s ScNonceKey) Equals(o ScNonceKey) bool {
	return s.Nonce == o.Nonce
}
