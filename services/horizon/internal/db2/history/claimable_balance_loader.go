package history

import (
	"context"
	"database/sql/driver"
	"fmt"
	"sort"

	"github.com/stellar/go/support/collections/set"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/ordered"
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
	return a.loader.getNow(a.id)
}

// ClaimableBalanceLoader will map claimable balance ids to their internal
// history ids. If there is no existing mapping for a given claimable balance id,
// the ClaimableBalanceLoader will insert into the history_claimable_balances table to
// establish a mapping.
type ClaimableBalanceLoader struct {
	sealed bool
	set    set.Set[string]
	ids    map[string]int64
}

// NewClaimableBalanceLoader will construct a new ClaimableBalanceLoader instance.
func NewClaimableBalanceLoader() *ClaimableBalanceLoader {
	return &ClaimableBalanceLoader{
		sealed: false,
		set:    set.Set[string]{},
		ids:    map[string]int64{},
	}
}

// GetFuture registers the given claimable balance into the loader and
// returns a FutureClaimableBalanceID which will hold the internal history id for
// the claimable balance after Exec() is called.
func (a *ClaimableBalanceLoader) GetFuture(id string) FutureClaimableBalanceID {
	if a.sealed {
		panic(errSealed)
	}

	a.set.Add(id)
	return FutureClaimableBalanceID{
		id:     id,
		loader: a,
	}
}

// getNow returns the internal history id for the given claimable balance.
// getNow should only be called on values which were registered by
// GetFuture() calls. Also, Exec() must be called before any getNow
// call can succeed.
func (a *ClaimableBalanceLoader) getNow(id string) (int64, error) {
	if !a.sealed {
		return 0, fmt.Errorf(`invalid claimable balance loader state, 
		Exec was not called yet to properly seal and resolve %v id`, id)
	}
	if internalID, ok := a.ids[id]; !ok {
		return 0, fmt.Errorf(`claimable balance loader id %q was not found`, id)
	} else {
		return internalID, nil
	}
}

func (a *ClaimableBalanceLoader) lookupKeys(ctx context.Context, q *Q, ids []string) error {
	for i := 0; i < len(ids); i += loaderLookupBatchSize {
		end := ordered.Min(len(ids), i+loaderLookupBatchSize)

		cbs, err := q.ClaimableBalancesByIDs(ctx, ids[i:end])
		if err != nil {
			return errors.Wrap(err, "could not select claimable balances")
		}

		for _, cb := range cbs {
			a.ids[cb.BalanceID] = cb.InternalID
		}
	}
	return nil
}

// Exec will look up all the internal history ids for the claimable balances registered in the loader.
// If there are no internal ids for a given set of claimable balances, Exec will insert rows
// into the history_claimable_balances table.
func (a *ClaimableBalanceLoader) Exec(ctx context.Context, session db.SessionInterface) error {
	a.sealed = true
	if len(a.set) == 0 {
		return nil
	}
	q := &Q{session}
	ids := make([]string, 0, len(a.set))
	for id := range a.set {
		ids = append(ids, id)
	}

	if err := a.lookupKeys(ctx, q, ids); err != nil {
		return err
	}

	insert := 0
	for _, id := range ids {
		if _, ok := a.ids[id]; ok {
			continue
		}
		ids[insert] = id
		insert++
	}
	if insert == 0 {
		return nil
	}
	ids = ids[:insert]
	// sort entries before inserting rows to prevent deadlocks on acquiring a ShareLock
	// https://github.com/stellar/go/issues/2370
	sort.Strings(ids)

	err := bulkInsert(
		ctx,
		q,
		"history_claimable_balances",
		[]string{"claimable_balance_id"},
		[]bulkInsertField{
			{
				name:    "claimable_balance_id",
				dbType:  "text",
				objects: ids,
			},
		},
	)
	if err != nil {
		return err
	}

	return a.lookupKeys(ctx, q, ids)
}
