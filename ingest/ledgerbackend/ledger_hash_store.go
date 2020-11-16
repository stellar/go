package ledgerbackend

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/support/db"
)

// TrustedLedgerHashStore is used to query ledger data from a trusted source.
// The store should contain ledgers verified by Stellar-Core, do not use untrusted
// source like history archives.
type TrustedLedgerHashStore interface {
	// GetLedgerHash returns the ledger hash for the given sequence number
	GetLedgerHash(seq uint32) (string, bool, error)
}

// HorizonDBLedgerHashStore is a TrustedLedgerHashStore which uses horizon's db to look up ledger hashes
type HorizonDBLedgerHashStore struct {
	session *db.Session
}

// NewHorizonDBLedgerHashStore constructs a new TrustedLedgerHashStore backed by the horizon db
func NewHorizonDBLedgerHashStore(session *db.Session) TrustedLedgerHashStore {
	return HorizonDBLedgerHashStore{session: session}
}

// GetLedgerHash returns the ledger hash for the given sequence number
func (h HorizonDBLedgerHashStore) GetLedgerHash(seq uint32) (string, bool, error) {
	sql := sq.Select("hl.ledger_hash").From("history_ledgers hl").
		Limit(1).Where("sequence = ?", seq)

	var hash string
	err := h.session.Get(&hash, sql)
	if h.session.NoRows(err) {
		return hash, false, nil
	}
	return hash, true, err
}

// MockLedgerHashStore is a mock implementation of TrustedLedgerHashStore
type MockLedgerHashStore struct {
	mock.Mock
}

// GetLedgerHash returns the ledger hash for the given sequence number
func (m *MockLedgerHashStore) GetLedgerHash(seq uint32) (string, bool, error) {
	args := m.Called(seq)
	return args.Get(0).(string), args.Get(1).(bool), args.Error(2)
}
