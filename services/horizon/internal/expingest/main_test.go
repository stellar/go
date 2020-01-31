package expingest

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	issuer   = xdr.MustAddress("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")
	usdAsset = xdr.Asset{
		Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
		AlphaNum4: &xdr.AssetAlphaNum4{
			AssetCode: [4]byte{'u', 's', 'd', 0},
			Issuer:    issuer,
		},
	}

	nativeAsset = xdr.Asset{
		Type: xdr.AssetTypeAssetTypeNative,
	}

	eurAsset = xdr.Asset{
		Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
		AlphaNum4: &xdr.AssetAlphaNum4{
			AssetCode: [4]byte{'e', 'u', 'r', 0},
			Issuer:    issuer,
		},
	}
	eurOffer = xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(4),
		Buying:   eurAsset,
		Selling:  nativeAsset,
		Price: xdr.Price{
			N: 1,
			D: 1,
		},
		Flags:  1,
		Amount: xdr.Int64(500),
	}
	twoEurOffer = xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(5),
		Buying:   eurAsset,
		Selling:  nativeAsset,
		Price: xdr.Price{
			N: 2,
			D: 1,
		},
		Flags:  2,
		Amount: xdr.Int64(500),
	}
)

func TestCheckVerifyStateVersion(t *testing.T) {
	assert.Equal(
		t,
		CurrentVersion,
		stateVerifierExpectedIngestionVersion,
		"State verifier is outdated, update it, then update stateVerifierExpectedIngestionVersion value",
	)
}

func TestStateMachineRunReturnsUnexpectedTransaction(t *testing.T) {
	historyQ := &mockDBQ{}
	system := &System{
		historyQ: historyQ,
		ctx:      context.Background(),
	}

	historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()

	assert.PanicsWithValue(t, "unexpected transaction", func() {
		system.Run()
	})
}

func TestStateMachineTransition(t *testing.T) {
	historyQ := &mockDBQ{}
	system := &System{
		historyQ: historyQ,
		ctx:      context.Background(),
	}

	historyQ.On("GetTx").Return(nil).Once()
	historyQ.On("Begin").Return(errors.New("my error")).Once()
	historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()

	assert.PanicsWithValue(t, "unexpected transaction", func() {
		system.Run()
	})
}

func TestContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	historyQ := &mockDBQ{}
	system := &System{
		historyQ: historyQ,
		ctx:      ctx,
		state: state{
			systemState: initState,
		},
	}

	historyQ.On("GetTx").Return(nil).Once()
	historyQ.On("Begin").Return(errors.New("my error")).Once()

	cancel()
	assert.NoError(t, system.run())
}

// TestStateMachineRunReturnsErrorWhenNextStateIsShutdownWithError checks if the
// state that goes to shutdownState and returns an error will make `run` function
// return that error. This is essential because some commands rely on this to return
// non-zero exit code.
func TestStateMachineRunReturnsErrorWhenNextStateIsShutdownWithError(t *testing.T) {
	historyQ := &mockDBQ{}
	graph := &mockOrderBookGraph{}
	system := &System{
		ctx: context.Background(),
		state: state{
			systemState: verifyRangeState,
		},
		historyQ: historyQ,
		graph:    graph,
	}

	historyQ.On("GetTx").Return(nil).Once()

	err := system.run()
	assert.Error(t, err)
	assert.EqualError(t, err, "invalid range: [0, 0]")
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

func (m *mockDBQ) BeginTx(txOpts *sql.TxOptions) error {
	args := m.Called(txOpts)
	return args.Error(0)
}

func (m *mockDBQ) CloneIngestionQ() history.IngestionQ {
	args := m.Called()
	return args.Get(0).(history.IngestionQ)
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

type mockProcessorsRunner struct {
	mock.Mock
}

func (m *mockProcessorsRunner) RunHistoryArchiveIngestion(checkpointLedger uint32) error {
	args := m.Called(checkpointLedger)
	return args.Error(0)
}

func (m *mockProcessorsRunner) RunAllProcessorsOnLedger(sequence uint32) error {
	args := m.Called(sequence)
	return args.Error(0)
}

func (m *mockProcessorsRunner) RunTransactionProcessorsOnLedger(sequence uint32) error {
	args := m.Called(sequence)
	return args.Error(0)
}

func (m *mockProcessorsRunner) RunOrderBookProcessorOnLedger(sequence uint32) error {
	args := m.Called(sequence)
	return args.Error(0)
}

var _ ProcessorRunnerInterface = (*mockProcessorsRunner)(nil)

type mockStellarCoreClient struct {
	mock.Mock
}

func (m *mockStellarCoreClient) SetCursor(ctx context.Context, id string, cursor int32) error {
	args := m.Called(ctx, id, cursor)
	return args.Error(0)
}

var _ stellarCoreClient = (*mockStellarCoreClient)(nil)
