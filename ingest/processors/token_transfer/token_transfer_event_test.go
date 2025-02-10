package token_transfer

import (
	"github.com/stellar/go/ingest/address"
	"github.com/stellar/go/ingest/asset"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"testing"
	"time"
)

func newTestAddress() *address.Address {
	return &address.Address{
		AddressType: address.AddressType_ADDRESS_TYPE_ACCOUNT,
		StrKey:      "GBRX5D3FLJ72FHYFVFF2BOICRVDF7FESIAT6GQ4K3ST2MXXJJZC24H2F",
	}
}

func newTestAsset() *asset.Asset {
	return &asset.Asset{
		AssetType: &asset.Asset_IssuedAsset{
			IssuedAsset: &asset.IssuedAsset{
				AssetCode: "USDC",
				Issuer:    "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
			},
		},
	}
}

func newTestEventMeta() *EventMeta {
	return &EventMeta{
		LedgerSequence: 12345,
		ClosedAt:       timestamppb.New(time.Now()),
		TxHash:         "abc123xyz",
	}
}

func TestNewTransferEvent_Serialization(t *testing.T) {
	from := newTestAddress()
	to := newTestAddress()
	amount := "1000"
	meta := newTestEventMeta()
	token := newTestAsset()
	event := NewTransferEvent(meta, from, to, amount, token)

	data, err := proto.Marshal(event)
	assert.NoError(t, err)

	var deserializedEvent TokenTransferEvent
	err = proto.Unmarshal(data, &deserializedEvent)
	assert.NoError(t, err)

	assert.True(t, proto.Equal(event.Meta, deserializedEvent.Meta))
	assert.True(t, proto.Equal(event.Asset, deserializedEvent.Asset))
	assert.True(t, proto.Equal(event.GetTransfer(), deserializedEvent.GetTransfer()))
}

func TestNewMintEvent_Serialization(t *testing.T) {
	to := newTestAddress()
	amount := "500"
	meta := newTestEventMeta()
	token := newTestAsset()
	event := NewMintEvent(meta, to, amount, token)

	data, err := proto.Marshal(event)
	assert.NoError(t, err)

	var deserializedEvent TokenTransferEvent
	err = proto.Unmarshal(data, &deserializedEvent)
	assert.NoError(t, err)

	assert.True(t, proto.Equal(event.Meta, deserializedEvent.Meta))
	assert.True(t, proto.Equal(event.Asset, deserializedEvent.Asset))
	assert.True(t, proto.Equal(event.GetMint(), deserializedEvent.GetMint()))
}

func TestNewBurnEvent_Serialization(t *testing.T) {
	from := newTestAddress()
	amount := "200"
	meta := newTestEventMeta()
	token := newTestAsset()
	event := NewBurnEvent(meta, from, amount, token)

	// Serialize the event
	data, err := proto.Marshal(event)
	assert.NoError(t, err)

	var deserializedEvent TokenTransferEvent
	err = proto.Unmarshal(data, &deserializedEvent)
	assert.NoError(t, err)

	assert.True(t, proto.Equal(event.Meta, deserializedEvent.Meta))
	assert.True(t, proto.Equal(event.Asset, deserializedEvent.Asset))
	assert.True(t, proto.Equal(event.GetBurn(), deserializedEvent.GetBurn()))
}

func TestNewClawbackEvent_Serialization(t *testing.T) {
	from := newTestAddress()
	amount := "300"
	meta := newTestEventMeta()
	token := newTestAsset()
	event := NewClawbackEvent(meta, from, amount, token)

	data, err := proto.Marshal(event)
	assert.NoError(t, err)

	var deserializedEvent TokenTransferEvent
	err = proto.Unmarshal(data, &deserializedEvent)
	assert.NoError(t, err)

	assert.True(t, proto.Equal(event.Meta, deserializedEvent.Meta))
	assert.True(t, proto.Equal(event.Asset, deserializedEvent.Asset))
	assert.True(t, proto.Equal(event.GetClawback(), deserializedEvent.GetClawback()))
}

func TestNewFeeEvent_Serialization(t *testing.T) {
	from := newTestAddress()
	amount := "50"
	token := newTestAsset()
	event := NewFeeEvent(12345, time.Now(), "abc123xyz", from, amount, token)

	data, err := proto.Marshal(event)
	assert.NoError(t, err)

	var deserializedEvent TokenTransferEvent
	err = proto.Unmarshal(data, &deserializedEvent)
	assert.NoError(t, err)

	assert.True(t, proto.Equal(event.Meta, deserializedEvent.Meta))
	assert.True(t, proto.Equal(event.Asset, deserializedEvent.Asset))
	assert.True(t, proto.Equal(event.GetFee(), deserializedEvent.GetFee()))
}
