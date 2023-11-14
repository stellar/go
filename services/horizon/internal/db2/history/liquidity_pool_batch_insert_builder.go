package history

import (
	"context"

	"github.com/stellar/go/support/db"
)

type LiquidityPoolBatchInsertBuilder interface {
	Add(liquidityPool LiquidityPool) error
	Exec(ctx context.Context) error
}

type liquidityPoolBatchInsertBuilder struct {
	session db.SessionInterface
	builder db.FastBatchInsertBuilder
	table   string
}

func (q *Q) NewLiquidityPoolBatchInsertBuilder() LiquidityPoolBatchInsertBuilder {
	return &liquidityPoolBatchInsertBuilder{
		session: q,
		builder: db.FastBatchInsertBuilder{},
		table:   "liquidity_pools",
	}
}

// Add adds a new liquidity pool to the batch
func (i *liquidityPoolBatchInsertBuilder) Add(liquidityPool LiquidityPool) error {
	return i.builder.RowStruct(liquidityPool)
}

// Exec writes the batch of liquidity pools to the database.
func (i *liquidityPoolBatchInsertBuilder) Exec(ctx context.Context) error {
	return i.builder.Exec(ctx, i.session, i.table)
}
