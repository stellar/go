package ledgerbackend

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/support/db"
)

// Ledger contains information about a ledger (sequence number and hash)
type Ledger struct {
	Sequence uint32 `db:"sequence"`
	Hash     string `db:"ledger_hash"`
}

// LedgerStore is used to query ledger data from the Horizon DB
type LedgerStore interface {
	// LastLedger returns the highest ledger which is less than `seq` if there exists such a ledger
	LastLedger(seq uint32) (Ledger, bool, error)
}

// EmptyLedgerStore is a ledger store which is empty
type EmptyLedgerStore struct{}

// LastLedger always returns false indicating there is no ledger
func (e EmptyLedgerStore) LastLedger(seq uint32) (Ledger, bool, error) {
	return Ledger{}, false, nil
}

// DBLedgerStore is a ledger store backed by the Horizon database
type DBLedgerStore struct {
	session *db.Session
}

// NewDBLedgerStore constructs a new DBLedgerStore
func NewDBLedgerStore(session *db.Session) LedgerStore {
	return DBLedgerStore{session: session}
}

// LastLedger returns the highest ledger which is less than `seq` if there exists such a ledger
func (l DBLedgerStore) LastLedger(seq uint32) (Ledger, bool, error) {
	sql := sq.Select(
		"hl.sequence",
		"hl.ledger_hash",
	).From("history_ledgers hl").
		Limit(1).
		Where("sequence < ?", seq).
		OrderBy("sequence desc")

	var dest Ledger
	err := l.session.Get(&dest, sql)
	if l.session.NoRows(err) {
		return dest, false, nil
	}
	return dest, true, err
}
