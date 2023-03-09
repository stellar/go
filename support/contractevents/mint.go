package contractevents

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

var ErrNotMintEvent = errors.New("event is not a valid 'mint' event")

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

	rawAdmin, rawTo := topics[1], topics[2]
	admin, to := parseAddress(&rawAdmin), parseAddress(&rawTo)
	if admin == nil || to == nil {
		return ErrNotMintEvent
	}

	event.Admin = MustScAddressToString(admin)
	event.To = MustScAddressToString(to)

	valueObj, ok := value.GetObj()
	if !ok || valueObj == nil || valueObj.Type != xdr.ScObjectTypeScoI128 {
		return ErrNotMintEvent
	}

	event.Amount = *valueObj.I128
	return nil
}
