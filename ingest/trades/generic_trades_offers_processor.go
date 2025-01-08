package trades

import (
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"io"
)

func ProcessTradesFromLedger(ledger xdr.LedgerCloseMeta, networkPassPhrase string) ([]TradeEvent, error) {
	changeReader, err := ingest.NewLedgerChangeReaderFromLedgerCloseMeta(networkPassPhrase, ledger)
	if err != nil {
		return []TradeEvent{}, errors.Wrap(err, "Error creating ledger change reader")
	}
	defer changeReader.Close()

	tradeEvents := make([]TradeEvent, 0)
	for {
		change, err := changeReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []TradeEvent{}, errors.Wrap(err, "Error reading ledger change")
		}
		// Process trades from the change
		tradesFromChange, err := processTradesFromChange(change)
		if err != nil {
			return nil, errors.Wrap(err, "Error processing trades from change")
		}

		// Append to the overall trade events slice
		tradeEvents = append(tradeEvents, tradesFromChange...)
	}

	return tradeEvents, nil
}

func processTradesFromChange(change ingest.Change) ([]TradeEvent, error) {
	var tradeEvents []TradeEvent

	switch change.Type {
	case xdr.LedgerEntryTypeOffer:
		trades, err := processOfferEventsFromChange(change)
		if err != nil {
			return nil, errors.Wrap(err, "Error processing offers")
		}
		tradeEvents = append(tradeEvents, trades...)
	case xdr.LedgerEntryTypeLiquidityPool:
		trades, err := processLiquidityPoolEventsFromChange(change)
		if err != nil {
			return nil, errors.Wrap(err, "Error processing liquidity pool events")
		}
		tradeEvents = append(tradeEvents, trades...)

	default:
		// Nothing to do
	}

	return tradeEvents, nil
}

func getOperationAndResultFromTransaction(index uint32, tx *ingest.LedgerTransaction) (xdr.Operation, xdr.OperationResult, error) {
	opResults, ok := tx.Result.OperationResults()
	if !ok {
		return xdr.Operation{}, xdr.OperationResult{}, errors.New("transaction has no operation results")
	}
	return tx.Envelope.Operations()[index], opResults[index], nil
}

func getClaimsFromOperationAndResult(op xdr.Operation, opResult xdr.OperationResult) ([]xdr.ClaimAtom, *xdr.OfferEntry) {
	var claims []xdr.ClaimAtom
	var offer *xdr.OfferEntry

	switch op.Body.Type {
	case xdr.OperationTypePathPaymentStrictReceive:
		claims, offer = opResult.MustTr().MustPathPaymentStrictReceiveResult().
			MustSuccess().
			Offers, nil

	case xdr.OperationTypePathPaymentStrictSend:
		claims, offer = opResult.MustTr().
			MustPathPaymentStrictSendResult().
			MustSuccess().
			Offers, nil

	case xdr.OperationTypeManageBuyOffer:
		manageOfferResult := opResult.MustTr().MustManageBuyOfferResult().
			MustSuccess()
		claims, offer = manageOfferResult.OffersClaimed, manageOfferResult.Offer.Offer

	case xdr.OperationTypeManageSellOffer:
		manageOfferResult := opResult.MustTr().MustManageSellOfferResult().
			MustSuccess()
		claims, offer = manageOfferResult.OffersClaimed, manageOfferResult.Offer.Offer

	case xdr.OperationTypeCreatePassiveSellOffer:
		result := opResult.MustTr()

		// KNOWN ISSUE:  stellar-core creates results for CreatePassiveOffer operations
		// with the wrong result arm set.
		if result.Type == xdr.OperationTypeManageSellOffer {
			manageOfferResult := result.MustManageSellOfferResult().MustSuccess()
			claims, offer = manageOfferResult.OffersClaimed, manageOfferResult.Offer.Offer

		} else {
			passiveOfferResult := result.MustCreatePassiveSellOfferResult().MustSuccess()
			claims, offer = passiveOfferResult.OffersClaimed, passiveOfferResult.Offer.Offer
		}
	default:
		// pass
	}
	return claims, offer
}

func groupClaimsByOfferId(claims []xdr.ClaimAtom) map[xdr.Int64]xdr.ClaimAtom {
	offerIdToClaimMap := make(map[xdr.Int64]xdr.ClaimAtom)
	for _, claim := range claims {
		offerIdToClaimMap[claim.OfferId()] = claim
	}
	return offerIdToClaimMap
}

func operationSourceAccount(tx *ingest.LedgerTransaction, op xdr.Operation) xdr.AccountId {
	var accountId xdr.AccountId
	if acc := op.SourceAccount; acc != nil {
		accountId = acc.ToAccountId()
	} else {
		accountId = tx.Envelope.SourceAccount().ToAccountId()
	}
	return accountId
}

func fillSourceInfo(tx *ingest.LedgerTransaction, op xdr.Operation, opResult xdr.OperationResult, takerOfferEntry *xdr.OfferEntry) FillSource {
	fillSource := FillSource{}

	// Helper function to create ManageOfferInfo
	createManageOfferInfo := func(operationType FillSourceOperationType) *ManageOfferInfo {
		offerInfo := &ManageOfferInfo{
			SourceAccount:    operationSourceAccount(tx, op),
			OfferFullyFilled: takerOfferEntry != nil,
		}
		if takerOfferEntry != nil {
			offerInfo.OfferId = &takerOfferEntry.OfferId
		} else {
			offerInfo.OfferId = nil
		}
		return offerInfo
	}

	// Helper function to create PathPaymentInfo
	createPathPaymentInfo := func(opType xdr.OperationType, opResult xdr.OperationResult) *PathPaymentInfo {
		var destinationAccount xdr.AccountId
		if opType == xdr.OperationTypePathPaymentStrictReceive {
			destinationAccount = opResult.MustTr().PathPaymentStrictReceiveResult.Success.Last.Destination
		} else if opType == xdr.OperationTypePathPaymentStrictSend {
			destinationAccount = opResult.MustTr().PathPaymentStrictSendResult.Success.Last.Destination
		}
		return &PathPaymentInfo{
			SourceAccount:      operationSourceAccount(tx, op),
			DestinationAccount: destinationAccount,
		}
	}

	// Switch on the operation type and use helper functions to reduce duplication
	switch op.Body.Type {
	case xdr.OperationTypePathPaymentStrictReceive:
		fillSource.SourceOperation = FillSourceOperationTypePathPaymentStrictSend
		fillSource.PathPaymentInfo = createPathPaymentInfo(xdr.OperationTypePathPaymentStrictReceive, opResult)

	case xdr.OperationTypePathPaymentStrictSend:
		fillSource.SourceOperation = FillSourceOperationTypePathPaymentStrictReceive
		fillSource.PathPaymentInfo = createPathPaymentInfo(xdr.OperationTypePathPaymentStrictSend, opResult)

	case xdr.OperationTypeManageBuyOffer:
		fillSource.SourceOperation = FillSourceOperationTypeManageBuy
		fillSource.ManageOfferInfo = createManageOfferInfo(FillSourceOperationTypeManageBuy)

	case xdr.OperationTypeManageSellOffer:
		fillSource.SourceOperation = FillSourceOperationTypeManageSell
		fillSource.ManageOfferInfo = createManageOfferInfo(FillSourceOperationTypeManageSell)

	case xdr.OperationTypeCreatePassiveSellOffer:
		fillSource.SourceOperation = FillSourceOperationTypePassiveSellOffer
		fillSource.ManageOfferInfo = createManageOfferInfo(FillSourceOperationTypePassiveSellOffer)
	}

	return fillSource
}

type FillInfo2 struct {
	AssetSold    xdr.Asset
	AmountSold   xdr.Int64
	AssetBought  xdr.Asset
	AmountBought xdr.Int64
	Type         // Either MatchingOffer or LiquidtiyPool
	OfferId      *int64
	SellerId     *AccountId

	PoolId *PoolId
}

func generateOfferFills(tx *ingest.LedgerTransaction, op xdr.Operation, opResult xdr.OperationResult, claimsMap []xdr.ClaimAtom, takerOfferEntry *xdr.OfferEntry) []FillInfo {
	fills := make([]FillInfo, 0)
	fillSource := fillSourceInfo(tx, op, opResult, takerOfferEntry)

	for _, claim := range claimsMap {
		switch claim.Type {
		case xdr.ClaimAtomTypeClaimAtomTypeV0, xdr.ClaimAtomTypeClaimAtomTypeOrderBook:
			fill := FillInfo{
				AssetSold:    claim.AssetSold(),
				AssetBought:  claim.AssetBought(),
				AmountSold:   claim.AmountSold(),
				AmountBought: claim.AmountBought(),
				LedgerSeq:    tx.Ledger.LedgerSequence(),
				// FillSourceInfo: fillSource,
			}
			fills = append(fills, fill)
		case xdr.ClaimAtomTypeClaimAtomTypeLiquidityPool:
			continue // You dont expect LiquidityPool type claim atoms to show up here
		}
	}
	return fills
}

func processOfferEventsFromChange(change ingest.Change) ([]TradeEvent, error) {
	if !(change.Type == xdr.LedgerEntryTypeOffer) {
		return nil, nil
	}

	//TODO: Handle the case where the an upgrade might have caused an eviction of offerEntry
	if !(change.Reason == ingest.LedgerEntryChangeReasonOperation) {
		return nil, nil
	}

	op, opResult, err := getOperationAndResultFromTransaction(change.OperationIndex, change.Transaction)
	if err != nil {
		return []TradeEvent{}, err
	}

	offerEvents := make([]TradeEvent, 0)
	claims, takerOfferEntry := getClaimsFromOperationAndResult(op, opResult)

	switch {
	case change.Pre == nil && change.Post != nil:
		// A new offer Entry is Created because of an operation. Might have partial fills
		// This is the case where there is a incoming new offer
		// You WONT be in this case statement, if an order was immediately filled, i.e if takerOfferEntry exists
		newOffer := change.Post.Data.MustOffer()
		offerEvent := OfferCreatedEvent{
			SellerId:         newOffer.SellerId,
			OfferId:          newOffer.OfferId, // offerID = 222 aka new offerId
			CreatedLedgerSeq: uint32(change.Post.LastModifiedLedgerSeq),
			OfferState:       newOffer,
			Fills:            generateOfferFills(change.Transaction, op, opResult, claims, nil),
		}
		offerEvents = append(offerEvents, offerEvent)

	case change.Pre != nil && change.Post != nil:
		// An existing offer is updated. Might have fills
		// You will be in this case only if you are an already existing order, i.e maker
		oldOffer := change.Pre.Data.MustOffer()
		updatedOffer := change.Post.Data.MustOffer()
		offerEvent := OfferUpdatedEvent{
			SellerId:             oldOffer.SellerId,
			OfferId:              oldOffer.OfferId,
			UpdatedLedgerSeq:     uint32(change.Post.LastModifiedLedgerSeq),
			PrevUpdatedLedgerSeq: uint32(change.Pre.LastModifiedLedgerSeq),
			PreviousOfferState:   oldOffer,
			UpdatedOfferState:    updatedOffer,
			Fills:                generateOfferFills(change.Transaction, op, opResult, claims, takerOfferEntry),
		}
		offerEvents = append(offerEvents, offerEvent)

	case change.Pre != nil && change.Post == nil:
		oldOffer := change.Pre.Data.MustOffer()
		fills := generateOfferFills(change.Transaction, op, opResult, claims, takerOfferEntry)
		var closeReason OfferCloseReason
		if len(fills) > 0 {
			closeReason = OfferCloseReasonOfferFullyFilled
		} else {
			closeReason = OfferCloseReasonOfferCancelled
		}
		offerEvent := OfferClosedEvent{
			SellerId:             oldOffer.SellerId,
			OfferId:              oldOffer.OfferId,
			PreviousOfferState:   oldOffer,
			PrevUpdatedLedgerSeq: uint32(change.Pre.LastModifiedLedgerSeq),
			ClosedLedgerSeq:      change.Ledger.LedgerSequence(),
			CloseReason:          closeReason,
			Fills:                fills,
		}
		offerEvents = append(offerEvents, offerEvent)
	}

	return offerEvents, nil
}

func processLiquidityPoolEventsFromChange(change ingest.Change) ([]TradeEvent, error) {
	if !(change.Type == xdr.LedgerEntryTypeLiquidityPool) {
		return nil, nil
	}
	return []TradeEvent{}, nil
}
