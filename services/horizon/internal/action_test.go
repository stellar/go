package horizon

import (
	"context"
	"testing"
	"time"

	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/support/render/hal"
)

var defaultPage db2.PageQuery = db2.PageQuery{
	Order:  db2.OrderAscending,
	Limit:  db2.DefaultPageSize,
	Cursor: "",
}

func TestGetTransactionPage(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	ctx := context.Background()
	w := mustInitWeb(ctx, &history.Q{tt.HorizonSession()}, time.Duration(5), 0, true)

	// filter by account
	params := &indexActionQueryParams{
		AccountID:        "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		PagingParams:     defaultPage,
		IncludeFailedTxs: true,
	}

	page, err := w.getTransactionPage(ctx, params)
	pageVal, ok := page.(hal.Page)
	if !ok {
		tt.Assert.FailNow("returned type mismatch")
	}
	tt.Assert.NoError(err)
	tt.Assert.Equal(3, len(pageVal.Embedded.Records))

	// filter by ledger
	params = &indexActionQueryParams{
		LedgerID:         3,
		PagingParams:     defaultPage,
		IncludeFailedTxs: true,
	}

	page, err = w.getTransactionPage(ctx, params)
	pageVal, ok = page.(hal.Page)
	if !ok {
		tt.Assert.FailNow("returned type mismatch")
	}
	tt.Assert.NoError(err)
	tt.Assert.Equal(1, len(pageVal.Embedded.Records))

	// no filter
	params = &indexActionQueryParams{
		PagingParams:     defaultPage,
		IncludeFailedTxs: true,
	}

	page, err = w.getTransactionPage(ctx, params)
	pageVal, ok = page.(hal.Page)
	if !ok {
		tt.Assert.FailNow("returned type mismatch")
	}
	tt.Assert.NoError(err)
	tt.Assert.Equal(4, len(pageVal.Embedded.Records))
}
