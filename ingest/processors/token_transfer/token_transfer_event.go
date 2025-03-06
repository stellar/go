package token_transfer

import (
	"github.com/stellar/go/ingest"
	addressProto "github.com/stellar/go/ingest/address"
	assetProto "github.com/stellar/go/ingest/asset"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewTransferEvent(meta *EventMeta, from, to *addressProto.Address, amount string, token *assetProto.Asset) *TokenTransferEvent {
	return &TokenTransferEvent{
		Meta:  meta,
		Asset: token,
		Event: &TokenTransferEvent_Transfer{
			Transfer: &Transfer{
				From:   from,
				To:     to,
				Amount: amount,
			},
		},
	}
}

func NewMintEvent(meta *EventMeta, to *addressProto.Address, amount string, token *assetProto.Asset) *TokenTransferEvent {
	return &TokenTransferEvent{
		Meta:  meta,
		Asset: token,
		Event: &TokenTransferEvent_Mint{
			Mint: &Mint{
				To:     to,
				Amount: amount,
			},
		},
	}
}

func NewBurnEvent(meta *EventMeta, from *addressProto.Address, amount string, token *assetProto.Asset) *TokenTransferEvent {
	return &TokenTransferEvent{
		Meta:  meta,
		Asset: token,
		Event: &TokenTransferEvent_Burn{
			Burn: &Burn{
				From:   from,
				Amount: amount,
			},
		},
	}
}

func NewClawbackEvent(meta *EventMeta, from *addressProto.Address, amount string, token *assetProto.Asset) *TokenTransferEvent {
	return &TokenTransferEvent{
		Meta:  meta,
		Asset: token,
		Event: &TokenTransferEvent_Clawback{
			Clawback: &Clawback{
				From:   from,
				Amount: amount,
			},
		},
	}
}

func NewFeeEvent(ledgerSequence uint32, closedAt time.Time, txHash string, from *addressProto.Address, amount string) *TokenTransferEvent {
	return &TokenTransferEvent{
		Meta: &EventMeta{
			LedgerSequence: ledgerSequence,
			ClosedAt:       timestamppb.New(closedAt),
			TxHash:         txHash,
		},
		Asset: assetProto.NewNativeAsset(),
		Event: &TokenTransferEvent_Fee{
			Fee: &Fee{
				From:   from,
				Amount: amount,
			},
		},
	}
}

func NewEventMeta(tx ingest.LedgerTransaction, operationIndex *uint32, contractAddress *addressProto.Address) *EventMeta {
	return &EventMeta{
		LedgerSequence:  tx.Ledger.LedgerSequence(),
		ClosedAt:        timestamppb.New(tx.Ledger.ClosedAt()),
		TxHash:          tx.Hash.HexString(),
		OperationIndex:  operationIndex,
		ContractAddress: contractAddress,
	}
}

func (event *TokenTransferEvent) GetEventType() string {
	switch event.GetEvent().(type) {
	case *TokenTransferEvent_Transfer:
		return "Transfer"
	case *TokenTransferEvent_Mint:
		return "Mint"
	case *TokenTransferEvent_Burn:
		return "Burn"
	case *TokenTransferEvent_Clawback:
		return "Clawback"
	case *TokenTransferEvent_Fee:
		return "Fee"
	default:
		return "Unknown"
	}
}
