package history

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/support/errors"
)

// QOffers defines offer related queries.
type QOffers interface {
	GetAllOffers() ([]Offer, error)
	GetOffersByIDs(ids []int64) ([]Offer, error)
	CountOffers() (int, error)
	GetUpdatedOffers(newerThanSequence uint32) ([]Offer, error)
	NewOffersBatchInsertBuilder(maxBatchSize int) OffersBatchInsertBuilder
	UpdateOffer(offer Offer) (int64, error)
	RemoveOffers(offerIDs []int64, lastModifiedLedger uint32) (int64, error)
	CompactOffers(cutOffSequence uint32) (int64, error)
}

func (q *Q) CountOffers() (int, error) {
	sql := sq.Select("count(*)").Where("deleted = ?", false).From("offers")

	var count int
	if err := q.Get(&count, sql); err != nil {
		return 0, errors.Wrap(err, "could not run select query")
	}

	return count, nil
}

// GetOfferByID loads a row from the `offers` table, selected by offerid.
func (q *Q) GetOfferByID(id int64) (Offer, error) {
	var offer Offer
	sql := selectOffers.Where("deleted = ?", false).
		Where("offers.offer_id = ?", id)
	err := q.Get(&offer, sql)
	return offer, err
}

// GetOffersByIDs loads a row from the `offers` table, selected by multiple offerid.
func (q *Q) GetOffersByIDs(ids []int64) ([]Offer, error) {
	var offers []Offer
	sql := selectOffers.Where("deleted = ?", false).
		Where(map[string]interface{}{"offers.offer_id": ids})
	err := q.Select(&offers, sql)
	return offers, err
}

// GetOffers loads rows from `offers` by paging query.
func (q *Q) GetOffers(query OffersQuery) ([]Offer, error) {
	sql := selectOffers.Where("deleted = ?", false)
	sql, err := query.PageQuery.ApplyTo(sql, "offers.offer_id")

	if err != nil {
		return nil, errors.Wrap(err, "could not apply query to page")
	}

	if query.SellerID != "" {
		sql = sql.Where("offers.seller_id = ?", query.SellerID)
	}

	if query.Selling != nil {
		sql = sql.Where("offers.selling_asset = ?", query.Selling)
	}

	if query.Buying != nil {
		sql = sql.Where("offers.buying_asset = ?", query.Buying)
	}

	if query.Sponsor != "" {
		sql = sql.Where("offers.sponsor = ?", query.Sponsor)
	}

	var offers []Offer
	if err := q.Select(&offers, sql); err != nil {
		return nil, errors.Wrap(err, "could not run select query")
	}

	return offers, nil
}

// GetAllOffers loads all non deleted offers
func (q *Q) GetAllOffers() ([]Offer, error) {
	var offers []Offer
	err := q.Select(&offers, selectOffers.Where("deleted = ?", false))
	return offers, err
}

// GetUpdatedOffers returns all offers created, updated, or deleted after the given ledger sequence.
func (q *Q) GetUpdatedOffers(newerThanSequence uint32) ([]Offer, error) {
	var offers []Offer
	err := q.Select(&offers, selectOffers.Where("offers.last_modified_ledger > ?", newerThanSequence))
	return offers, err
}

// UpdateOffer updates a row in the offers table.
// Returns number of rows affected and error.
func (q *Q) UpdateOffer(offer Offer) (int64, error) {
	updateBuilder := q.GetTable("offers").Update()
	result, err := updateBuilder.SetStruct(offer, []string{}).Where("offer_id = ?", offer.OfferID).Exec()
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// RemoveOffers marks rows in the offers table as deleted.
// Returns number of rows affected and error.
func (q *Q) RemoveOffers(offerIDs []int64, lastModifiedLedger uint32) (int64, error) {
	sql := sq.Update("offers").
		Set("deleted", true).
		Set("last_modified_ledger", lastModifiedLedger).
		Where(map[string]interface{}{"offer_id": offerIDs})

	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// CompactOffers removes rows from the offers table which are marked for deletion.
func (q *Q) CompactOffers(cutOffSequence uint32) (int64, error) {
	sql := sq.Delete("offers").
		Where("deleted = ?", true).
		Where("last_modified_ledger <= ?", cutOffSequence)

	result, err := q.Exec(sql)
	if err != nil {
		return 0, errors.Wrap(err, "cannot delete offer rows")
	}

	if err = q.UpdateOfferCompactionSequence(cutOffSequence); err != nil {
		return 0, errors.Wrap(err, "cannot update offer compaction sequence")
	}

	return result.RowsAffected()
}

var selectOffers = sq.Select(`
	seller_id,
	offer_id,
	selling_asset,
	buying_asset,
	amount,
	pricen,
	priced,
	price,
	flags,
	deleted,
	last_modified_ledger,
	sponsor
`).From("offers")
