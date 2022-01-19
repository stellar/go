package processors

import (
	"context"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
)

type AccountFilter struct {
	q history.QAccountFilterWhitelist
}

func NewAccountFilter(q history.QAccountFilterWhitelist) *AccountFilter {
	return &AccountFilter{q}
}

func (a *AccountFilter) FilterTransaction(ctx context.Context, transaction ingest.LedgerTransaction) (bool, error) {
	// TODO: we should cache this set
	whitelistedAccounts, err := a.q.GetAccountFilterWhitelist(ctx)
	if err != nil {
		return false, err
	}
	whitelistedAccountsSet := map[string]struct{}{}
	for _, account := range whitelistedAccounts {
		whitelistedAccountsSet[account] = struct{}{}
	}

	// TODO: what is the sequence used for?
	participants, err := participantsForTransaction(0, transaction)
	if err != nil {
		return false, err
	}

	// NOTE: this assumes that the participant list is small
	for _, p := range participants {
		if _, ok := whitelistedAccountsSet[p.Address()]; ok {
			return true, nil
		}
	}
	return false, nil
}
