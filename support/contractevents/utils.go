package contractevents

import (
	"fmt"

	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

var ErrNotBalanceChangeEvent = errors.New("event doesn't represent a balance change")

// MustScAddressToString converts the low-level `xdr.ScAddress` union into the
// appropriate strkey (contract C... or account ID G...), panicking on any
// error. If the address is a nil pointer, this returns the empty string.
func MustScAddressToString(address *xdr.ScAddress) string {
	str, err := ScAddressToString(address)
	if err != nil {
		panic(err)
	}
	return str
}

func ScAddressToString(address *xdr.ScAddress) (string, error) {
	if address == nil {
		return "", nil
	}

	var result string
	var err error

	switch address.Type {
	case xdr.ScAddressTypeScAddressTypeAccount:
		pubkey := address.MustAccountId().Ed25519
		result, err = strkey.Encode(strkey.VersionByteAccountID, pubkey[:])
	case xdr.ScAddressTypeScAddressTypeContract:
		contractId := *address.ContractId
		result, err = strkey.Encode(strkey.VersionByteContract, contractId[:])
	default:
		return "", fmt.Errorf("unfamiliar address type: %v", address.Type)
	}

	if err != nil {
		return "", err
	}

	return result, nil
}

func parseAddress(val *xdr.ScVal) *xdr.ScAddress {
	if val == nil {
		return nil
	}

	address, ok := val.GetObj()
	if !ok || address == nil || address.Type != xdr.ScObjectTypeScoAddress {
		return nil
	}

	return address.Address
}

func parseAmount(val *xdr.ScVal) *xdr.Int128Parts {
	valueObj, ok := val.GetObj()
	if !ok || valueObj == nil || valueObj.Type != xdr.ScObjectTypeScoI128 {
		return nil
	}

	return valueObj.I128
}

// parseBalanceChangeEvent is a generalization of a subset of the Stellar Asset
// Contract events. Transfer, mint, clawback, and burn events all have two
// addresses and an amount involved. The addresses represent different things in
// different event types (e.g. "from" or "admin"), but the parsing is identical.
// This helper extracts all three parts or returns a generic error if it can't.
func parseBalanceChangeEvent(topics xdr.ScVec, value xdr.ScVal) (string, string, xdr.Int128Parts, error) {
	first, second, amount := "", "", xdr.Int128Parts{}

	if len(topics) != 4 {
		return first, second, amount, ErrNotBalanceChangeEvent
	}

	rawFirst, rawSecond := topics[1], topics[2]
	firstSc, secondSc := parseAddress(&rawFirst), parseAddress(&rawSecond)
	if firstSc == nil || secondSc == nil {
		return first, second, amount, ErrNotBalanceChangeEvent
	}

	first, second = MustScAddressToString(firstSc), MustScAddressToString(secondSc)

	amountPtr := parseAmount(&value)
	if amountPtr == nil {
		return first, second, amount, ErrNotBalanceChangeEvent
	}

	amount = *amountPtr
	return first, second, amount, nil
}
