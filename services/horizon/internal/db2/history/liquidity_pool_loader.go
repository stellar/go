package history

import (
	"context"
	"database/sql/driver"
	"sort"

	"github.com/stellar/go/support/collections/set"
	"github.com/stellar/go/support/errors"
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
	return a.loader.GetNow(a.id), nil
}

// LiquidityPoolLoader will map liquidity pools to their internal
// history ids. If there is no existing mapping for a given liquidity pool,
// the LiquidityPoolLoader will insert into the history_liquidity_pools table to
// establish a mapping.
type LiquidityPoolLoader struct {
	loader[string, FutureLiquidityPoolID]
}

// NewLiquidityPoolLoader will construct a new LiquidityPoolLoader instance.
func NewLiquidityPoolLoader() *LiquidityPoolLoader {
	l := &LiquidityPoolLoader{
		loader: loader[string, FutureLiquidityPoolID]{
			sealed: false,
			set:    set.Set[string]{},
			ids:    map[string]int64{},
			sort:   sort.Strings,
			insert: func(ctx context.Context, q *Q, keys []string) error {
				return bulkInsert(
					ctx,
					q,
					"history_liquidity_pools",
					[]string{"liquidity_pool_id"},
					[]bulkInsertField{
						{
							name:    "liquidity_pool_id",
							dbType:  "text",
							objects: keys,
						},
					},
				)
			},
		},
	}
	l.fetchAndUpdate = func(ctx context.Context, q *Q, keys []string) error {
		lps, err := q.LiquidityPoolsByIDs(ctx, keys)
		if err != nil {
			return errors.Wrap(err, "could not select accounts")
		}

		for _, lp := range lps {
			l.ids[lp.PoolID] = lp.InternalID
		}
		return nil
	}
	l.newFuture = func(key string) FutureLiquidityPoolID {
		return FutureLiquidityPoolID{
			id:     key,
			loader: l,
		}
	}

	return l
}
