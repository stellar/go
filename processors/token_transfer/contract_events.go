package token_transfer

import (
	"fmt"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

type ErrNotSep41TokenEvent struct {
	Message string
}

func (e ErrNotSep41TokenEvent) Error() string {
	return e.Message
}

func errNotSep41TokenFromMsg(msg string) ErrNotSep41TokenEvent {
	return ErrNotSep41TokenEvent{msg}
}

func errNotSep41TokenFromError(err error) ErrNotSep41TokenEvent {
	return ErrNotSep41TokenEvent{err.Error()}
}

// parseEvent is the main entry point for parsing contract events
// It attempts to parse events with a flexible, hierarchical approach
func (p *EventsProcessor) parseEvent(tx ingest.LedgerTransaction, opIndex *uint32, contractEvent xdr.ContractEvent) (*TokenTransferEvent, error) {
	// Validate basic contract contractEvent structure
	if contractEvent.Type != xdr.ContractEventTypeContract ||
		contractEvent.ContractId == nil ||
		contractEvent.Body.V != 0 {
		return nil, errNotSep41TokenFromMsg("invalid contractEvent")
	}

	topics := contractEvent.Body.V0.Topics

	// Require at least 2 topics for meaningful contractEvent parsing
	if len(topics) < 2 {
		return nil, errNotSep41TokenFromMsg("insufficient topics in contract event")
	}

	// Extract the contractEvent function name
	fn, ok := topics[0].GetSym()
	if !ok {
		return nil, errNotSep41TokenFromMsg("invalid function name")
	}

	// First, try parsing as a standard SEP41 token contractEvent
	var protoEvent *TokenTransferEvent
	protoEvent, sepErr := parseCustomTokenEvent(string(fn), tx, opIndex, contractEvent)
	if sepErr != nil {
		return nil, sepErr
	}

	// This has passed validation for SEP-41 complaint token.
	// At the very least, you will now emit a contractEvent.
	// Attempt SAC validation if possible, to get asset name

	// SAC validation requires a very strict check on len(topics)
	// For transfer, mint and clawback - there will be exactly 4 elements
	// For burn, there will be exactly 3 events
	// transfer - "transfer", toAddr, fromAddr, sep11AssetString
	// mint - "mint", admin, toAddr, sep11AssetString
	// clawback - "clawback", admin, fromAddr, sep11AssetString
	// burn - "burn", fromAddr, sep11AssetString
	if len(topics) == 3 || len(topics) == 4 {
		lastTopic := topics[len(topics)-1]
		if assetStr, ok := lastTopic.GetStr(); ok && assetStr != "" {
			// Try parsing the asset from its SEP-11 representation
			assets, err := xdr.BuildAssets(string(assetStr))
			if err == nil && len(assets) == 1 {
				asset := assets[0]
				// Verify contract ID matches expected asset contract ID
				expectedId, idErr := asset.ContractID(p.networkPassphrase)
				if idErr == nil && expectedId == *contractEvent.ContractId {
					// If contract ID matches, update with validated asset
					protoEvent.SetAsset(asset)

					// This is tricky. Burn and mint events currently show up as transfer in SAC events
					// This will be fixed once CAP-67 unified events is released:
					// https://github.com/stellar/stellar-protocol/blob/master/core/cap-0067.md#protocol-upgrade-transition
					// Meanwhile, we fix it here manually by checking if src/dst is issuer of asset, and if it is, we issue mint/burn instead
					maybeTransferEvent := protoEvent.GetTransfer()
					if maybeTransferEvent != nil {
						protoEvent, err = p.mintOrBurnOrTransferEvent(
							tx, opIndex,
							maybeTransferEvent.Asset.ToXdrAsset(),
							maybeTransferEvent.From,
							maybeTransferEvent.To,
							maybeTransferEvent.Amount,
							true,
						)
						if err != nil {
							return nil, fmt.Errorf("contract transfer event error: %w", err)
						}

					}
				}
			}
		}
	}

	return protoEvent, nil
}

// parseCustomTokenEvent attempts to parse a generic SEP41 token event
func parseCustomTokenEvent(
	eventType string, tx ingest.LedgerTransaction, opIndex *uint32, contractEvent xdr.ContractEvent,
) (*TokenTransferEvent, error) {

	topics := contractEvent.Body.V0.Topics
	value := contractEvent.Body.V0.Data

	// Parse token amount. If that fails, then no need to bother checking for eventType
	amt, ok := value.GetI128()
	if !ok {
		return nil, errNotSep41TokenFromMsg("invalid event amount")
	}
	amtRaw128 := amount.String128Raw(amt)

	contractAddress := strkey.MustEncode(strkey.VersionByteContract, contractEvent.ContractId[:])
	meta := NewEventMetaFromTx(tx, opIndex, contractAddress)
	var event *TokenTransferEvent

	// Determine event type based on function name
	lenTopics := len(topics)
	switch eventType {
	case TransferEvent:
		// Transfer requires MINIMUM 3 topics: event type, fromAddr, toAddr
		if lenTopics < 3 {
			return nil, errNotSep41TokenFromMsg(fmt.Sprintf("transfer event requires minimum 3 topics, found: %v", lenTopics))
		}
		from, err := extractAddress(topics[1])
		if err != nil {
			return nil, errNotSep41TokenFromError(fmt.Errorf("invalid fromAddress. error: %w", err))
		}
		to, err := extractAddress(topics[2])
		if err != nil {
			return nil, errNotSep41TokenFromError(fmt.Errorf("invalid toAddress. error: %w", err))
		}
		event = NewTransferEvent(meta, from, to, amtRaw128, nil)

	case MintEvent:
		// Mint requires MINIMUM 3 topics - event type, admin, toAddr
		if lenTopics < 3 {
			return nil, errNotSep41TokenFromMsg(fmt.Sprintf("mint event requires minimum 3 topics, found: %v", lenTopics))
		}
		// Dont care for admin when generating proto, but validating nonetheless
		_, err := extractAddress(topics[1])
		if err != nil {
			return nil, errNotSep41TokenFromError(fmt.Errorf("invalid adminAddress. error: %w", err))
		}
		to, err := extractAddress(topics[2])
		if err != nil {
			return nil, errNotSep41TokenFromError(fmt.Errorf("invalid toAddress error: %w", err))
		}
		event = NewMintEvent(meta, to, amtRaw128, nil)

	case ClawbackEvent:
		// Clawback requires MINIMUM 3 topics - event type, admin, fromAddr
		if lenTopics < 3 {
			return nil, errNotSep41TokenFromMsg(fmt.Sprintf("clawback event requires minimum 3 topics, found: %v", lenTopics))
		}
		// Dont care for admin when generating proto, but validating nonetheless
		_, err := extractAddress(topics[1])
		if err != nil {
			return nil, errNotSep41TokenFromError(fmt.Errorf("invalid adminAddress. error: %w", err))
		}
		from, err := extractAddress(topics[2])
		if err != nil {
			return nil, errNotSep41TokenFromError(fmt.Errorf("invalid fromAddress error: %w", err))
		}
		event = NewClawbackEvent(meta, from, amtRaw128, nil)

	case BurnEvent:
		// Burn requires MINIMUM 2 topics - event type, fromAddr
		if lenTopics < 2 {
			return nil, errNotSep41TokenFromMsg(fmt.Sprintf("burn event requires minimum 2 topics, found: %v", lenTopics))
		}
		from, err := extractAddress(topics[1])
		if err != nil {
			return nil, errNotSep41TokenFromError(fmt.Errorf("invalid fromAddress error: %w", err))
		}
		event = NewBurnEvent(meta, from, amtRaw128, nil)

	default:
		return nil, errNotSep41TokenFromMsg(fmt.Sprintf("unsupported custom token event type: %v", eventType))
	}

	return event, nil
}
