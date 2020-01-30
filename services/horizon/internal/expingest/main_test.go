package expingest

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCheckVerifyStateVersion(t *testing.T) {
	assert.Equal(
		t,
		CurrentVersion,
		stateVerifierExpectedIngestionVersion,
		"State verifier is outdated, update it, then update stateVerifierExpectedIngestionVersion value",
	)
}

type mockDBQ struct {
	mock.Mock

	history.MockQAccounts
	history.MockQAssetStats
	history.MockQData
	history.MockQEffects
	history.MockQLedgers
	history.MockQOffers
	history.MockQOperations
	history.MockQSigners
	history.MockQTransactions
	history.MockQTrustLines
}

func (m *mockDBQ) Begin() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockDBQ) Clone() *db.Session {
	args := m.Called()
	return args.Get(0).(*db.Session)
}

func (m *mockDBQ) Commit() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockDBQ) Rollback() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockDBQ) GetTx() *sqlx.Tx {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*sqlx.Tx)
}

func (m *mockDBQ) GetLastLedgerExpIngest() (uint32, error) {
	args := m.Called()
	return args.Get(0).(uint32), args.Error(1)
}

func (m *mockDBQ) GetExpIngestVersion() (int, error) {
	args := m.Called()
	return args.Get(0).(int), args.Error(1)
}

func (m *mockDBQ) UpdateLastLedgerExpIngest(sequence uint32) error {
	args := m.Called(sequence)
	return args.Error(0)
}

func (m *mockDBQ) UpdateExpStateInvalid(invalid bool) error {
	args := m.Called(invalid)
	return args.Error(0)
}

func (m *mockDBQ) UpdateExpIngestVersion(version int) error {
	args := m.Called(version)
	return args.Error(0)
}

func (m *mockDBQ) GetExpStateInvalid() (bool, error) {
	args := m.Called()
	return args.Get(0).(bool), args.Error(1)
}

func (m *mockDBQ) GetAllOffers() ([]history.Offer, error) {
	args := m.Called()
	return args.Get(0).([]history.Offer), args.Error(1)
}

func (m *mockDBQ) GetLatestLedger() (uint32, error) {
	args := m.Called()
	return args.Get(0).(uint32), args.Error(1)
}

func (m *mockDBQ) TruncateExpingestStateTables() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockDBQ) DeleteRangeAll(start, end int64) error {
	args := m.Called(start, end)
	return args.Error(0)
}

// Methods from interfaces duplicating methods:

func (m *mockDBQ) NewTransactionParticipantsBatchInsertBuilder(maxBatchSize int) history.TransactionParticipantsBatchInsertBuilder {
	args := m.Called(maxBatchSize)
	return args.Get(0).(history.TransactionParticipantsBatchInsertBuilder)
}

func (m *mockDBQ) NewOperationParticipantBatchInsertBuilder(maxBatchSize int) history.OperationParticipantBatchInsertBuilder {
	args := m.Called(maxBatchSize)
	return args.Get(0).(history.TransactionParticipantsBatchInsertBuilder)
}

func (m *mockDBQ) NewTradeBatchInsertBuilder(maxBatchSize int) history.TradeBatchInsertBuilder {
	args := m.Called(maxBatchSize)
	return args.Get(0).(history.TradeBatchInsertBuilder)
}

func (m *mockDBQ) CreateAssets(assets []xdr.Asset) (map[string]history.Asset, error) {
	args := m.Called(assets)
	return args.Get(0).(map[string]history.Asset), args.Error(1)
}
