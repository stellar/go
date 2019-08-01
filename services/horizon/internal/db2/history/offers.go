package history

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/xdr"
)

// GetAllOffers loads a row from `history_accounts`, by address
func (q *Q) GetAllOffers() ([]Offer, error) {
	var offers []Offer
	err := q.Select(&offers, selectOffers)
	return offers, err
}

// UpsertOffer creates / updates a row in the offers table
func (q *Q) UpsertOffer(offer xdr.OfferEntry) error {
	var price float64
	if offer.Price.N > 0 {
		price = float64(offer.Price.N) / float64(offer.Price.D)
	}
	sql := sq.Insert("offers").
		Columns(
			"sellerid",
			"offerid",
			"sellingasset",
			"buyingasset",
			"amount",
			"pricen",
			"priced",
			"price",
			"flags",
		).
		Values(
			offer.SellerId,
			offer.OfferId,
			offer.Selling,
			offer.Buying,
			offer.Amount,
			offer.Price.N,
			offer.Price.D,
			price,
			offer.Flags,
		).
		Suffix(`
			ON CONFLICT (offerid) DO UPDATE SET
				sellerid=EXCLUDED.sellerid,
				sellingasset=EXCLUDED.sellingasset,
				buyingasset=EXCLUDED.buyingasset,
				amount=EXCLUDED.amount,
				pricen=EXCLUDED.pricen,
				priced=EXCLUDED.priced,
				price=EXCLUDED.price,
				flags=EXCLUDED.flags
		`)

	_, err := q.Exec(sql)
	return err
}

// RemoveOffer deletes a row in the offers table
func (q *Q) RemoveOffer(offerID xdr.Int64) error {
	sql := sq.Delete("offers").Where(sq.Eq{
		"id": offerID,
	})

	_, err := q.Exec(sql)
	return err
}

var selectOffers = sq.Select("o.*").From("offers o")
