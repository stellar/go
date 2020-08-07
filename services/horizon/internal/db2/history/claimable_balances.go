package history

import (
	"crypto/sha256"
	"encoding/hex"

	sq "github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
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

// NewClaimableBalancesBatchInsertBuilder constructs a new ClaimableBalancesBatchInsertBuilder instance
func (q *Q) NewClaimableBalancesBatchInsertBuilder(maxBatchSize int) ClaimableBalancesBatchInsertBuilder {
	return &claimableBalancesBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("claimable_balances"),
			MaxBatchSize: maxBatchSize,
		},
	}
}

type claimableBalancesBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

func balanceIDToHex(balanceID xdr.ClaimableBalanceId) (string, error) {
	b, err := balanceID.MarshalBinary()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func (i *claimableBalancesBatchInsertBuilder) Add(entry *xdr.LedgerEntry) error {
	cBalance := entry.Data.MustClaimableBalance()
	id, err := balanceIDToHex(cBalance.BalanceId)
	if err != nil {
		return errors.Wrap(err, "encoding balanceID")
	}
	row := ClaimableBalance{
		ID:                 id,
		BalanceID:          cBalance.BalanceId,
		Asset:              cBalance.Asset,
		Amount:             cBalance.Amount,
		Sponsor:            null.StringFromPtr(nil), // TDB - we can add this later since there might be code from Bartek's PR which we can use to pull the sponsor,
		LastModifiedLedger: uint32(entry.LastModifiedLedgerSeq),
	}
	return i.builder.RowStruct(row)
}

func (i *claimableBalancesBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}

var selectClaimableBalances = sq.Select(
	"cb.id, " +
		"cb.balance_id, " +
		"cb.asset, " +
		"cb.amount, " +
		"cb.sponsor, " +
		"cb.last_modified_ledger").
	From("claimable_balances cb")
