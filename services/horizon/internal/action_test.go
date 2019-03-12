package horizon

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/support/errors"
)

func TestGetAccountInfo(t *testing.T) {
	tt := test.Start(t).Scenario("allow_trust")
	defer tt.Finish()

	w := mustInitWeb(context.Background(), &history.Q{tt.HorizonSession()}, &core.Q{tt.CoreSession()}, time.Duration(5))

	res, err := w.getAccountInfo(tt.Ctx, "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU")
	tt.Assert.NoError(err)

	account, ok := res.(*horizon.Account)
	if !ok {
		tt.Assert.FailNow("type assertion failed when getting account info")
	}

	tt.Assert.Equal("8589934593", account.Sequence)
	tt.Assert.NotEqual(0, account.LastModifiedLedger)

	for _, balance := range account.Balances {
		if balance.Type == "native" {
			tt.Assert.Equal(uint32(0), balance.LastModifiedLedger)
		} else {
			tt.Assert.NotEqual(uint32(0), balance.LastModifiedLedger)
		}
	}

	_, err = w.getAccountInfo(tt.Ctx, "GDBAPLDCAEJV6LSEDFEAUDAVFYSNFRUYZ4X75YYJJMMX5KFVUOHX46SQ")
	tt.Assert.Equal(errors.Cause(err), sql.ErrNoRows)
}

func TestLoadAccountEvent(t *testing.T) {
	tt := test.Start(t).Scenario("allow_trust")
	defer tt.Finish()

	w := mustInitWeb(context.Background(), &history.Q{tt.HorizonSession()}, &core.Q{tt.CoreSession()}, time.Duration(5))

	event, err := w.loadAccountEvent(tt.Ctx, "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU")
	tt.Assert.NoError(err)

	account, ok := event.Data.(*horizon.Account)
	if !ok {
		tt.Assert.FailNow("type assertion failed when getting account info")
	}

	tt.Assert.Equal("8589934593", event.Data.(*horizon.Account).Sequence)
	tt.Assert.NotEqual(0, account.LastModifiedLedger)

	for _, balance := range account.Balances {
		if balance.Type == "native" {
			tt.Assert.Equal(uint32(0), balance.LastModifiedLedger)
		} else {
			tt.Assert.NotEqual(uint32(0), balance.LastModifiedLedger)
		}
	}
}
