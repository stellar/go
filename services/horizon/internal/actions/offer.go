package actions

import (
	"context"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/errors"
)

// OfferResource returns a single offer resource identified by offerID.
func OfferResource(ctx context.Context, hq *history.Q, offerID int64) (horizon.Offer, error) {
	var resource horizon.Offer

	record, err := hq.GetOfferByID(offerID)

	if err != nil {
		return resource, errors.Wrap(err, "loading offer record")
	}

	resourceadapter.PopulateHistoryOffer(ctx, &resource, record)
	return resource, nil
}
