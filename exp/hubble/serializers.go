// +build go1.13

package hubble

import (
	"bytes"

	"github.com/stellar/go/xdr"
	goxdr "github.com/xdrpp/goxdr/xdr"
	"github.com/xdrpp/stc/stcdetail"
	"github.com/xdrpp/stc/stx"
)

func serializeLedgerEntryChange(lec xdr.LedgerEntryChange) (string, error) {
	stxlec := stx.LedgerEntryChange{}
	lecBytes, err := lec.MarshalBinary()
	if err != nil {
		return "", err
	}
	stx.XDR_LedgerEntryChange(&stxlec).XdrMarshal(&goxdr.XdrIn{In: bytes.NewReader(lecBytes)}, "")
	lecJSONBytes, err := stcdetail.XdrToJson(&stxlec)
	if err != nil {
		return "", err
	}
	return string(lecJSONBytes), nil
}
