package contractevents

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

var ErrNotTransferEvent = errors.New("event is an invalid 'transfer' event")

type TransferEvent struct {
	sacEvent

	From   string
	To     string
	Amount xdr.Int128Parts
}

// parseTransferEvent tries to parse the given topics and value as a SAC
// "transfer" event. It assumes that the `topics` array has already validated
// both the function name AND the asset <--> contract ID relationship. It will
// return a best-effort parsing even in error cases.
func (event *TransferEvent) parse(topics xdr.ScVec, value xdr.ScVal) error {
	//
	// The transfer event format is:
	//
	// 	"transfer"  Symbol
	//  <from> 		Address
	//  <to> 		Address
	// 	<asset>		Bytes
	//
	// 	<amount> 	i128
	//
	if len(topics) != 4 {
		return ErrNotTransferEvent
	}

	from, to := topics[1], topics[2]
	if from.Type != xdr.ScValTypeScvObject || to.Type != xdr.ScValTypeScvObject {
		return ErrNotTransferEvent
	}

	fromObj, ok := from.GetObj()
	if !ok || fromObj == nil || fromObj.Type != xdr.ScObjectTypeScoAddress {
		return ErrNotTransferEvent
	}

	toObj, ok := to.GetObj()
	if !ok || toObj == nil || toObj.Type != xdr.ScObjectTypeScoAddress {
		return ErrNotTransferEvent
	}

	event.From = ScAddressToString(fromObj.Address)
	event.To = ScAddressToString(toObj.Address)
	event.Asset = xdr.Asset{} // TODO

	valueObj, ok := value.GetObj()
	if !ok || valueObj == nil || valueObj.Type != xdr.ScObjectTypeScoI128 {
		return ErrNotTransferEvent
	}

	event.Amount = *valueObj.I128
	return nil
}
