package ingest

import (
	"fmt"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

type Event = xdr.ContractEvent
type EventType = int

// Note that there is no distinction between xfer() and xfer_from() in events,
// and this is true for the other *_from variants, as well.

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

type StellarAssetContractEvent struct {
	Type  int
	Asset xdr.Asset

	From   *xdr.ScAddress   // transfer, clawback, burn
	To     *xdr.ScAddress   // transfer, mint
	Amount *xdr.Int128Parts // transfer, mint, clawback, burn
	Admin  *xdr.ScAddress   // mint, clawback
}

func NewStellarAssetContractEvent(event *Event, networkPassphrase string) (*StellarAssetContractEvent, error) {
	evt := &StellarAssetContractEvent{}

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
	fmt.Println("here", len(assetBytes), assetBytes)
	if !ok || assetBytes == nil {
		return evt, ErrNotStellarAssetContract
	}

	asset, err := txnbuild.ParseAssetString(string(assetBytes))
	if err != nil {
		return evt, ErrNotStellarAssetContract
	}

	xdrAsset, err := xdr.NewCreditAsset(asset.GetCode(), asset.GetIssuer())
	if err != nil {
		return evt, ErrNotStellarAssetContract
	}

	expectedId, err := xdrAsset.ContractID(networkPassphrase)
	if err != nil {
		return evt, errors.Wrap(ErrNotStellarAssetContract, err.Error())
	}

	// This is the DEFINITIVE integrity check for whether or not this is a
	// SAC event. At this point, we can parse the event and treat it as
	// truth, mapping it to effects where appropriate.
	if expectedId != *event.ContractId { // nil check was earlier
		return evt, ErrNotStellarAssetContract
	}

	switch evt.Type {
	case EventTypeTransfer:
		return evt, evt.parseTransferEvent(topics, value)
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
// "transfer" event. It assumes that the `topics` EXCLUDES the function name,
// i.e. that its been validated previously. It will return a best-effort parsing
// even in error cases.
func (event *StellarAssetContractEvent) parseTransferEvent(topics xdr.ScVec, value xdr.ScVal) error {
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

	fmt.Println("xfer here")
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

	event.From = fromObj.Address
	event.To = toObj.Address

	valueObj, ok := value.GetObj()
	if !ok || valueObj == nil || valueObj.Type != xdr.ScObjectTypeScoI128 {
		return ErrNotTransferEvent
	}

	event.Amount = valueObj.I128
	return nil
}
