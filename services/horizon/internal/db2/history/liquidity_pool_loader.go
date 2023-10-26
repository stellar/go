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

// FutureLiquidityPoolID represents a future history liquidity pool.
// A FutureLiquidityPoolID is created by an LiquidityPoolLoader and
// the liquidity pool id is available after calling Exec() on
// the LiquidityPoolLoader.
type FutureLiquidityPoolID struct {
	id     string
	loader *LiquidityPoolLoader
}

// Value implements the database/sql/driver Valuer interface.
func (a FutureLiquidityPoolID) Value() (driver.Value, error) {
	return a.loader.GetNow(a.id)
}

// LiquidityPoolLoader will map liquidity pools to their internal
// history ids. If there is no existing mapping for a given liquidity pool,
// the LiquidityPoolLoader will insert into the history_liquidity_pools table to
// establish a mapping.
type LiquidityPoolLoader struct {
	sealed bool
	set    set.Set[string]
	ids    map[string]int64
}

// NewLiquidityPoolLoader will construct a new LiquidityPoolLoader instance.
func NewLiquidityPoolLoader() *LiquidityPoolLoader {
	return &LiquidityPoolLoader{
		sealed: false,
		set:    set.Set[string]{},
		ids:    map[string]int64{},
	}
}

// GetFuture registers the given liquidity pool into the loader and
// returns a FutureLiquidityPoolID which will hold the internal history id for
// the liquidity pool after Exec() is called.
func (a *LiquidityPoolLoader) GetFuture(id string) FutureLiquidityPoolID {
	if a.sealed {
		panic(errSealed)
	}

	a.set.Add(id)
	return FutureLiquidityPoolID{
		id:     id,
		loader: a,
	}
}

// GetNow returns the internal history id for the given liquidity pool.
// GetNow should only be called on values which were registered by
// GetFuture() calls. Also, Exec() must be called before any GetNow
// call can succeed.
func (a *LiquidityPoolLoader) GetNow(id string) (int64, error) {
	if !a.sealed {
		return 0, fmt.Errorf(`invalid liquidity pool loader state,  
		Exec was not called yet to properly seal and resolve %v id`, id)
	}
	if internalID, ok := a.ids[id]; !ok {
		return 0, fmt.Errorf(`liquidity pool loader id %q was not found`, id)
	} else {
		return internalID, nil
	}
}

func (a *LiquidityPoolLoader) lookupKeys(ctx context.Context, q *Q, ids []string) error {
	for i := 0; i < len(ids); i += loaderLookupBatchSize {
		end := ordered.Min(len(ids), i+loaderLookupBatchSize)

		lps, err := q.LiquidityPoolsByIDs(ctx, ids[i:end])
		if err != nil {
			return errors.Wrap(err, "could not select accounts")
		}

		for _, lp := range lps {
			a.ids[lp.PoolID] = lp.InternalID
		}
	}
	return nil
}

// Exec will look up all the internal history ids for the liquidity pools registered in the loader.
// If there are no internal history ids for a given set of liquidity pools, Exec will insert rows
// into the history_liquidity_pools table.
func (a *LiquidityPoolLoader) Exec(ctx context.Context, session db.SessionInterface) error {
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
		"history_liquidity_pools",
		[]string{"liquidity_pool_id"},
		[]bulkInsertField{
			{
				name:    "liquidity_pool_id",
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

// LiquidityPoolLoaderStub is a stub wrapper around LiquidityPoolLoader which allows
// you to manually configure the mapping of liquidity pools to history liquidity ppol ids
type LiquidityPoolLoaderStub struct {
	Loader *LiquidityPoolLoader
}

// NewLiquidityPoolLoaderStub returns a new LiquidityPoolLoader instance
func NewLiquidityPoolLoaderStub() LiquidityPoolLoaderStub {
	return LiquidityPoolLoaderStub{Loader: NewLiquidityPoolLoader()}
}

// Insert updates the wrapped LiquidityPoolLoader so that the given liquidity pool
// is mapped to the provided history liquidity pool id
func (a LiquidityPoolLoaderStub) Insert(lp string, id int64) {
	a.Loader.sealed = true
	a.Loader.ids[lp] = id
}
