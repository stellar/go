package token_transfer

import (
	"fmt"
	"github.com/stellar/go/ingest"
	assetProto "github.com/stellar/go/ingest/asset"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	TransferEvent = "Transfer"
	MintEvent     = "Mint"
	BurnEvent     = "Burn"
	ClawbackEvent = "Clawback"
	FeeEvent      = "Fee"
)

func NewTransferEvent(meta *EventMeta, from, to string, amount string, token *assetProto.Asset) *TokenTransferEvent {
	return &TokenTransferEvent{
		Meta: meta,
		Event: &TokenTransferEvent_Transfer{
			Transfer: &Transfer{
				From:  from,
				To:    to,
				Asset: token,

				Amount: amount,
			},
		},
	}
}

func NewMintEvent(meta *EventMeta, to string, amount string, token *assetProto.Asset) *TokenTransferEvent {
	return &TokenTransferEvent{
		Meta: meta,
		Event: &TokenTransferEvent_Mint{
			Mint: &Mint{
				To:    to,
				Asset: token,

				Amount: amount,
			},
		},
	}
}

func NewBurnEvent(meta *EventMeta, from string, amount string, token *assetProto.Asset) *TokenTransferEvent {
	return &TokenTransferEvent{
		Meta: meta,
		Event: &TokenTransferEvent_Burn{
			Burn: &Burn{
				From:   from,
				Asset:  token,
				Amount: amount,
			},
		},
	}
}

func NewClawbackEvent(meta *EventMeta, from string, amount string, token *assetProto.Asset) *TokenTransferEvent {
	return &TokenTransferEvent{
		Meta: meta,
		Event: &TokenTransferEvent_Clawback{
			Clawback: &Clawback{
				From:   from,
				Asset:  token,
				Amount: amount,
			},
		},
	}
}

func NewFeeEvent(meta *EventMeta, from string, amount string, token *assetProto.Asset) *TokenTransferEvent {
	return &TokenTransferEvent{
		Meta: meta,
		Event: &TokenTransferEvent_Fee{
			Fee: &Fee{
				From:   from,
				Asset:  token,
				Amount: amount,
			},
		},
	}
}

func NewEventMetaFromTx(tx ingest.LedgerTransaction, operationIndex *uint32, contractAddress string) *EventMeta {
	// The input operationIndex is 0-indexed.
	// As per SEP-35, the OperationIndex in the output proto should be 1-indexed.
	// Make that conversion here
	var outputOpIndex *uint32
	if operationIndex != nil {
		temp := *operationIndex + 1
		outputOpIndex = &temp
	}
	return &EventMeta{
		LedgerSequence:   tx.Ledger.LedgerSequence(),
		ClosedAt:         timestamppb.New(tx.Ledger.ClosedAt()),
		TxHash:           tx.Hash.HexString(),
		TransactionIndex: tx.Index, // The index in ingest.LedgerTransaction is already 1-indexed
		OperationIndex:   outputOpIndex,
		ContractAddress:  contractAddress,
	}
}

func (event *TokenTransferEvent) GetEventType() string {
	switch event.GetEvent().(type) {
	case *TokenTransferEvent_Transfer:
		return TransferEvent
	case *TokenTransferEvent_Mint:
		return MintEvent
	case *TokenTransferEvent_Burn:
		return BurnEvent
	case *TokenTransferEvent_Clawback:
		return ClawbackEvent
	case *TokenTransferEvent_Fee:
		return FeeEvent
	default:
		return "Unknown"
	}
}

func (event *TokenTransferEvent) GetAsset() *assetProto.Asset {
	var asset *assetProto.Asset
	switch event.GetEvent().(type) {
	case *TokenTransferEvent_Mint:
		asset = event.GetMint().GetAsset()
	case *TokenTransferEvent_Burn:
		asset = event.GetBurn().GetAsset()
	case *TokenTransferEvent_Clawback:
		asset = event.GetClawback().GetAsset()
	case *TokenTransferEvent_Fee:
		asset = event.GetFee().GetAsset()
	case *TokenTransferEvent_Transfer:
		asset = event.GetTransfer().GetAsset()
	default:
		panic(fmt.Errorf("unkown event type:%v", event))
	}
	return asset
}
