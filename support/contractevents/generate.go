package contractevents

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"

	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

// GenerateEvent is a utility function to be used by testing frameworks in order
// to generate Stellar Asset Contract events.
//
// To provide a generic interface, there are more arguments than apply to the
// type, but you should only expect the relevant ones to be set (for example,
// transfer events have no admin, so it will be ignored). This means you can
// always pass your set of testing parameters, modify the type, and get the
// event filled out with the details you expect.
func GenerateEvent(
	type_ EventType,
	from, to, admin string,
	asset xdr.Asset,
	amount *big.Int,
	passphrase string,
) xdr.ContractEvent {
	var topics []xdr.ScVal
	data := makeBigAmount(amount)

	switch type_ {
	case EventTypeTransfer:
		topics = []xdr.ScVal{
			makeSymbol("transfer"),
			makeAddress(from),
			makeAddress(to),
			makeAsset(asset),
		}

	case EventTypeMint:
		topics = []xdr.ScVal{
			makeSymbol("mint"),
			makeAddress(admin),
			makeAddress(to),
			makeAsset(asset),
		}

	case EventTypeClawback:
		topics = []xdr.ScVal{
			makeSymbol("clawback"),
			makeAddress(admin),
			makeAddress(from),
			makeAsset(asset),
		}

	case EventTypeBurn:
		topics = []xdr.ScVal{
			makeSymbol("burn"),
			makeAddress(from),
			makeAsset(asset),
		}

	default:
		panic(fmt.Errorf("event type %v unsupported", type_))
	}

	rawContractId, err := asset.ContractID(passphrase)
	if err != nil {
		panic(err)
	}
	contractId := xdr.Hash(rawContractId)

	event := xdr.ContractEvent{
		Type:       xdr.ContractEventTypeContract,
		ContractId: &contractId,
		Body: xdr.ContractEventBody{
			V: 0,
			V0: &xdr.ContractEventV0{
				Topics: xdr.ScVec(topics),
				Data:   data,
			},
		},
	}

	return event
}

func contractIdToHash(contractId string) *xdr.Hash {
	idBytes := [32]byte{}
	rawBytes, err := hex.DecodeString(contractId)
	if err != nil {
		panic(fmt.Errorf("invalid contract id (%s): %v", contractId, err))
	}
	if copy(idBytes[:], rawBytes[:]) != 32 {
		panic("couldn't copy 32 bytes to contract hash")
	}

	hash := xdr.Hash(idBytes)
	return &hash
}

func makeSymbol(sym string) xdr.ScVal {
	symbol := xdr.ScSymbol(sym)
	return xdr.ScVal{
		Type: xdr.ScValTypeScvSymbol,
		Sym:  &symbol,
	}
}

func makeBigAmount(amount *big.Int) xdr.ScVal {
	// TODO: Better check, as MaxUint128 shouldn't be allowed
	if amount.BitLen() > 128 {
		panic(fmt.Errorf(
			"amount is too large: %d bits (max 128)",
			amount.BitLen()))
	}

	//
	// We create the two Uint64 parts as follows:
	//
	//  - take the upper 64 by shifting 64 right
	//  - take the lower 64 by zeroing the top 64
	//
	keepLower := big.NewInt(0).SetUint64(math.MaxUint64)

	hi := new(big.Int).Rsh(amount, 64)
	lo := amount.And(amount, keepLower)

	amountObj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoI128,
		I128: &xdr.Int128Parts{
			Lo: xdr.Uint64(lo.Uint64()),
			Hi: xdr.Uint64(hi.Uint64()),
		},
	}

	return xdr.ScVal{
		Type: xdr.ScValTypeScvObject,
		Obj:  &amountObj,
	}
}

func makeAddress(address string) xdr.ScVal {
	scAddress := xdr.ScAddress{}
	scObject := &xdr.ScObject{
		Type:    xdr.ScObjectTypeScoAddress,
		Address: &scAddress,
	}

	switch address[0] {
	case 'C':
		scAddress.Type = xdr.ScAddressTypeScAddressTypeContract
		contractHash := strkey.MustDecode(strkey.VersionByteContract, address)
		scAddress.ContractId = contractIdToHash(hex.EncodeToString(contractHash))

	case 'G':
		scAddress.Type = xdr.ScAddressTypeScAddressTypeAccount
		scAddress.AccountId = xdr.MustAddressPtr(address)

	default:
		panic(fmt.Errorf("unsupported address: %s", address))
	}

	return xdr.ScVal{
		Type: xdr.ScValTypeScvObject,
		Obj:  &scObject,
	}
}

func makeAsset(asset xdr.Asset) xdr.ScVal {
	buffer := new(bytes.Buffer)

	switch asset.Type {
	case xdr.AssetTypeAssetTypeNative:
		_, err := buffer.WriteString("native")
		if err != nil {
			panic(err)
		}

	case xdr.AssetTypeAssetTypeCreditAlphanum4:
		_, err := xdr.Marshal(buffer, asset.AlphaNum4.AssetCode)
		if err != nil {
			panic(err)
		}
		pubkey, ok := asset.AlphaNum4.Issuer.GetEd25519()
		if !ok {
			panic("issuer not a public key")
		}
		buffer.WriteString(":")
		_, err = xdr.Marshal(buffer, pubkey)
		if err != nil {
			panic(err)
		}

	case xdr.AssetTypeAssetTypeCreditAlphanum12:
		_, err := xdr.Marshal(buffer, asset.AlphaNum12.AssetCode)
		if err != nil {
			panic(err)
		}
		pubkey, ok := asset.AlphaNum12.Issuer.GetEd25519()
		if !ok {
			panic("issuer not a public key")
		}
		buffer.WriteString(":")
		_, err = xdr.Marshal(buffer, pubkey)
		if err != nil {
			panic(err)
		}

	default:
		panic("unexpected asset type")
	}

	slice := buffer.Bytes()
	scObject := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoBytes,
		Bin:  &slice,
	}

	return xdr.ScVal{
		Type: xdr.ScValTypeScvObject,
		Obj:  &scObject,
	}
}
