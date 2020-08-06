package history

import (
	"github.com/guregu/null"
	"github.com/stellar/go/xdr"
)

// ClaimableBalance is a row of data from the `claimable_balances` table.
type ClaimableBalance struct {
	ID                 string                 `db:"id"`
	BalanceID          xdr.ClaimableBalanceId `db:"balance_id"`
	Asset              xdr.Asset              `db:"asset"`
	Amount             xdr.Int64              `db:"amount"`
	Sponsor            null.String            `db:"sponsor"`
	LastModifiedLedger uint32                 `db:"last_modified_ledger"`
}

// Claimant is a row of data from the `claimable_balances_claimants` table
type Claimant struct {
	ID          string             `db:"id"`
	Destination string             `db:"destination"`
	Predicate   xdr.ClaimPredicate `db:"predicate"`
}

type ClaimableBalancesBatchInsertBuilder interface {
	Add(entry *xdr.LedgerEntry) error
	Exec() error
}

// QClaimableBalances defines account related queries.
type QClaimableBalances interface {
	NewClaimableBalancesBatchInsertBuilder(maxBatchSize int) ClaimableBalancesBatchInsertBuilder
	UpdateClaimableBalance(entry *xdr.LedgerEntry) (int64, error)
	RemoveClaimableBalance(key xdr.LedgerKeyClaimableBalance) (int64, error)
}
