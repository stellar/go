package trades

import "github.com/stellar/go/xdr"

type TradeEventType int

const (
	TradeEventTypeUnknown              TradeEventType = iota // Default value
	TradeEventTypeOfferCreated                               // Offer created event
	TradeEventTypeOfferUpdated                               // Offer updated event
	TradeEventTypeOfferClosed                                // Offer closed event
	TradeEventTypeLiquidityPoolUpdated                       // Liquidity pool update event
)

type TradeEvent interface {
	GetTradeEventType() TradeEventType // Method to retrieve the type of the trade event
}

type OfferBase struct {
	Selling xdr.Asset // Asset being sold
	Buying  xdr.Asset // Asset being bought
	Amount  xdr.Int64 // Total amount of the selling asset
	Price   xdr.Price // Price of the offer
	Flags   uint32    // Flags for the offer (e.g., passive, sell offers)
}

type OfferCreatedEvent struct {
	SellerId         xdr.AccountId // Account ID of the seller
	OfferId          xdr.Int64     // ID of the created offer
	OfferState       OfferBase     // Initial state of the offer
	CreatedLedgerSeq uint32        // Ledger sequence where the offer was created
	Fills            []FillInfo    // List of fills that occurred during the creation
}

func (e OfferCreatedEvent) GetTradeEventType() TradeEventType {
	return TradeEventTypeOfferCreated
}

type OfferUpdatedEvent struct {
	SellerId             xdr.AccountId // Account ID of the seller
	OfferId              xdr.Int64     // ID of the updated offer
	PrevUpdatedLedgerSeq uint32        // Ledger sequence of the previous update
	PreviousOfferState   OfferBase     // Previous state of the offer
	UpdatedOfferState    OfferBase     // Updated state of the offer
	UpdatedLedgerSeq     uint32        // Ledger sequence where the offer was updated
	Fills                []FillInfo    // List of fills that occurred during the update
}

func (e OfferUpdatedEvent) GetTradeEventType() TradeEventType {
	return TradeEventTypeOfferUpdated
}

type OfferClosedEvent struct {
	SellerId        xdr.AccountId // Account ID of the seller
	OfferId         xdr.Int64     // ID of the closed offer
	LastOfferState  OfferBase     // Last state of the offer before closing
	ClosedLedgerSeq uint32        // Ledger sequence where the offer was closed
}

func (e OfferClosedEvent) GetTradeEventType() TradeEventType {
	return TradeEventTypeOfferClosed
}

type LiquidityPoolUpdateEvent struct {
	Fills []FillInfo // List of fills for this liquidity pool update
}

func (e LiquidityPoolUpdateEvent) GetTradeEventType() TradeEventType {
	return TradeEventTypeLiquidityPoolUpdated
}

type FillSourceOperationType uint32

const (
	FillSourceOperationTypeUnknown FillSourceOperationType = iota
	FillSourceOperationTypeManageBuy
	FillSourceOperationTypeManageSell
	FillSourceOperationTypePathPaymentStrictSend
	FillSourceOperationTypePathPaymentStrictReceive
)

type FillSource struct {
	// Type of the operation that caused this fill (ManageBuyOffer,  ManageSellOffer, PathPaymentStrictSend, PathPaymentStrictReceive)
	SourceOperation FillSourceOperationType

	// The taker's information. Who caused this fill???
	ManageOfferInfo *ManageOfferInfo // Details of a ManageBuy/ManageSell operation (optional)
	PathPaymentInfo *PathPaymentInfo // Details of a PathPayment operation (optional)
}

// ManageBuy/ManageSell operation details
type ManageOfferInfo struct {
	// Account that initiated the operation. Source of operation or source of transaction
	SourceAccount xdr.AccountId

	// Did the taking operation create an offerId/offerEntry that rested after being partially filled OR Was it fully filled
	OfferFullyFilled bool

	OfferId *xdr.Int64 // Offer ID, if an offer entry was created (nil if fully filled)
}

type PathPaymentInfo struct {
	SourceAccount      xdr.AccountId // Source account of the PathPayment
	DestinationAccount xdr.AccountId // Destination account of the PathPayment
}

type FillInfo struct {
	AssetSold    xdr.Asset // Asset sold in this fill
	AmountSold   xdr.Int64 // Amount of the asset sold in this fill
	AssetBought  xdr.Asset // Asset bought in this fill
	AmountBought xdr.Int64 // Amount of the asset bought in this fill
	LedgerSeq    uint32    // Ledger sequence in which the fill occurred

	FillSourceInfo FillSource // Details about what operation (and details) caused this fill
}
