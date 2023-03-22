package contractevents

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

var ErrNotBurnEvent = errors.New("event is not a valid 'burn' event")

type BurnEvent struct {
	sacEvent

	From   string
	Amount xdr.Int128Parts
}

// parseBurnEvent tries to parse the given topics and value as a SAC "burn"
// event.
//
// Internally, it assumes that the `topics` array has already validated both the
// function name AND the asset <--> contract ID relationship. It will return a
// best-effort parsing even in error cases.
func (event *BurnEvent) parse(topics xdr.ScVec, value xdr.ScVal) error {
	//
	// The burn event format is:
	//
	// 	"burn"  	Symbol
	//  <from>		Address
	// 	<asset>		Bytes
	//
	// 	<amount> 	i128
	//
	// Reference: https://github.com/stellar/rs-soroban-env/blob/main/soroban-env-host/src/native_contract/token/event.rs#L102-L109
	//
	if len(topics) != 3 {
		return ErrNotBurnEvent
	}

	from, ok := topics[1].GetAddress()
	if !ok {
		return ErrNotBurnEvent
	}
	event.From = MustScAddressToString(from)

	amount, ok := value.GetI128()
	if !ok {
		return ErrNotBurnEvent
	}
	event.Amount = amount

	return nil
}
