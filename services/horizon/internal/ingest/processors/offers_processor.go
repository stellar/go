package processors

import (
	"context"
	"fmt"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/offers"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
)

// The offers processor can be configured to trim the offers table
// by removing all offer rows which were marked for deletion at least 100 ledgers ago
const compactionWindow = uint32(100)

type OffersProcessor struct {
	offersQ  history.QOffers
	sequence uint32

	batchUpdateOffers  []history.Offer
	insertBatchBuilder history.OffersBatchInsertBuilder
}

func NewOffersProcessor(offersQ history.QOffers, sequence uint32) *OffersProcessor {
	p := &OffersProcessor{offersQ: offersQ, sequence: sequence}
	p.reset()
	return p
}

func (p *OffersProcessor) Name() string {
	return "processors.OffersProcessor"
}

func (p *OffersProcessor) reset() {
	p.batchUpdateOffers = []history.Offer{}
	p.insertBatchBuilder = p.offersQ.NewOffersBatchInsertBuilder()
}

func (p *OffersProcessor) ProcessChange(ctx context.Context, change ingest.Change) error {
	event := offers.ProcessOffer(change)

	xx := offers.ProcessOffer(change)

	switch xx.OfferEventType() {
	case offers.OfferCreatedEventType:
		ev := (event).(offers.OfferCreatedEvent)
		fmt.Println(ev)

	case offers.OfferUpdatedEventType:
	case offers.OfferClosedEventType:

	}

	if event == nil {
		return nil
	}

	switch evv := event.(type) {
	case offers.OfferCreatedEvent:

	}

	switch ev := event.(type) {
	case offers.OfferCreatedEvent:
		row := p.offerEventToRow(ev.OfferEventData)
		err := p.insertBatchBuilder.Add(row)
		if err != nil {
			return errors.New("Error adding to OffersBatchInsertBuilder")
		}
	case offers.OfferFillEvent:
		row := p.offerEventToRow(ev.OfferEventData)
		p.batchUpdateOffers = append(p.batchUpdateOffers, row)
	case offers.OfferClosedEvent:
		row := p.offerEventToRow(ev.OfferEventData)
		row.Deleted = true
		row.LastModifiedLedger = p.sequence
		p.batchUpdateOffers = append(p.batchUpdateOffers, row)
	default:
		return errors.New("Unknown offer event")

	}
	if p.insertBatchBuilder.Len()+len(p.batchUpdateOffers) > maxBatchSize {
		if err := p.flushCache(ctx); err != nil {
			return errors.Wrap(err, "error in Commit")
		}
	}

	return nil

}

func (p *OffersProcessor) offerEventToRow(e offers.OfferEventData) history.Offer {
	return history.Offer{
		SellerID:           e.SellerId,
		OfferID:            e.OfferID,
		SellingAsset:       e.SellingAsset,
		BuyingAsset:        e.BuyingAsset,
		Amount:             e.RemainingAmount,
		Pricen:             e.PriceN,
		Priced:             e.PriceD,
		Price:              float64(e.PriceN) / float64(e.PriceD),
		Flags:              e.Flags,
		LastModifiedLedger: e.LastModifiedLedger,
		Sponsor:            e.Sponsor,
	}
}

func (p *OffersProcessor) flushCache(ctx context.Context) error {
	defer p.reset()

	err := p.insertBatchBuilder.Exec(ctx)
	if err != nil {
		return errors.New("Error executing OffersBatchInsertBuilder")
	}

	if len(p.batchUpdateOffers) > 0 {
		err := p.offersQ.UpsertOffers(ctx, p.batchUpdateOffers)
		if err != nil {
			return errors.Wrap(err, "errors in UpsertOffers")
		}
	}

	return nil
}

func (p *OffersProcessor) Commit(ctx context.Context) error {
	if err := p.flushCache(ctx); err != nil {
		return errors.Wrap(err, "error flushing cache")
	}

	if p.sequence > compactionWindow {
		// trim offers table by removing offers which were deleted before the cutoff ledger
		if offerRowsRemoved, err := p.offersQ.CompactOffers(ctx, p.sequence-compactionWindow); err != nil {
			return errors.Wrap(err, "could not compact offers")
		} else {
			log.WithField("offer_rows_removed", offerRowsRemoved).Info("Trimmed offers table")
		}
	}

	return nil
}

/*

	I want to model offers and traders together,coz, in my mind, they are the same, in that, a trade is just an orderFillEvent

	OfferCreated --> a event emitted when a new ManageBuy/Sell operation is done

	OfferFillEvent --> an event emitted when there is a fill, will have a fillInfo
						FillInfo wil have info about restingOrder or LiquidityPool

	OfferUpdateEvent --> if an existing ogffer is updated by user interaction

	OfferCancelEvent --> when an offer is cancelled either by user interaction OR by system itself - for e.g revocatin of issing asset accounts
											trustline


	OFferCloseEvent --> this is to indicate that there will be no event for that offer anymore.
						CloseReason == offer fully filled OR offer cancelled coz of manaul cancellation or offer cancelled coz of system



10 BTC already being on orderbook -- offerId-123
I create a MangeBuy to buy 2 BTC.


In the operationMeta, there will be a changeEntry for OfferId=123, pre=10, post = 8
But there will be no entry for the new operation

1  Sell BTC

OfferId=12345


10 Buy BTC

OfferId=12345 ceases to exist


Tx = 1, Sell 2 BTC

ChangeEntry after tx-1:
prestate=null
postState=OferId=456-Tx1


------------------------
Tx = 2, Buy 20 BTC
preState = offerId=456-Tx1,
postState=null

preState=Null
postState=OfferId-789-Tx-2, amount = 18


extint sate -- soemone sells 10 BTC
somene else is selling 5 BTC
OfferId=123

1 txhas 1 operationto buy 15 BTC - MAngeBuy.made by Karthik

operationMeta will have 1 changeEntry
OfferId=123, pre=10, post=8
opfferId=46. pre=5, post=0

What I am saying is - there will be no offerEntry created with an offerId = Karthiks' offer, coz
when the operation was applied, Karthik as a taker was fully filled.


-- a NEW Offer filled fully in a same ledger -- OfferCreated, OfferFill, OfferClose

-- a new offer is partiallu filled in same ledger -- OfferCreated, OfferFill

-- a existing offer is updated (price/quantity is changed), but no activity happens for that offer other than that ---
		OfferUpdated

-- a existing offer is updated (price/quantity is changed), and coz of that it is partially or fully filled
	partiallyfilled -- OfferUpdated, OfferFill
	fullyfilled - Offerupdated, OfferFill, OfferClose

-- offer is cancelled by user
	OfferCancel, OfferClose

-- offer is cancelled by system:
	OfferCancel, OfferClose


*/

/*

	func processAllEventsInLedger(lcm xdr.LcM) []OfferEvents {

	changeReder = non-compatcint_change_reasder
	var []offerEvents
	for change in changes:
		if changeType = Offer
			add 1 or more events to Offer

	return offerEvents
}

}

*/

/*


	Exisint offer 10  BTC

	New offer to buy 2 BTC - ManageBuyOperation

	There will be no change entry for the new offer, if it is filled

	Buy the resing offer just changed from 10 to 8. Change Entry - pre== 10  BTC, post = 8 BTC
		within this changes operation meta, there will be a claimAtom









*/
