package ingest

import (
	"fmt"

	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

type Event = xdr.ContractEvent
type EventType = int

// Note that there is no distinction between xfer() and xfer_from() in events,
// nor the other *_from variants. This is intentional from the host environment.

const (
	// Implemented
	EventTypeTransfer = iota
	// TODO: Not implemented
	EventTypeIncrAllow = iota
	EventTypeDecrAllow = iota
	EventTypeSetAuth   = iota
	EventTypeSetAdmin  = iota
	EventTypeMint      = iota
	EventTypeClawback  = iota
	EventTypeBurn      = iota
)

var (
	STELLAR_ASSET_CONTRACT_TOPICS = map[xdr.ScSymbol]EventType{
		xdr.ScSymbol("mint"):     EventTypeMint,
		xdr.ScSymbol("transfer"): EventTypeTransfer,
		xdr.ScSymbol("clawback"): EventTypeClawback,
		xdr.ScSymbol("burn"):     EventTypeBurn,
	}

	// TODO: Better parsing errors
	ErrNotStellarAssetContract = errors.New("event was not from a Stellar Asset Contract")
	ErrEventUnsupported        = errors.New("this type of Stellar Asset Contract event is unsupported")
	ErrNotTransferEvent        = errors.New("event is an invalid 'transfer' event")
)

type StellarAssetContractEvent interface {
	GetType() int
	GetAsset() xdr.Asset
}

type sacEvent struct {
	Type  int
	Asset xdr.Asset
}

func (e *sacEvent) GetAsset() xdr.Asset {
	return e.Asset
}

func (e *sacEvent) GetType() int {
	return e.Type
}

type TransferEvent struct {
	sacEvent

	From   string
	To     string
	Amount *xdr.Int128Parts
}

func NewStellarAssetContractEvent(event *Event, networkPassphrase string) (StellarAssetContractEvent, error) {
	evt := &sacEvent{}

	if event.Type != xdr.ContractEventTypeContract || event.ContractId == nil || event.Body.V != 0 {
		return evt, ErrNotStellarAssetContract
	}

	// SAC event topics take the form <fn name>/<params...>/<token name>.
	//
	// For specific event forms, see here:
	// https://github.com/stellar/rs-soroban-env/blob/main/soroban-env-host/src/native_contract/token/event.rs#L44-L49
	topics := event.Body.MustV0().Topics
	value := event.Body.MustV0().Data

	// No relevant SAC events have <= 2 topics
	if len(topics) <= 2 {
		return evt, ErrNotStellarAssetContract
	}
	fn := topics[0]

	// Filter out events for function calls we don't care about
	if fn.Type != xdr.ScValTypeScvSymbol {
		return evt, ErrNotStellarAssetContract
	}
	if eventType, ok := STELLAR_ASSET_CONTRACT_TOPICS[fn.MustSym()]; !ok {
		return evt, ErrNotStellarAssetContract
	} else {
		evt.Type = eventType
	}

	// This looks like a SAC event, but does it act like a SAC event?
	//
	// To check that, ensure that the contract ID of the event matches the
	// contract ID that *would* represent the asset the event is claiming to
	// be. The asset is in canonical SEP-11 form:
	//  https://stellar.org/protocol/sep-11#alphanum4-alphanum12
	//
	// For all parsing errors, we just continue, since it's not a real
	// error, just an event non-complaint with SAC events.
	rawAsset := topics[len(topics)-1]
	assetContainer, ok := rawAsset.GetObj()
	if !ok || assetContainer == nil {
		return evt, ErrNotStellarAssetContract
	}

	assetBytes, ok := assetContainer.GetBin()
	if !ok || assetBytes == nil {
		return evt, ErrNotStellarAssetContract
	}

	asset, err := txnbuild.ParseAssetString(string(assetBytes))
	if err != nil {
		return evt, errors.Wrap(ErrNotStellarAssetContract, err.Error())
	}

	switch asset.IsNative() {
	case true:
		evt.Asset = xdr.MustNewNativeAsset()
	case false:
		evt.Asset, err = xdr.NewCreditAsset(asset.GetCode(), asset.GetIssuer())
	}
	if err != nil {
		return evt, errors.Wrap(ErrNotStellarAssetContract, err.Error())
	}

	expectedId, err := evt.Asset.ContractID(networkPassphrase)
	if err != nil {
		return evt, errors.Wrap(ErrNotStellarAssetContract, err.Error())
	}

	// This is the DEFINITIVE integrity check for whether or not this is a
	// SAC event. At this point, we can parse the event and treat it as
	// truth, mapping it to effects where appropriate.
	fmt.Println(asset.IsNative(), "here?")

	if expectedId != *event.ContractId { // nil check was earlier
		return evt, ErrNotStellarAssetContract
	}

	switch evt.GetType() {
	case EventTypeTransfer:
		xferEvent := TransferEvent{}
		return &xferEvent, xferEvent.parse(topics, value)

	case EventTypeMint:
	case EventTypeClawback:
	case EventTypeBurn:
	default:
		return evt, errors.Wrapf(ErrEventUnsupported,
			"event type %d ('%s') unsupported", evt.Type, fn.MustSym())
	}

	return evt, nil
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

	toObj, ok := from.GetObj()
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

	event.Amount = valueObj.I128
	return nil
}

// ScAddressToString converts the low-level `xdr.ScAddress` union into the
// appropriate strkey (contract C... or account ID G...).
//
// TODO: Should this return errors or just panic? Maybe just slap the "Must"
// prefix on the helper name?
func ScAddressToString(address *xdr.ScAddress) string {
	if address == nil {
		return ""
	}

	var result string
	var err error

	switch address.Type {
	case xdr.ScAddressTypeScAddressTypeAccount:
		pubkey := address.MustAccountId().Ed25519
		fmt.Println("pubkey:", address.MustAccountId())

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
