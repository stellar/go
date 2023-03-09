package contractevents

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

type Event = xdr.ContractEvent
type EventType int

// Note that there is no distinction between xfer() and xfer_from() in events,
// nor the other *_from variants. This is intentional from the host environment.

const (
	// Implemented
	EventTypeTransfer EventType = iota
	EventTypeMint
	EventTypeClawback
	EventTypeBurn
	// TODO: Not implemented
	EventTypeIncrAllow
	EventTypeDecrAllow
	EventTypeSetAuth
	EventTypeSetAdmin
)

var (
	STELLAR_ASSET_CONTRACT_TOPICS = map[xdr.ScSymbol]EventType{
		xdr.ScSymbol("transfer"): EventTypeTransfer,
		xdr.ScSymbol("mint"):     EventTypeMint,
		xdr.ScSymbol("clawback"): EventTypeClawback,
		xdr.ScSymbol("burn"):     EventTypeBurn,
	}

	// TODO: Finer-grained parsing errors
	ErrNotStellarAssetContract = errors.New("event was not from a Stellar Asset Contract")
	ErrEventUnsupported        = errors.New("this type of Stellar Asset Contract event is unsupported")
)

type StellarAssetContractEvent interface {
	GetType() EventType
	GetAsset() xdr.Asset
}

type sacEvent struct {
	Type  EventType
	Asset xdr.Asset
}

func (e *sacEvent) GetAsset() xdr.Asset {
	return e.Asset
}

func (e *sacEvent) GetType() EventType {
	return e.Type
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

	if eventType, ok := STELLAR_ASSET_CONTRACT_TOPICS[*fn.Sym]; !ok {
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

	if !asset.IsNative() {
		evt.Asset, err = xdr.NewCreditAsset(asset.GetCode(), asset.GetIssuer())
		if err != nil {
			return evt, errors.Wrap(ErrNotStellarAssetContract, err.Error())
		}
	} else {
		evt.Asset = xdr.MustNewNativeAsset()
	}

	expectedId, err := evt.Asset.ContractID(networkPassphrase)
	if err != nil {
		return evt, errors.Wrap(ErrNotStellarAssetContract, err.Error())
	}

	// This is the DEFINITIVE integrity check for whether or not this is a
	// SAC event. At this point, we can parse the event and treat it as
	// truth, mapping it to effects where appropriate.
	if expectedId != *event.ContractId { // nil check was earlier
		return evt, ErrNotStellarAssetContract
	}

	switch evt.GetType() {
	case EventTypeTransfer:
		xferEvent := TransferEvent{sacEvent: *evt}
		return &xferEvent, xferEvent.parse(topics, value)

	case EventTypeMint:
		mintEvent := MintEvent{sacEvent: *evt}
		return &mintEvent, mintEvent.parse(topics, value)

	case EventTypeClawback:
		cbEvent := ClawbackEvent{sacEvent: *evt}
		return &cbEvent, cbEvent.parse(topics, value)

	case EventTypeBurn:
		burnEvent := BurnEvent{sacEvent: *evt}
		return &burnEvent, burnEvent.parse(topics, value)

	default:
		return evt, errors.Wrapf(ErrEventUnsupported,
			"event type %d ('%s') unsupported", evt.Type, fn.MustSym())
	}
}
