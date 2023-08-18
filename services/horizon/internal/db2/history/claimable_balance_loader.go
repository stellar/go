package history

import (
	"context"
	"database/sql/driver"
	"sort"

	"github.com/stellar/go/support/collections/set"
	"github.com/stellar/go/support/errors"
)

// FutureClaimableBalanceID represents a future history claimable balance.
// A FutureClaimableBalanceID is created by a ClaimableBalanceLoader and
// the claimable balance id is available after calling Exec() on
// the ClaimableBalanceLoader.
type FutureClaimableBalanceID struct {
	id     string
	loader *ClaimableBalanceLoader
}

// Value implements the database/sql/driver Valuer interface.
func (a FutureClaimableBalanceID) Value() (driver.Value, error) {
	return a.loader.GetNow(a.id), nil
}

// ClaimableBalanceLoader will map claimable balance ids to their internal
// history ids. If there is no existing mapping for a given claimable balance id,
// the ClaimableBalanceLoader will insert into the history_claimable_balances table to
// establish a mapping.
type ClaimableBalanceLoader struct {
	loader[string, FutureClaimableBalanceID]
}

// NewClaimableBalanceLoader will construct a new ClaimableBalanceLoader instance.
func NewClaimableBalanceLoader() *ClaimableBalanceLoader {
	l := &ClaimableBalanceLoader{
		loader: loader[string, FutureClaimableBalanceID]{
			sealed: false,
			set:    set.Set[string]{},
			ids:    map[string]int64{},
			sort:   sort.Strings,
			insert: func(ctx context.Context, q *Q, keys []string) error {
				return bulkInsert(
					ctx,
					q,
					"history_claimable_balances",
					[]string{"claimable_balance_id"},
					[]bulkInsertField{
						{
							name:    "claimable_balance_id",
							dbType:  "text",
							objects: keys,
						},
					},
				)
			},
		},
	}
	l.fetchAndUpdate = func(ctx context.Context, q *Q, keys []string) error {
		cbs, err := q.ClaimableBalancesByIDs(ctx, keys)
		if err != nil {
			return errors.Wrap(err, "could not select claimable balances")
		}

		for _, cb := range cbs {
			l.ids[cb.BalanceID] = cb.InternalID
		}
		return nil
	}
	l.newFuture = func(key string) FutureClaimableBalanceID {
		return FutureClaimableBalanceID{
			id:     key,
			loader: l,
		}
	}

	return l
}
