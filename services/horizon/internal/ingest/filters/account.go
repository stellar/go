package filters

import (
	"context"
	"encoding/json"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/processors"
	"github.com/stellar/go/support/errors"
)

// TODO:(fons) I don't think we should be using a singleton
// (we should just create an instance which lives in the processor)
var accountFilter = &AccountFilter{
	whitelistedAccountsSet: map[string]struct{}{},
	lastModified:           0,
}

type AccountFilterRules struct {
	CanonicalWhitelist []string `json:"account_whitelist"`
}

type AccountFilter struct {
	whitelistedAccountsSet map[string]struct{}
	lastModified           int64
}

// TODO:(fons) this code should probably be generic for all filters
func GetAccountFilter(filterConfig *history.FilterConfig) (*AccountFilter, error) {
	// only need to re-initialize the filter config state(rules) if it's cached version(in  memory)
	// is older than the incoming config version based on lastModified epoch timestamp
	if filterConfig.LastModified > accountFilter.lastModified {
		var assetFilterRules AssetFilterRules
		if err := json.Unmarshal([]byte(filterConfig.Rules), &assetFilterRules); err != nil {
			return nil, errors.Wrap(err, "unable to serialize asset filter rules")
		}
		accountFilter = &AccountFilter{
			whitelistedAccountsSet: listToMap(assetFilterRules.CanonicalWhitelist),
			lastModified:           filterConfig.LastModified,
		}
	}

	return accountFilter, nil
}

func (f *AccountFilter) FilterTransaction(ctx context.Context, transaction ingest.LedgerTransaction) (bool, error) {
	// Whitelisting is disabled if the whitelist is empty
	if len(f.whitelistedAccountsSet) == 0 {
		return true, nil
	}

	// TODO: what is the sequence used for?
	participants, err := processors.ParticipantsForTransaction(0, transaction)
	if err != nil {
		return false, err
	}

	// NOTE: this assumes that the participant list has a small memory footprint
	//       otherwise, we should be doing the filtering on the DB side
	for _, p := range participants {
		if _, ok := f.whitelistedAccountsSet[p.Address()]; ok {
			return true, nil
		}
	}
	return false, nil
}
