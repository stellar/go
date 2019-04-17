package txnbuild

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

type MemoText string
type MemoID uint64
type MemoHash [32]byte
type MemoReturn [32]byte

const MemoTextMaxLength = 28

type Memo interface {
	ToXDR() (xdr.Memo, error)
}

func (mt MemoText) ToXDR() (xdr.Memo, error) {
	if len(mt) > MemoTextMaxLength {
		return xdr.Memo{}, fmt.Errorf("Memo text too long (more than %d bytes", MemoTextMaxLength)
	}

	return xdr.NewMemo(xdr.MemoTypeMemoText, string(mt))
}
