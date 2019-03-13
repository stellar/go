package actions

import (
	"context"

	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/render/hal"
)

type TransactionParams struct {
	AccountFilter string
	LedgerFilter  int32
}

func TransactionPageByAccount(ctx context.Context, cq *core.Q, hq *history.Q, addr string) (hal.Page, error) {
	return hal.Page{}, nil
}
