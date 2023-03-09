package contractevents

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

var ErrNotMintEvent = errors.New("event is an invalid 'mint' event")

type MintEvent struct {
	sacEvent

	Admin  string
	To     string
	Amount xdr.Int128Parts
}

// parseTransferEvent tries to parse the given topics and value as a SAC
// "transfer" event. It assumes that the `topics` array has already validated
// both the function name AND the asset <--> contract ID relationship. It will
// return a best-effort parsing even in error cases.
func (event *MintEvent) parse(topics xdr.ScVec, value xdr.ScVal) error {
	//
	// The mint event format is:
	//
	// 	"mint"  	Symbol
	//  <admin>		Address
	//  <to> 		Address
	// 	<asset>		Bytes
	//
	// 	<amount> 	i128
	//
	if len(topics) != 4 {
		return ErrNotTransferEvent
	}

	admin, to := topics[1], topics[2]
	if admin.Type != xdr.ScValTypeScvObject || to.Type != xdr.ScValTypeScvObject {
		return ErrNotMintEvent
	}

	adminObj, ok := admin.GetObj()
	if !ok || adminObj == nil || adminObj.Type != xdr.ScObjectTypeScoAddress {
		return ErrNotMintEvent
	}

	toObj, ok := to.GetObj()
	if !ok || toObj == nil || toObj.Type != xdr.ScObjectTypeScoAddress {
		return ErrNotMintEvent
	}

	event.Admin = ScAddressToString(adminObj.Address)
	event.To = ScAddressToString(toObj.Address)
	event.Asset = xdr.Asset{} // TODO

	valueObj, ok := value.GetObj()
	if !ok || valueObj == nil || valueObj.Type != xdr.ScObjectTypeScoI128 {
		return ErrNotMintEvent
	}

	event.Amount = *valueObj.I128
	return nil
}
