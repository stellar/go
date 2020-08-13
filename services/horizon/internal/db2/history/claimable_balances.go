package history

import (
	"crypto/sha256"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"

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
	Claimants          Claimants              `db:"claimants"`
	Asset              xdr.Asset              `db:"asset"`
	Amount             xdr.Int64              `db:"amount"`
	Sponsor            null.String            `db:"sponsor"`
	LastModifiedLedger uint32                 `db:"last_modified_ledger"`
}

type Claimants []Claimant

func (c Claimants) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *Claimants) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &c)
}

type Claimant struct {
	Destination string             `json:"destination"`
	Predicate   xdr.ClaimPredicate `json:"predicate"`
}

// internal representation of Claimant as a JSON. Don't use this directly, use
// Claimant instead.
var dbClaim struct {
	Destination string `json:"destination"`
	Predicate   string `json:"predicate"`
}

func (c *Claimant) MarshalJSON() ([]byte, error) {
	dbClaim.Destination = c.Destination
	predicate, err := xdr.MarshalBase64(c.Predicate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode predicate to base64")
	}
	dbClaim.Predicate = predicate

	return json.Marshal(dbClaim)
}

func (c *Claimant) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, &dbClaim)
	if err != nil {
		return errors.Wrap(err, "failed decoding claimant")
	}

	c.Destination = dbClaim.Destination
	err = xdr.SafeUnmarshalBase64(dbClaim.Predicate, &c.Predicate)
	if err != nil {
		return errors.Wrap(err, "failed decoding xdr.ClaimPredicate")
	}

	return nil
}

type ClaimableBalancesBatchInsertBuilder interface {
	Add(entry *xdr.LedgerEntry) error
	Exec() error
}

// QClaimableBalances defines account related queries.
type QClaimableBalances interface {
	NewClaimableBalancesBatchInsertBuilder(maxBatchSize int) ClaimableBalancesBatchInsertBuilder
	UpdateClaimableBalance(entry xdr.LedgerEntry) (int64, error)
	RemoveClaimableBalance(cBalance xdr.ClaimableBalanceEntry) (int64, error)
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

// UpdateClaimableBalance updates a row in the claimable_balances table.
// The only updatable value on claimable_balances is sponsor
// Returns number of rows affected and error.
func (q *Q) UpdateClaimableBalance(entry xdr.LedgerEntry) (int64, error) {
	cBalance := entry.Data.MustClaimableBalance()
	id, err := balanceIDToHex(cBalance.BalanceId)
	if err != nil {
		return 0, errors.Wrap(err, "encoding balanceID")
	}
	cBalanceMap := map[string]interface{}{
		"last_modified_ledger": entry.LastModifiedLedgerSeq,
		"sponsor":              ledgerEntrySponsorToNullString(entry),
	}

	sql := sq.Update("claimable_balances").SetMap(cBalanceMap).Where("id = ?", id)
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// RemoveClaimableBalance deletes a row in the claimable_balances table.
// Returns number of rows affected and error.
func (q *Q) RemoveClaimableBalance(cBalance xdr.ClaimableBalanceEntry) (int64, error) {
	id, err := balanceIDToHex(cBalance.BalanceId)
	if err != nil {
		return 0, errors.Wrap(err, "encoding balanceID")
	}
	sql := sq.Delete("claimable_balances").
		Where(sq.Eq{"id": id})
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// FindClaimableBalancesByDestination finds all claimable balances where accountID is one of the claimants
func (q *Q) FindClaimableBalancesByDestination(accountID xdr.AccountId) ([]ClaimableBalance, error) {
	sql := selectClaimableBalances.
		Where("claimants @> '[{\"destination\": \"" + accountID.Address() + "\"}]'")

	var results []ClaimableBalance
	if err := q.Select(&results, sql); err != nil {
		return nil, errors.Wrap(err, "could not run select query")
	}

	return results, nil
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

func buildClaimants(claimants []xdr.Claimant) Claimants {
	hClaimants := Claimants{}
	for _, c := range claimants {
		xc := c.MustV0()
		hClaimants = append(hClaimants, Claimant{
			Destination: xc.Destination.Address(),
			Predicate:   xc.Predicate,
		})
	}

	return hClaimants
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
		Claimants:          buildClaimants(cBalance.Claimants),
		Asset:              cBalance.Asset,
		Amount:             cBalance.Amount,
		Sponsor:            ledgerEntrySponsorToNullString(*entry),
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
		"cb.claimants, " +
		"cb.asset, " +
		"cb.amount, " +
		"cb.sponsor, " +
		"cb.last_modified_ledger").
	From("claimable_balances cb")
