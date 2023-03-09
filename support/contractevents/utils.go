package contractevents

import (
	"fmt"

	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

// MustScAddressToString converts the low-level `xdr.ScAddress` union into the
// appropriate strkey (contract C... or account ID G...).
func MustScAddressToString(address *xdr.ScAddress) string {
	if address == nil {
		return ""
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
		panic(fmt.Errorf("unfamiliar address type: %v", address.Type))
	}

	if err != nil {
		panic(err)
	}

	return result
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
