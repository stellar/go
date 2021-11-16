package history

import (
	"context"

	"github.com/stellar/go/support/db"
)

func (i *accountSignersBatchInsertBuilder) Add(ctx context.Context, signer AccountSigner) error {
	return i.builder.Row(ctx, map[string]interface{}{
		"account_id": signer.Account,
		"signer":     signer.Signer,
		"weight":     signer.Weight,
		"sponsor":    signer.Sponsor,
	})
}

func (i *accountSignersBatchInsertBuilder) Exec(ctx context.Context) error {
	ctx = context.WithValue(ctx, &db.RouteContextKey, "accountSignersBatchInsertBuilder")
	return i.builder.Exec(ctx)
}
