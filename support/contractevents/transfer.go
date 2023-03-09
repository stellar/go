package contractevents

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

var ErrNotTransferEvent = errors.New("event is not a valid 'transfer' event")

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

	rawFrom, rawTo := topics[1], topics[2]
	from, to := parseAddress(&rawFrom), parseAddress(&rawTo)
	if from == nil || to == nil {
		return ErrNotTransferEvent
	}

	event.From = MustScAddressToString(from)
	event.To = MustScAddressToString(to)

	valueObj, ok := value.GetObj()
	if !ok || valueObj == nil || valueObj.Type != xdr.ScObjectTypeScoI128 {
		return ErrNotTransferEvent
	}

	event.Amount = *valueObj.I128
	return nil
}
