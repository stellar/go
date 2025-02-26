package xdr

import "fmt"

func (e *EncodingBuffer) claimableBalanceCompressEncodeTo(cb ClaimableBalanceId) error {
	if err := e.xdrEncoderBuf.WriteByte(byte(cb.Type)); err != nil {
		return err
	}
	switch cb.Type {
	case ClaimableBalanceIdTypeClaimableBalanceIdTypeV0:
		_, err := e.xdrEncoderBuf.Write(cb.V0[:])
		return err
	default:
		panic("Unknown type")
	}
}

func (a *ClaimableBalanceId) Equals(b ClaimableBalanceId) bool {
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case ClaimableBalanceIdTypeClaimableBalanceIdTypeV0:
		return a.MustV0().Equals(b.MustV0())
	default:
		panic(fmt.Errorf("Unknown claimable balance type: %v", a.Type))
	}
}
