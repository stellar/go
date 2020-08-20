package expingest

import (
	"bytes"
	"context"
	"database/sql"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	logpkg "github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
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

func TestNewSystem(t *testing.T) {
	config := Config{
		CoreSession: &db.Session{
			DB:  &sqlx.DB{},
			Ctx: context.Background(),
		},
		HistorySession: &db.Session{
			DB:  &sqlx.DB{},
			Ctx: context.Background(),
		},
		DisableStateVerification: true,
		HistoryArchiveURL:        "https://history.stellar.org/prd/core-live/core_live_001",
	}

	sIface, err := NewSystem(config)
	assert.NoError(t, err)
	system := sIface.(*system)

	assert.Equal(t, config, system.config)
	assert.Equal(t, config.DisableStateVerification, system.disableStateVerification)

	assert.Equal(t, config, system.runner.(*ProcessorRunner).config)
	assert.Equal(t, system.ctx, system.runner.(*ProcessorRunner).ctx)
}

func TestStateMachineRunReturnsUnexpectedTransaction(t *testing.T) {
	historyQ := &mockDBQ{}
	system := &system{
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
	system := &system{
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
	system := &system{
		historyQ: historyQ,
		ctx:      ctx,
	}

	historyQ.On("GetTx").Return(nil).Once()
	historyQ.On("Begin").Return(errors.New("my error")).Once()

	cancel()
	assert.NoError(t, system.runStateMachine(startState{}))
}

// TestStateMachineRunReturnsErrorWhenNextStateIsShutdownWithError checks if the
// state that goes to shutdownState and returns an error will make `run` function
// return that error. This is essential because some commands rely on this to return
// non-zero exit code.
func TestStateMachineRunReturnsErrorWhenNextStateIsShutdownWithError(t *testing.T) {
	historyQ := &mockDBQ{}
	system := &system{
		ctx:      context.Background(),
		historyQ: historyQ,
	}

	historyQ.On("GetTx").Return(nil).Once()

	err := system.runStateMachine(verifyRangeState{})
	assert.Error(t, err)
	assert.EqualError(t, err, "invalid range: [0, 0]")
}

func TestMaybeVerifyStateGetExpStateInvalidDBErrCancelOrContextCanceled(t *testing.T) {
	historyQ := &mockDBQ{}
	system := &system{
		historyQ: historyQ,
		ctx:      context.Background(),
	}

	var out bytes.Buffer
	logger := logpkg.New()
	logger.Logger.Out = &out
	done := logger.StartTest(logpkg.InfoLevel)

	oldLogger := log
	log = logger
	defer func() { log = oldLogger }()

	historyQ.On("GetExpStateInvalid").Return(false, db.ErrCancelled).Once()
	system.maybeVerifyState(0)

	historyQ.On("GetExpStateInvalid").Return(false, context.Canceled).Once()
	system.maybeVerifyState(0)

	logged := done()
	assert.Len(t, logged, 0)
	historyQ.AssertExpectations(t)
}
func TestMaybeVerifyInternalDBErrCancelOrContextCanceled(t *testing.T) {
	historyQ := &mockDBQ{}
	system := &system{
		historyQ: historyQ,
		ctx:      context.Background(),
	}

	var out bytes.Buffer
	logger := logpkg.New()
	logger.Logger.Out = &out
	done := logger.StartTest(logpkg.InfoLevel)

	oldLogger := log
	log = logger
	defer func() { log = oldLogger }()

	historyQ.On("GetExpStateInvalid").Return(false, nil).Twice()
	historyQ.On("Rollback").Return(nil).Twice()
	historyQ.On("CloneIngestionQ").Return(historyQ).Twice()

	historyQ.On("BeginTx", mock.Anything).Return(db.ErrCancelled).Once()
	system.maybeVerifyState(63)
	system.wg.Wait()

	historyQ.On("BeginTx", mock.Anything).Return(context.Canceled).Once()
	system.maybeVerifyState(63)
	system.wg.Wait()

	logged := done()

	// it logs "State verification finished" twice, but no errors
	assert.Len(t, logged, 2)

	historyQ.AssertExpectations(t)
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

func (m *mockDBQ) GetOfferCompactionSequence() (uint32, error) {
	args := m.Called()
	return args.Get(0).(uint32), args.Error(1)
}

func (m *mockDBQ) GetLastLedgerExpIngestNonBlocking() (uint32, error) {
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

func (m *mockDBQ) CreateAssets(assets []xdr.Asset, batchSize int) (map[string]history.Asset, error) {
	args := m.Called(assets)
	return args.Get(0).(map[string]history.Asset), args.Error(1)
}

type mockLedgerBackend struct {
	mock.Mock
}

func (m *mockLedgerBackend) GetLatestLedgerSequence() (sequence uint32, err error) {
	args := m.Called()
	return args.Get(0).(uint32), args.Error(1)
}

func (m *mockLedgerBackend) GetLedger(sequence uint32) (bool, xdr.LedgerCloseMeta, error) {
	args := m.Called(sequence)
	return args.Get(0).(bool), args.Get(1).(xdr.LedgerCloseMeta), args.Error(2)
}

func (m *mockLedgerBackend) PrepareRange(ledgerRange ledgerbackend.Range) error {
	args := m.Called(ledgerRange)
	return args.Error(0)
}

func (m *mockLedgerBackend) IsPrepared(ledgerRange ledgerbackend.Range) (bool, error) {
	args := m.Called(ledgerRange)
	return args.Get(0).(bool), args.Error(1)
}

func (m *mockLedgerBackend) Close() error {
	args := m.Called()
	return args.Error(0)
}

type mockProcessorsRunner struct {
	mock.Mock
}

func (m *mockProcessorsRunner) SetLedgerBackend(ledgerBackend ledgerbackend.LedgerBackend) {
	m.Called(ledgerBackend)
}

func (m *mockProcessorsRunner) SetHistoryAdapter(historyAdapter adapters.HistoryArchiveAdapterInterface) {
	m.Called(historyAdapter)
}

func (m *mockProcessorsRunner) EnableMemoryStatsLogging() {
	m.Called()
}

func (m *mockProcessorsRunner) DisableMemoryStatsLogging() {
	m.Called()
}

func (m *mockProcessorsRunner) RunHistoryArchiveIngestion(checkpointLedger uint32) (io.StatsChangeProcessorResults, error) {
	args := m.Called(checkpointLedger)
	return args.Get(0).(io.StatsChangeProcessorResults), args.Error(1)
}

func (m *mockProcessorsRunner) RunAllProcessorsOnLedger(sequence uint32) (io.StatsChangeProcessorResults, io.StatsLedgerTransactionProcessorResults, error) {
	args := m.Called(sequence)
	return args.Get(0).(io.StatsChangeProcessorResults),
		args.Get(1).(io.StatsLedgerTransactionProcessorResults),
		args.Error(2)
}

func (m *mockProcessorsRunner) RunTransactionProcessorsOnLedger(sequence uint32) (io.StatsLedgerTransactionProcessorResults, error) {
	args := m.Called(sequence)
	return args.Get(0).(io.StatsLedgerTransactionProcessorResults), args.Error(1)
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

type mockSystem struct {
	mock.Mock
}

func (m *mockSystem) Run() {
	m.Called()
}

func (m *mockSystem) Metrics() Metrics {
	args := m.Called()
	return args.Get(0).(Metrics)
}

func (m *mockSystem) StressTest(numTransactions, changesPerTransaction int) error {
	args := m.Called(numTransactions, changesPerTransaction)
	return args.Error(0)
}

func (m *mockSystem) VerifyRange(fromLedger, toLedger uint32, verifyState bool) error {
	args := m.Called(fromLedger, toLedger, verifyState)
	return args.Error(0)
}

func (m *mockSystem) ReingestRange(fromLedger, toLedger uint32, force bool) error {
	args := m.Called(fromLedger, toLedger, force)
	return args.Error(0)
}

func (m *mockSystem) Shutdown() {
	m.Called()
}

var _ System = (*mockSystem)(nil)
