// +build go1.13

package hubble

import (
	"bytes"

	"github.com/stellar/go/xdr"
	goxdr "github.com/xdrpp/goxdr/xdr"
	"github.com/xdrpp/stc/stcdetail"
	"github.com/xdrpp/stc/stx"
)

func serializeLedgerEntryChange(lec xdr.LedgerEntryChange) ([]byte, error) {
	stxlec := stx.LedgerEntryChange{}
	lecbytes, err := lec.MarshalBinary()
	if err != nil {
		return []byte{}, err
	}
	stx.XDR_LedgerEntryChange(&stxlec).XdrMarshal(&goxdr.XdrIn{In: bytes.NewReader(lecbytes)}, "")
	j, err := stcdetail.XdrToJson(&stxlec)
	if err != nil {
		return []byte{}, err
	}
	return j, nil
}
