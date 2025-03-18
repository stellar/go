package token_transfer

import (
	"github.com/stellar/go/ingest"
	assetProto "github.com/stellar/go/ingest/asset"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewTransferEvent(meta *EventMeta, from, to string, amount string, token *assetProto.Asset) *TokenTransferEvent {
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

func NewMintEvent(meta *EventMeta, to string, amount string, token *assetProto.Asset) *TokenTransferEvent {
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

func NewBurnEvent(meta *EventMeta, from string, amount string, token *assetProto.Asset) *TokenTransferEvent {
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

func NewClawbackEvent(meta *EventMeta, from string, amount string, token *assetProto.Asset) *TokenTransferEvent {
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

func NewFeeEvent(meta *EventMeta, from string, amount string, token *assetProto.Asset) *TokenTransferEvent {
	return &TokenTransferEvent{
		Meta:  meta,
		Asset: token,
		Event: &TokenTransferEvent_Fee{
			Fee: &Fee{
				From:   from,
				Amount: amount,
			},
		},
	}
}

func NewEventMetaFromTx(tx ingest.LedgerTransaction, operationIndex *uint32, contractAddress string) *EventMeta {
	return &EventMeta{
		LedgerSequence:   tx.Ledger.LedgerSequence(),
		ClosedAt:         timestamppb.New(tx.Ledger.ClosedAt()),
		TxHash:           tx.Hash.HexString(),
		TransactionIndex: tx.Index - 1,
		OperationIndex:   operationIndex,
		ContractAddress:  contractAddress,
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
