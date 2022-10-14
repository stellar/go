package xdr

import "bytes"

func (s ScBigInt) Equals(o ScBigInt) bool {
	if s.Sign != o.Sign {
		return false
	}

	switch s.Sign {
	case ScNumSignZero:
		return true
	case ScNumSignNegative, ScNumSignPositive:
		return bytes.Equal(s.MustMagnitude(), o.MustMagnitude())
	default:
		panic("unknown Sign type: " + s.Sign.String())
	}
}

func (s ScContractCode) Equals(o ScContractCode) bool {
	if s.Type != o.Type {
		return false
	}
	switch s.Type {
	case ScContractCodeTypeSccontractCodeToken:
		return true
	case ScContractCodeTypeSccontractCodeWasm:
		return bytes.Equal(s.MustWasm(), o.MustWasm())
	default:
		panic("unknown ScContractCode type: " + s.Type.String())
	}
}

func (s *ScObject) Equals(o *ScObject) bool {
	if (s == nil) != (o == nil) {
		return false
	}
	if s == nil {
		return true
	}

	switch s.Type {
	case ScObjectTypeScoI64:
		return s.MustI64() == o.MustI64()
	case ScObjectTypeScoContractCode:
		return s.MustContractCode().Equals(o.MustContractCode())
	case ScObjectTypeScoAccountId:
		aid := s.MustAccountId()
		return aid.Equals(o.MustAccountId())
	case ScObjectTypeScoBigInt:
		return s.MustBigInt().Equals(o.MustBigInt())
	case ScObjectTypeScoBytes:
		return bytes.Equal(s.MustBin(), o.MustBin())
	case ScObjectTypeScoMap:
		myMap := s.MustMap()
		otherMap := o.MustMap()
		if len(myMap) != len(otherMap) {
			return false
		}
		for i := range myMap {
			if !myMap[i].Key.Equals(otherMap[i].Key) ||
				!myMap[i].Val.Equals(otherMap[i].Val) {
				return false
			}
		}
		return true
	case ScObjectTypeScoU64:
		return s.MustU64() == o.MustU64()
	case ScObjectTypeScoVec:
		myVec := s.MustVec()
		otherVec := o.MustVec()
		if len(myVec) != len(otherVec) {
			return false
		}
		for i := range myVec {
			if !myVec[i].Equals(otherVec[i]) {
				return false
			}
		}
		return true
	default:
		panic("unknown ScObject type: " + s.Type.String())
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
	default:
		panic("unknown ScStatus type: " + s.Type.String())
	}
}

func (s ScVal) Equals(o ScVal) bool {
	if s.Type != o.Type {
		return false
	}

	switch s.Type {
	case ScValTypeScvObject:
		return s.MustObj().Equals(o.MustObj())
	case ScValTypeScvBitset:
		return s.MustBits() == o.MustBits()
	case ScValTypeScvStatic:
		return s.MustIc() == o.MustIc()
	case ScValTypeScvStatus:
		return s.MustStatus().Equals(o.MustStatus())
	case ScValTypeScvSymbol:
		return s.MustSym() == o.MustSym()
	case ScValTypeScvI32:
		return s.MustI32() == o.MustI32()
	case ScValTypeScvU32:
		return s.MustU32() == o.MustU32()
	case ScValTypeScvU63:
		return s.MustU63() == o.MustU63()
	default:
		panic("unknown ScVal type: " + s.Type.String())
	}
}
