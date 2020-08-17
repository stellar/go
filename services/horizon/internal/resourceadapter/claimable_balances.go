package resourceadapter

import (
	"context"
	"fmt"

	"github.com/stellar/go/amount"
	protocol "github.com/stellar/go/protocols/horizon"
	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/xdr"
)

// PopulateClaimableBalance fills out the resource's fields
func PopulateClaimableBalance(
	ctx context.Context,
	dest *protocol.ClaimableBalance,
	claimableBalance history.ClaimableBalance,
) error {
	dest.ID = claimableBalance.ID
	balanceID, err := xdr.MarshalBase64(claimableBalance.BalanceID)
	if err != nil {
		return errors.Wrap(err, "marshalling BalanceID")
	}
	dest.BalanceID = balanceID
	dest.Asset = claimableBalance.Asset.StringCanonical()
	dest.Amount = amount.StringFromInt64(int64(claimableBalance.Amount))
	if claimableBalance.Sponsor.Valid {
		dest.Sponsor = claimableBalance.Sponsor.String
	}
	dest.LastModifiedLedger = claimableBalance.LastModifiedLedger
	dest.Claimants = make([]protocol.Claimant, len(claimableBalance.Claimants))
	for i, c := range claimableBalance.Claimants {
		predicate, err := xdr.MarshalBase64(c.Predicate)
		if err != nil {
			errors.Wrap(err, "failed to encode predicate to base64")
		}
		dest.Claimants[i].Destination = c.Destination
		dest.Claimants[i].Predicate = predicate
	}

	lb := hal.LinkBuilder{Base: horizonContext.BaseURL(ctx)}
	self := fmt.Sprintf("/claimable_balances/%s", claimableBalance.ID)
	dest.Links.Self = lb.Link(self)
	dest.PT = fmt.Sprintf("%d-%s", claimableBalance.LastModifiedLedger, claimableBalance.ID)
	return nil
}
