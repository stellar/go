package offers

import (
	"github.com/guregu/null"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/utils"
	"github.com/stellar/go/xdr"
)

// Constants for event types
const (
	EventTypeOfferCreated = "OfferCreated"
	EventTypeOfferFill    = "OfferUpdated"
	EventTypeOfferClosed  = "OfferClosed"
)

// Base struct with common fields for all offer events.
type OfferEventData struct {
	SellerId string
	OfferID  int64

	BuyingAsset  xdr.Asset
	SellingAsset xdr.Asset

	RemainingAmount int64 // Remaining amount that still needs to be filled for this offer
	PriceN          int32
	PriceD          int32

	IsPassive          bool
	LastModifiedLedger uint32
	Sponsor            null.String
}

type OfferEvent interface {
	OfferEventType() string
	GetOfferData() OfferEventData
}

type OfferCreatedEvent struct {
	OfferEventData
}

// Method to get common event data
func (e OfferEventData) GetOfferData() OfferEventData {
	return e
}

func (e OfferCreatedEvent) OfferEventType() string { return EventTypeOfferCreated }

type OfferFillEvent struct {
	OfferEventData
	FillAmount     int64
	MatchingOrders []OfferFill
}

func (e OfferFillEvent) OfferEventType() string { return EventTypeOfferFill }

type OfferClosedEvent struct {
	OfferEventData
	CloseReason string
}

func (e OfferClosedEvent) OfferEventType() string { return EventTypeOfferClosed }

type OfferFillInfo struct {
	AssetSold    xdr.Asset
	AmountSold   int64
	AssetBought  xdr.Asset
	AmountBought int64
}

type OfferFill interface {
	OfferClaimantType() string
}

type LiquidityPoolClaimant struct {
	PoolId xdr.PoolId
	*OfferFillInfo
}

type OrderBookClaimant struct {
	SellerId xdr.AccountId
	OfferId  int64
	*OfferFillInfo
}

const (
	OrderBookClaimantType     = "OrderBookClaimant"
	LiquidityPoolClaimantType = "LiquidityPoolClaimant"
)

func (o *LiquidityPoolClaimant) OfferClaimantType() string {
	return LiquidityPoolClaimantType
}

func (o *OrderBookClaimant) OfferClaimantType() string {
	return OrderBookClaimantType
}

func populateOfferData(e *xdr.LedgerEntry) OfferEventData {
	offer := e.Data.MustOffer()
	return OfferEventData{
		SellerId: offer.SellerId.Address(),
		OfferID:  int64(offer.OfferId),

		BuyingAsset:        offer.Buying,
		SellingAsset:       offer.Selling,
		RemainingAmount:    int64(offer.Amount),
		PriceN:             int32(offer.Price.N),
		PriceD:             int32(offer.Price.D),
		IsPassive:          (offer.Flags & 0x1) != 0,
		LastModifiedLedger: uint32(e.LastModifiedLedgerSeq),
		Sponsor:            utils.LedgerEntrySponsorToNullString(*e),
	}
}

func ProcessOffer(change ingest.Change) OfferEvent {
	if change.Type != xdr.LedgerEntryTypeOffer {
		return nil
	}
	var o OfferEventData
	var event OfferEvent

	switch {
	case change.Pre == nil && change.Post != nil:
		// New offer
		o = populateOfferData(change.Post)
		event = OfferCreatedEvent{OfferEventData: o}

	case change.Pre != nil && change.Post != nil:
		// Order Fill
		o = populateOfferData(change.Post)
		fillAmt := int64(change.Pre.Data.MustOffer().Amount - change.Post.Data.MustOffer().Amount)
		event = OfferFillEvent{OfferEventData: o, FillAmount: fillAmt}
		//TODO: populate MatchingOrders field in OfferFillEvent

		// Offer Fill
	case change.Pre != nil && change.Post == nil:
		// Offer Removed
		o = populateOfferData(change.Pre)
		event = OfferClosedEvent{OfferEventData: o}
		//TODO: populate CloseReason field in OfferClosedEvent
	}
	return event

}
