package ledgerbackend

import (
	"context"

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
	Close() error
}

// HorizonDBLedgerHashStore is a TrustedLedgerHashStore which uses horizon's db to look up ledger hashes
type HorizonDBLedgerHashStore struct {
	cancel  context.CancelFunc
	ctx     context.Context
	session *db.Session
}

// NewHorizonDBLedgerHashStore constructs a new TrustedLedgerHashStore backed by the horizon db
func NewHorizonDBLedgerHashStore(ctx context.Context, session *db.Session) TrustedLedgerHashStore {
	ctx, cancel := context.WithCancel(ctx)
	return HorizonDBLedgerHashStore{
		cancel:  cancel,
		ctx:     ctx,
		session: session,
	}
}

// GetLedgerHash returns the ledger hash for the given sequence number
func (h HorizonDBLedgerHashStore) GetLedgerHash(seq uint32) (string, bool, error) {
	sql := sq.Select("hl.ledger_hash").From("history_ledgers hl").
		Limit(1).Where("sequence = ?", seq)

	var hash string
	err := h.session.Get(h.ctx, &hash, sql)
	if h.session.NoRows(err) {
		return hash, false, nil
	}
	return hash, true, err
}

func (h HorizonDBLedgerHashStore) Close() error {
	h.cancel()
	return h.session.Close()
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

func (m *MockLedgerHashStore) Close() error {
	args := m.Called()
	return args.Error(0)
}
