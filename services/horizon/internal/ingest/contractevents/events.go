package contractevents

import (
	"errors"
	"fmt"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/xdr"
)

// EventType represents the type of Stellar Asset Contract event
type EventType int

const (
	EventTypeTransfer EventType = iota
	EventTypeMint
	EventTypeClawback
	EventTypeBurn
)

var (
	eventTypeMap = map[xdr.ScSymbol]EventType{
		xdr.ScSymbol("transfer"): EventTypeTransfer,
		xdr.ScSymbol("mint"):     EventTypeMint,
		xdr.ScSymbol("clawback"): EventTypeClawback,
		xdr.ScSymbol("burn"):     EventTypeBurn,
	}

	ErrUnsupportedTxMetaVersion = errors.New("tx meta version not supported")
	ErrNotStellarAssetContract  = errors.New("event was not from a Stellar Asset Contract")
	ErrEventUnsupported         = errors.New("this type of Stellar Asset Contract event is unsupported")
	ErrEventIntegrity           = errors.New("contract ID doesn't match asset + passphrase")
)

// StellarAssetContractEvent represents a parsed SAC event
type StellarAssetContractEvent struct {
	Type            EventType
	Asset           xdr.Asset
	From            string // For transfer, burn, clawback
	To              string // For transfer, mint
	Amount          xdr.Int128Parts
	destinationMemo xdr.Memo // Can be uint64, []byte, or string (V4 only)
}

// parseAddress extracts and converts an address from an ScVal
func parseAddress(topic xdr.ScVal) (string, error) {
	addr, ok := topic.GetAddress()
	if !ok {
		return "", errors.New("topic is not an address")
	}
	return addr.String()
}

// NewStellarAssetContractEvent parses a contract event into a SAC event
func NewStellarAssetContractEvent(tx ingest.LedgerTransaction, event *xdr.ContractEvent, networkPassphrase string) (*StellarAssetContractEvent, error) {
	switch tx.UnsafeMeta.V {
	case 3:
		return parseSacEventFromTxMetaV3(event, networkPassphrase)
	case 4:
		return parseSacEventFromTxMetaV4(event, networkPassphrase)
	default:
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedTxMetaVersion, tx.UnsafeMeta.V)
	}
}

func parseTransferEventFromTxMetaV4(topics xdr.ScVec, event *StellarAssetContractEvent) error {
	// Format: ["transfer", from addr, to addr, sep11 asset]
	if len(topics) != 4 {
		return errors.New("transfer event requires 4 topics")
	}

	from, err := parseAddress(topics[1])
	if err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	to, err := parseAddress(topics[2])
	if err != nil {
		return fmt.Errorf("invalid to address: %w", err)
	}
	event.From = from
	event.To = to
	return nil
}

func parseMintEventFromTxMetaV4(topics xdr.ScVec, event *StellarAssetContractEvent) error {
	// Format: ["mint", to addr, sep11 asset] - NO admin address in V4
	if len(topics) != 3 {
		return errors.New("mint event requires 3 topics")
	}

	to, err := parseAddress(topics[1])
	if err != nil {
		return fmt.Errorf("invalid to address: %w", err)
	}
	event.To = to
	return nil
}

func parseClawbackEventFromTxMetaV4(topics xdr.ScVec, event *StellarAssetContractEvent) error {
	// Format: ["clawback", from addr, sep11 asset] - NO admin address in V4
	if len(topics) != 3 {
		return errors.New("clawback event requires 3 topics")
	}

	from, err := parseAddress(topics[1])
	if err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	event.From = from
	return nil
}

func parseBurnEventFromTxMetaV4(topics xdr.ScVec, event *StellarAssetContractEvent) error {
	// Format: ["burn", from addr, sep11 asset] - same as V3
	if len(topics) != 3 {
		return errors.New("burn event requires 3 topics")
	}

	from, err := parseAddress(topics[1])
	if err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	event.From = from
	return nil
}

func parseSacEventFromTxMetaV4(event *xdr.ContractEvent, networkPassphrase string) (*StellarAssetContractEvent, error) {
	// Basic validation
	if event.Type != xdr.ContractEventTypeContract || event.ContractId == nil || event.Body.V != 0 {
		return nil, ErrNotStellarAssetContract
	}

	topics := event.Body.V0.Topics

	// Need at least 3 topics for any SAC event in V4
	if len(topics) < 3 {
		return nil, ErrNotStellarAssetContract
	}

	// Parse function name
	fn, ok := topics[0].GetSym()
	if !ok {
		return nil, ErrNotStellarAssetContract
	}

	eventType, found := eventTypeMap[fn]
	if !found {
		return nil, ErrNotStellarAssetContract
	}

	// Parse asset from last topic
	assetStr, ok := topics[len(topics)-1].GetStr()
	if !ok || assetStr == "" {
		return nil, ErrNotStellarAssetContract
	}

	// Try parsing the asset from its SEP-11 representation
	assets, err := xdr.BuildAssets(string(assetStr))
	if err != nil {
		return nil, errors.Join(ErrNotStellarAssetContract, err)
	} else if len(assets) > 1 {
		return nil, errors.Join(ErrNotStellarAssetContract, errors.New("more than one asset found in SEP-11 asset string"))
	}

	asset := assets[0]
	// Verify contract ID matches asset
	expectedId, err := asset.ContractID(networkPassphrase)
	if err != nil {
		return nil, errors.Join(ErrNotStellarAssetContract, err)
	}

	if expectedId != *event.ContractId {
		return nil, ErrEventIntegrity
	}

	// Parse amount and optional to_muxed_id from data
	value := event.Body.V0.Data
	var amount xdr.Int128Parts
	var memo xdr.Memo

	// Try to parse as ScMap first (V4 format with to_muxed_id)
	if mapData, ok := value.GetMap(); ok {
		if mapData == nil {
			return nil, errors.New("map is empty")
		}
		amount, memo, err = parseV4MapData(*mapData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse V4 map data: %w", err)
		}
	} else {
		// Fall back to direct i128 parsing (V4 without to_muxed_id)
		amount, ok = value.GetI128()
		if !ok {
			return nil, errors.New("invalid amount in event value")
		}
	}

	// Parse addresses based on event type
	sacEvent := &StellarAssetContractEvent{
		Type:            eventType,
		Asset:           asset,
		Amount:          amount,
		destinationMemo: memo,
	}

	switch eventType {
	case EventTypeTransfer:
		if err := parseTransferEventFromTxMetaV4(topics, sacEvent); err != nil {
			return nil, err
		}
	case EventTypeMint:
		if err := parseMintEventFromTxMetaV4(topics, sacEvent); err != nil {
			return nil, err
		}
	case EventTypeClawback:
		if err := parseClawbackEventFromTxMetaV4(topics, sacEvent); err != nil {
			return nil, err
		}
	case EventTypeBurn:
		if err := parseBurnEventFromTxMetaV4(topics, sacEvent); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%w: %v", ErrEventUnsupported, eventType)
	}

	return sacEvent, nil
}

// parseV4MapData parses the ScMap data format used in V4 events
func parseV4MapData(mapData xdr.ScMap) (xdr.Int128Parts, xdr.Memo, error) {
	var foundAmount, foundMuxedId bool
	var amount xdr.Int128Parts
	var memo xdr.Memo

	if len(mapData) != 2 {
		return amount, memo, fmt.Errorf("expected exactly 2 elements in map data, but found %d", len(mapData))
	}

	for _, entry := range mapData {
		key, ok := entry.Key.GetSym()
		if !ok {
			return amount, memo, fmt.Errorf("invalid key type in data map: %s", entry.Key.Type)
		}

		switch string(key) {
		case "amount":
			amount, ok = entry.Val.GetI128()
			if !ok {
				return amount, memo, errors.New("amount field is not i128")
			}
			foundAmount = true

		case "to_muxed_id":
			// Handle different types for to_muxed_id.. do i need a default to error?
			foundMuxedId = true
			switch entry.Val.Type {
			case xdr.ScValTypeScvU64:
				if val, ok := entry.Val.GetU64(); ok {
					memo = xdr.MemoID(uint64(val))
				}
			case xdr.ScValTypeScvBytes:
				if val, ok := entry.Val.GetBytes(); ok {
					memo = xdr.MemoHash(xdr.Hash(val[:]))
				}
			case xdr.ScValTypeScvString:
				if val, ok := entry.Val.GetStr(); ok {
					memo = xdr.MemoText(string(val))
				}
			default:
				return amount, memo, fmt.Errorf("invalid to_muxed_id type for data: %s", entry.Val.Type)
			}
		}
	}

	if !foundAmount {
		return amount, memo, errors.New("amount field not found in map")
	} else if !foundMuxedId {
		return amount, memo, errors.New("to_muxed_id field not found in map")
	}

	return amount, memo, nil
}

func parseTransferEventFromTxMetaV3(topics xdr.ScVec, event *StellarAssetContractEvent) error {
	// Format: ["transfer", from addr, to addr, sep11 asset], i128 amount
	if len(topics) != 4 {
		return errors.New("transfer event requires 4 topics")
	}

	from, err := parseAddress(topics[1])
	if err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	to, err := parseAddress(topics[2])
	if err != nil {
		return fmt.Errorf("invalid to address: %w", err)
	}
	event.From = from
	event.To = to
	return nil
}

func parseMintEventFromTxMetaV3(topics xdr.ScVec, event *StellarAssetContractEvent) error {
	// Format: ["mint", admin addr, to addr, sep11 asset], i128 amount
	if len(topics) != 4 {
		return errors.New("mint event requires 4 topics")
	}

	// Admin is not used. but needs to be parsed for SAC format correctness
	_, err := parseAddress(topics[1])
	if err != nil {
		return fmt.Errorf("invalid admin address: %w", err)
	}
	to, err := parseAddress(topics[2])
	if err != nil {
		return fmt.Errorf("invalid to address: %w", err)
	}
	event.To = to
	return nil
}

func parseClawbackEventFromTxMetaV3(topics xdr.ScVec, event *StellarAssetContractEvent) error {
	// Format: ["clawback", admin addr, from addr, sep11 asset], i128 amount
	if len(topics) != 4 {
		return errors.New("clawback event requires 4 topics")
	}

	// Admin is not used. but needs to be parsed for SAC format correctness
	_, err := parseAddress(topics[1])
	if err != nil {
		return fmt.Errorf("invalid admin address: %w", err)
	}
	from, err := parseAddress(topics[2])
	if err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	event.From = from
	return nil
}

func parseBurnEventFromTxMetaV3(topics xdr.ScVec, event *StellarAssetContractEvent) error {
	// Format: ["burn", from addr, sep11 asset], i128 amount
	if len(topics) != 3 {
		return errors.New("burn event requires 3 topics")
	}

	from, err := parseAddress(topics[1])
	if err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	event.From = from
	return nil
}

func parseSacEventFromTxMetaV3(event *xdr.ContractEvent, networkPassphrase string) (*StellarAssetContractEvent, error) {
	// Basic validation
	if event.Type != xdr.ContractEventTypeContract || event.ContractId == nil || event.Body.V != 0 {
		return nil, ErrNotStellarAssetContract
	}

	topics := event.Body.V0.Topics

	// Need at least 3 topics for any SAC event
	if len(topics) < 3 {
		return nil, ErrNotStellarAssetContract
	}

	// Parse function name
	fn, ok := topics[0].GetSym()
	if !ok {
		return nil, ErrNotStellarAssetContract
	}

	eventType, found := eventTypeMap[fn]
	if !found {
		return nil, ErrNotStellarAssetContract
	}

	// Parse asset from last topic
	assetStr, ok := topics[len(topics)-1].GetStr()
	if !ok || assetStr == "" {
		return nil, ErrNotStellarAssetContract
	}

	// Try parsing the asset from its SEP-11 representation
	assets, err := xdr.BuildAssets(string(assetStr))
	if err != nil {
		return nil, errors.Join(ErrNotStellarAssetContract, err)
	} else if len(assets) > 1 {
		return nil, errors.Join(ErrNotStellarAssetContract, errors.New("more than one asset found in SEP-11 asset string"))
	}

	asset := assets[0]
	// Verify contract ID matches asset
	expectedId, err := asset.ContractID(networkPassphrase)
	if err != nil {
		return nil, errors.Join(ErrNotStellarAssetContract, err)
	}

	if expectedId != *event.ContractId {
		return nil, ErrEventIntegrity
	}

	// Parse amount
	value := event.Body.V0.Data
	amount, ok := value.GetI128()
	if !ok {
		return nil, errors.New("invalid amount in event value")
	}

	// Parse addresses based on event type
	sacEvent := &StellarAssetContractEvent{
		Type:   eventType,
		Asset:  asset,
		Amount: amount,
	}

	switch eventType {
	case EventTypeTransfer:
		if err := parseTransferEventFromTxMetaV3(topics, sacEvent); err != nil {
			return nil, err
		}
	case EventTypeMint:
		if err := parseMintEventFromTxMetaV3(topics, sacEvent); err != nil {
			return nil, err
		}
	case EventTypeClawback:
		if err := parseClawbackEventFromTxMetaV3(topics, sacEvent); err != nil {
			return nil, err
		}
	case EventTypeBurn:
		if err := parseBurnEventFromTxMetaV3(topics, sacEvent); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%w: %v", ErrEventUnsupported, eventType)
	}

	return sacEvent, nil
}
