package resourceadapter

import (
	"context"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	. "github.com/stellar/go/protocols/resource"
)

func PopulateHistoryAccount(ctx context.Context, dest *HistoryAccount, row history.Account) {
	dest.ID = row.Address
	dest.AccountID = row.Address
}
