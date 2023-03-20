package xdr

import (
	"bytes"
)

func (s ScContractExecutable) Equals(o ScContractExecutable) bool {
	if s.Type != o.Type {
		return false
	}
	switch s.Type {
	case ScContractExecutableTypeSccontractExecutableToken:
		return true
	case ScContractExecutableTypeSccontractExecutableWasmRef:
		return s.MustWasmId().Equals(o.MustWasmId())
	default:
		panic("unknown ScContractExecutable type: " + s.Type.String())
	}
}

func (s ScStatus) Equals(o ScStatus) bool {
	if s.Type != o.Type {
		return false
	}

	switch s.Type {
	case ScStatusTypeSstContractError:
		return s.MustContractCode() == o.MustContractCode()
	case ScStatusTypeSstHostFunctionError:
		return s.MustFnCode() == o.MustFnCode()
	case ScStatusTypeSstHostObjectError:
		return s.MustObjCode() == o.MustObjCode()
	case ScStatusTypeSstHostContextError:
		return s.MustContextCode() == o.MustContextCode()
	case ScStatusTypeSstHostStorageError:
		return s.MustStorageCode() == o.MustStorageCode()
	case ScStatusTypeSstHostValueError:
		return s.MustValCode() == o.MustValCode()
	case ScStatusTypeSstOk:
		return true
	case ScStatusTypeSstVmError:
		return s.MustVmCode() == o.MustVmCode()
	case ScStatusTypeSstUnknownError:
		return s.MustUnknownCode() == o.MustUnknownCode()
	case ScStatusTypeSstHostAuthError:
		return s.MustAuthCode() == o.MustAuthCode()
	default:
		panic("unknown ScStatus type: " + s.Type.String())
	}
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
	case ScValTypeScvStatus:
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
	case ScValTypeScvContractExecutable:
		return s.MustExec().Equals(o.MustExec())
	case ScValTypeScvAddress:
		return s.MustAddress().Equals(o.MustAddress())
	case ScValTypeScvLedgerKeyContractExecutable:
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
	return s.NonceAddress.Equals(o.NonceAddress)
}
