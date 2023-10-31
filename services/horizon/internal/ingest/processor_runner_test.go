package ingest

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/guregu/null"
	"github.com/guregu/null/zero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/network"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/processors"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/xdr"
)

func TestProcessorRunnerRunHistoryArchiveIngestionGenesis(t *testing.T) {
	ctx := context.Background()
	maxBatchSize := 100000

	q := &mockDBQ{}

	q.MockQAccounts.On("UpsertAccounts", ctx, []history.AccountEntry{
		{
			LastModifiedLedger: 1,
			AccountID:          "GAAZI4TCR3TY5OJHCTJC2A4QSY6CJWJH5IAJTGKIN2ER7LBNVKOCCWN7",
			Balance:            int64(1000000000000000000),
			SequenceNumber:     0,
			SequenceTime:       zero.IntFrom(0),
			MasterWeight:       1,
		},
	}).Return(nil).Once()

	mockAccountSignersBatchInsertBuilder := &history.MockAccountSignersBatchInsertBuilder{}
	defer mock.AssertExpectationsForObjects(t, mockAccountSignersBatchInsertBuilder)
	mockAccountSignersBatchInsertBuilder.On("Add", ctx, history.AccountSigner{
		Account: "GAAZI4TCR3TY5OJHCTJC2A4QSY6CJWJH5IAJTGKIN2ER7LBNVKOCCWN7",
		Signer:  "GAAZI4TCR3TY5OJHCTJC2A4QSY6CJWJH5IAJTGKIN2ER7LBNVKOCCWN7",
		Weight:  1,
		Sponsor: null.String{},
	}).Return(nil).Once()
	mockAccountSignersBatchInsertBuilder.On("Exec", ctx).Return(nil).Once()
	q.MockQSigners.On("NewAccountSignersBatchInsertBuilder", maxBatchSize).
		Return(mockAccountSignersBatchInsertBuilder).Once()

	q.MockQClaimableBalances.On("NewClaimableBalanceClaimantBatchInsertBuilder", maxBatchSize).
		Return(&history.MockClaimableBalanceClaimantBatchInsertBuilder{}).Once()

	q.MockQAssetStats.On("InsertAssetStats", ctx, []history.ExpAssetStat{}, 100000).
		Return(nil)

	runner := ProcessorRunner{
		ctx: ctx,
		config: Config{
			NetworkPassphrase: network.PublicNetworkPassphrase,
		},
		historyQ: q,
		filters:  &MockFilters{},
	}

	_, err := runner.RunGenesisStateIngestion()
	assert.NoError(t, err)
}

func TestProcessorRunnerRunHistoryArchiveIngestionHistoryArchive(t *testing.T) {
	ctx := context.Background()
	maxBatchSize := 100000

	config := Config{
		NetworkPassphrase: network.PublicNetworkPassphrase,
	}

	q := &mockDBQ{}
	defer mock.AssertExpectationsForObjects(t, q)
	historyAdapter := &mockHistoryArchiveAdapter{}
	defer mock.AssertExpectationsForObjects(t, historyAdapter)

	bucketListHash := xdr.Hash([32]byte{0, 1, 2})
	historyAdapter.On("BucketListHash", uint32(63)).Return(bucketListHash, nil).Once()

	m := &ingest.MockChangeReader{}
	m.On("Read").Return(ingest.GenesisChange(network.PublicNetworkPassphrase), nil).Once()
	m.On("Read").Return(ingest.Change{}, io.EOF).Once()
	m.On("Close").Return(nil).Once()

	historyAdapter.
		On("GetState", ctx, uint32(63)).
		Return(
			m,
			nil,
		).Once()

	q.MockQAccounts.On("UpsertAccounts", ctx, []history.AccountEntry{
		{
			LastModifiedLedger: 1,
			AccountID:          "GAAZI4TCR3TY5OJHCTJC2A4QSY6CJWJH5IAJTGKIN2ER7LBNVKOCCWN7",
			Balance:            int64(1000000000000000000),
			SequenceNumber:     0,
			MasterWeight:       1,
		},
	}).Return(nil).Once()

	mockAccountSignersBatchInsertBuilder := &history.MockAccountSignersBatchInsertBuilder{}
	defer mock.AssertExpectationsForObjects(t, mockAccountSignersBatchInsertBuilder)
	mockAccountSignersBatchInsertBuilder.On("Add", ctx, history.AccountSigner{
		Account: "GAAZI4TCR3TY5OJHCTJC2A4QSY6CJWJH5IAJTGKIN2ER7LBNVKOCCWN7",
		Signer:  "GAAZI4TCR3TY5OJHCTJC2A4QSY6CJWJH5IAJTGKIN2ER7LBNVKOCCWN7",
		Weight:  1,
	}).Return(nil).Once()
	mockAccountSignersBatchInsertBuilder.On("Exec", ctx).Return(nil).Once()
	q.MockQSigners.On("NewAccountSignersBatchInsertBuilder", maxBatchSize).
		Return(mockAccountSignersBatchInsertBuilder).Once()
	q.MockQClaimableBalances.On("NewClaimableBalanceClaimantBatchInsertBuilder", maxBatchSize).
		Return(&history.MockClaimableBalanceClaimantBatchInsertBuilder{}).Once()

	q.MockQAssetStats.On("InsertAssetStats", ctx, []history.ExpAssetStat{}, 100000).
		Return(nil)

	runner := ProcessorRunner{
		ctx:            ctx,
		config:         config,
		historyQ:       q,
		historyAdapter: historyAdapter,
		filters:        &MockFilters{},
	}

	_, err := runner.RunHistoryArchiveIngestion(63, false, MaxSupportedProtocolVersion, bucketListHash)
	assert.NoError(t, err)
}

func TestProcessorRunnerRunHistoryArchiveIngestionProtocolVersionNotSupported(t *testing.T) {
	ctx := context.Background()
	maxBatchSize := 100000

	config := Config{
		NetworkPassphrase: network.PublicNetworkPassphrase,
	}

	q := &mockDBQ{}
	defer mock.AssertExpectationsForObjects(t, q)
	historyAdapter := &mockHistoryArchiveAdapter{}
	defer mock.AssertExpectationsForObjects(t, historyAdapter)

	// Batches

	mockAccountSignersBatchInsertBuilder := &history.MockAccountSignersBatchInsertBuilder{}
	defer mock.AssertExpectationsForObjects(t, mockAccountSignersBatchInsertBuilder)
	q.MockQSigners.On("NewAccountSignersBatchInsertBuilder", maxBatchSize).
		Return(mockAccountSignersBatchInsertBuilder).Once()
	q.MockQClaimableBalances.On("NewClaimableBalanceClaimantBatchInsertBuilder", maxBatchSize).
		Return(&history.MockClaimableBalanceClaimantBatchInsertBuilder{}).Once()

	q.MockQAssetStats.On("InsertAssetStats", ctx, []history.ExpAssetStat{}, 100000).
		Return(nil)

	runner := ProcessorRunner{
		ctx:            ctx,
		config:         config,
		historyQ:       q,
		historyAdapter: historyAdapter,
		filters:        &MockFilters{},
	}

	_, err := runner.RunHistoryArchiveIngestion(100, false, 200, xdr.Hash{})
	assert.EqualError(t, err,
		fmt.Sprintf(
			"Error while checking for supported protocol version: This Horizon version does not support protocol version 200. The latest supported protocol version is %d. Please upgrade to the latest Horizon version.",
			MaxSupportedProtocolVersion,
		),
	)
}

func TestProcessorRunnerBuildChangeProcessor(t *testing.T) {
	ctx := context.Background()
	maxBatchSize := 100000

	q := &mockDBQ{}
	defer mock.AssertExpectationsForObjects(t, q)

	// Twice = checking ledgerSource and historyArchiveSource
	q.MockQSigners.On("NewAccountSignersBatchInsertBuilder", maxBatchSize).
		Return(&history.MockAccountSignersBatchInsertBuilder{}).Twice()
	q.MockQClaimableBalances.On("NewClaimableBalanceClaimantBatchInsertBuilder", maxBatchSize).
		Return(&history.MockClaimableBalanceClaimantBatchInsertBuilder{}).Twice()
	runner := ProcessorRunner{
		ctx:      ctx,
		historyQ: q,
		filters:  &MockFilters{},
	}

	stats := &ingest.StatsChangeProcessor{}
	processor := buildChangeProcessor(runner.historyQ, stats, ledgerSource, 123, "")
	assert.IsType(t, &groupChangeProcessors{}, processor)

	assert.IsType(t, &statsChangeProcessor{}, processor.processors[0])
	assert.IsType(t, &processors.AccountDataProcessor{}, processor.processors[1])
	assert.IsType(t, &processors.AccountsProcessor{}, processor.processors[2])
	assert.IsType(t, &processors.OffersProcessor{}, processor.processors[3])
	assert.IsType(t, &processors.AssetStatsProcessor{}, processor.processors[4])
	assert.True(t, reflect.ValueOf(processor.processors[4]).
		Elem().FieldByName("useLedgerEntryCache").Bool())
	assert.IsType(t, &processors.SignersProcessor{}, processor.processors[5])
	assert.True(t, reflect.ValueOf(processor.processors[5]).
		Elem().FieldByName("useLedgerEntryCache").Bool())
	assert.IsType(t, &processors.TrustLinesProcessor{}, processor.processors[6])

	runner = ProcessorRunner{
		ctx:      ctx,
		historyQ: q,
		filters:  &MockFilters{},
	}

	processor = buildChangeProcessor(runner.historyQ, stats, historyArchiveSource, 456, "")
	assert.IsType(t, &groupChangeProcessors{}, processor)

	assert.IsType(t, &statsChangeProcessor{}, processor.processors[0])
	assert.IsType(t, &processors.AccountDataProcessor{}, processor.processors[1])
	assert.IsType(t, &processors.AccountsProcessor{}, processor.processors[2])
	assert.IsType(t, &processors.OffersProcessor{}, processor.processors[3])
	assert.IsType(t, &processors.AssetStatsProcessor{}, processor.processors[4])
	assert.False(t, reflect.ValueOf(processor.processors[4]).
		Elem().FieldByName("useLedgerEntryCache").Bool())
	assert.IsType(t, &processors.SignersProcessor{}, processor.processors[5])
	assert.False(t, reflect.ValueOf(processor.processors[5]).
		Elem().FieldByName("useLedgerEntryCache").Bool())
	assert.IsType(t, &processors.TrustLinesProcessor{}, processor.processors[6])
}

func TestProcessorRunnerBuildTransactionProcessor(t *testing.T) {
	ctx := context.Background()

	q := &mockDBQ{}
	defer mock.AssertExpectationsForObjects(t, q)

	q.MockQTransactions.On("NewTransactionBatchInsertBuilder").
		Return(&history.MockTransactionsBatchInsertBuilder{})
	q.On("NewTradeBatchInsertBuilder").Return(&history.MockTradeBatchInsertBuilder{})
	q.MockQLedgers.On("NewLedgerBatchInsertBuilder").
		Return(&history.MockLedgersBatchInsertBuilder{})
	q.MockQEffects.On("NewEffectBatchInsertBuilder").
		Return(&history.MockEffectBatchInsertBuilder{})
	q.MockQOperations.On("NewOperationBatchInsertBuilder").
		Return(&history.MockOperationsBatchInsertBuilder{})
	q.On("NewTransactionParticipantsBatchInsertBuilder").
		Return(&history.MockTransactionParticipantsBatchInsertBuilder{})
	q.On("NewOperationParticipantBatchInsertBuilder").
		Return(&history.MockOperationParticipantBatchInsertBuilder{})
	q.MockQHistoryClaimableBalances.On("NewTransactionClaimableBalanceBatchInsertBuilder").
		Return(&history.MockTransactionClaimableBalanceBatchInsertBuilder{})
	q.MockQHistoryClaimableBalances.On("NewOperationClaimableBalanceBatchInsertBuilder").
		Return(&history.MockOperationClaimableBalanceBatchInsertBuilder{})
	q.MockQHistoryLiquidityPools.On("NewTransactionLiquidityPoolBatchInsertBuilder").
		Return(&history.MockTransactionLiquidityPoolBatchInsertBuilder{})
	q.MockQHistoryLiquidityPools.On("NewOperationLiquidityPoolBatchInsertBuilder").
		Return(&history.MockOperationLiquidityPoolBatchInsertBuilder{})

	runner := ProcessorRunner{
		ctx:      ctx,
		config:   Config{},
		historyQ: q,
	}

	stats := &processors.StatsLedgerTransactionProcessor{}
	trades := &processors.TradeProcessor{}

	ledgersProcessor := &processors.LedgersProcessor{}

	processor := runner.buildTransactionProcessor(stats, trades, ledgersProcessor)
	assert.IsType(t, &groupTransactionProcessors{}, processor)
	assert.IsType(t, &statsLedgerTransactionProcessor{}, processor.processors[0])
	assert.IsType(t, &processors.EffectProcessor{}, processor.processors[1])
	assert.IsType(t, &processors.LedgersProcessor{}, processor.processors[2])
	assert.IsType(t, &processors.OperationProcessor{}, processor.processors[3])
	assert.IsType(t, &processors.TradeProcessor{}, processor.processors[4])
	assert.IsType(t, &processors.ParticipantsProcessor{}, processor.processors[5])
	assert.IsType(t, &processors.ClaimableBalancesTransactionProcessor{}, processor.processors[7])
	assert.IsType(t, &processors.LiquidityPoolsTransactionProcessor{}, processor.processors[8])
}

func TestProcessorRunnerWithFilterEnabled(t *testing.T) {
	ctx := context.Background()
	maxBatchSize := 100000

	config := Config{
		NetworkPassphrase:        network.PublicNetworkPassphrase,
		EnableIngestionFiltering: true,
	}

	q := &mockDBQ{}
	mockSession := &db.MockSession{}
	defer mock.AssertExpectationsForObjects(t, q)

	ledger := xdr.LedgerCloseMeta{
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					BucketListHash: xdr.Hash([32]byte{0, 1, 2}),
				},
			},
		},
	}

	// Batches
	mockTransactionsFilteredTmpBatchInsertBuilder := &history.MockTransactionsBatchInsertBuilder{}
	defer mock.AssertExpectationsForObjects(t, mockTransactionsFilteredTmpBatchInsertBuilder)
	mockTransactionsFilteredTmpBatchInsertBuilder.On("Exec", ctx, mockSession).Return(nil).Once()
	q.MockQTransactions.On("NewTransactionFilteredTmpBatchInsertBuilder").
		Return(mockTransactionsFilteredTmpBatchInsertBuilder)

	q.On("DeleteTransactionsFilteredTmpOlderThan", ctx, mock.AnythingOfType("uint64")).
		Return(int64(0), nil)

	defer mock.AssertExpectationsForObjects(t, mockBatchBuilders(q, mockSession, ctx, maxBatchSize)...)

	mockBatchInsertBuilder := &history.MockLedgersBatchInsertBuilder{}
	q.MockQLedgers.On("NewLedgerBatchInsertBuilder").Return(mockBatchInsertBuilder)
	mockBatchInsertBuilder.On(
		"Add",
		ledger.V0.LedgerHeader, 0, 0, 0, 0, CurrentVersion).Return(nil)
	mockBatchInsertBuilder.On(
		"Exec",
		ctx,
		mockSession,
	).Return(nil)

	runner := ProcessorRunner{
		ctx:      ctx,
		config:   config,
		historyQ: q,
		session:  mockSession,
		filters:  &MockFilters{},
	}

	_, err := runner.RunAllProcessorsOnLedger(ledger)
	assert.NoError(t, err)
}

func TestProcessorRunnerRunAllProcessorsOnLedger(t *testing.T) {
	ctx := context.Background()
	maxBatchSize := 100000

	config := Config{
		NetworkPassphrase: network.PublicNetworkPassphrase,
	}

	mockSession := &db.MockSession{}
	q := &mockDBQ{}
	defer mock.AssertExpectationsForObjects(t, q)

	ledger := xdr.LedgerCloseMeta{
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					BucketListHash: xdr.Hash([32]byte{0, 1, 2}),
				},
			},
		},
	}

	// Batches
	defer mock.AssertExpectationsForObjects(t, mockBatchBuilders(q, mockSession, ctx, maxBatchSize)...)

	mockBatchInsertBuilder := &history.MockLedgersBatchInsertBuilder{}
	q.MockQLedgers.On("NewLedgerBatchInsertBuilder").Return(mockBatchInsertBuilder)
	mockBatchInsertBuilder.On(
		"Add",
		ledger.V0.LedgerHeader, 0, 0, 0, 0, CurrentVersion).Return(nil)
	mockBatchInsertBuilder.On(
		"Exec",
		ctx,
		mockSession,
	).Return(nil)

	runner := ProcessorRunner{
		ctx:      ctx,
		config:   config,
		historyQ: q,
		session:  mockSession,
		filters:  &MockFilters{},
	}

	_, err := runner.RunAllProcessorsOnLedger(ledger)
	assert.NoError(t, err)
}

func TestProcessorRunnerRunAllProcessorsOnLedgerProtocolVersionNotSupported(t *testing.T) {
	ctx := context.Background()
	maxBatchSize := 100000

	config := Config{
		NetworkPassphrase: network.PublicNetworkPassphrase,
	}

	q := &mockDBQ{}
	defer mock.AssertExpectationsForObjects(t, q)

	ledger := xdr.LedgerCloseMeta{
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					LedgerVersion: 200,
				},
			},
		},
	}

	// Batches
	mockTransactionsBatchInsertBuilder := &history.MockTransactionsBatchInsertBuilder{}
	q.MockQTransactions.On("NewTransactionBatchInsertBuilder", maxBatchSize).
		Return(mockTransactionsBatchInsertBuilder).Twice()

	mockAccountSignersBatchInsertBuilder := &history.MockAccountSignersBatchInsertBuilder{}
	q.MockQSigners.On("NewAccountSignersBatchInsertBuilder", maxBatchSize).
		Return(mockAccountSignersBatchInsertBuilder).Once()

	mockOperationsBatchInsertBuilder := &history.MockOperationsBatchInsertBuilder{}
	q.MockQOperations.On("NewOperationBatchInsertBuilder").
		Return(mockOperationsBatchInsertBuilder).Twice()

	defer mock.AssertExpectationsForObjects(t, mockTransactionsBatchInsertBuilder,
		mockAccountSignersBatchInsertBuilder,
		mockOperationsBatchInsertBuilder)

	runner := ProcessorRunner{
		ctx:      ctx,
		config:   config,
		historyQ: q,
		filters:  &MockFilters{},
	}

	_, err := runner.RunAllProcessorsOnLedger(ledger)
	assert.EqualError(t, err,
		fmt.Sprintf(
			"Error while checking for supported protocol version: This Horizon version does not support protocol version 200. The latest supported protocol version is %d. Please upgrade to the latest Horizon version.",
			MaxSupportedProtocolVersion,
		),
	)
}

func mockBatchBuilders(q *mockDBQ, mockSession *db.MockSession, ctx context.Context, maxBatchSize int) []interface{} {
	mockTransactionsBatchInsertBuilder := &history.MockTransactionsBatchInsertBuilder{}
	mockTransactionsBatchInsertBuilder.On("Exec", ctx, mockSession).Return(nil).Once()
	q.MockQTransactions.On("NewTransactionBatchInsertBuilder").
		Return(mockTransactionsBatchInsertBuilder)

	mockAccountSignersBatchInsertBuilder := &history.MockAccountSignersBatchInsertBuilder{}
	q.MockQSigners.On("NewAccountSignersBatchInsertBuilder", maxBatchSize).
		Return(mockAccountSignersBatchInsertBuilder).Once()

	mockOperationsBatchInsertBuilder := &history.MockOperationsBatchInsertBuilder{}
	mockOperationsBatchInsertBuilder.On("Exec", ctx, mockSession).Return(nil).Once()
	q.MockQOperations.On("NewOperationBatchInsertBuilder").
		Return(mockOperationsBatchInsertBuilder).Twice()

	mockEffectBatchInsertBuilder := &history.MockEffectBatchInsertBuilder{}
	mockEffectBatchInsertBuilder.On("Exec", ctx, mockSession).Return(nil).Once()
	q.MockQEffects.On("NewEffectBatchInsertBuilder").
		Return(mockEffectBatchInsertBuilder)

	mockTransactionsParticipantsBatchInsertBuilder := &history.MockTransactionParticipantsBatchInsertBuilder{}
	mockTransactionsParticipantsBatchInsertBuilder.On("Exec", ctx, mockSession).Return(nil)
	q.On("NewTransactionParticipantsBatchInsertBuilder").
		Return(mockTransactionsParticipantsBatchInsertBuilder)

	mockOperationParticipantBatchInsertBuilder := &history.MockOperationParticipantBatchInsertBuilder{}
	mockOperationParticipantBatchInsertBuilder.On("Exec", ctx, mockSession).Return(nil)
	q.On("NewOperationParticipantBatchInsertBuilder").
		Return(mockOperationParticipantBatchInsertBuilder)

	mockTransactionClaimableBalanceBatchInsertBuilder := &history.MockTransactionClaimableBalanceBatchInsertBuilder{}
	mockTransactionClaimableBalanceBatchInsertBuilder.On("Exec", ctx, mockSession).Return(nil)
	q.MockQHistoryClaimableBalances.On("NewTransactionClaimableBalanceBatchInsertBuilder").
		Return(mockTransactionClaimableBalanceBatchInsertBuilder)

	mockOperationClaimableBalanceBatchInsertBuilder := &history.MockOperationClaimableBalanceBatchInsertBuilder{}
	mockOperationClaimableBalanceBatchInsertBuilder.On("Exec", ctx, mockSession).Return(nil)
	q.MockQHistoryClaimableBalances.On("NewOperationClaimableBalanceBatchInsertBuilder").
		Return(mockOperationClaimableBalanceBatchInsertBuilder)

	mockTransactionLiquidityPoolBatchInsertBuilder := &history.MockTransactionLiquidityPoolBatchInsertBuilder{}
	mockTransactionLiquidityPoolBatchInsertBuilder.On("Exec", ctx, mockSession).Return(nil)
	q.MockQHistoryLiquidityPools.On("NewTransactionLiquidityPoolBatchInsertBuilder").
		Return(mockTransactionLiquidityPoolBatchInsertBuilder)

	mockOperationLiquidityPoolBatchInsertBuilder := &history.MockOperationLiquidityPoolBatchInsertBuilder{}
	mockOperationLiquidityPoolBatchInsertBuilder.On("Exec", ctx, mockSession).Return(nil)
	q.MockQHistoryLiquidityPools.On("NewOperationLiquidityPoolBatchInsertBuilder").
		Return(mockOperationLiquidityPoolBatchInsertBuilder)

	q.MockQClaimableBalances.On("NewClaimableBalanceClaimantBatchInsertBuilder", maxBatchSize).
		Return(&history.MockClaimableBalanceClaimantBatchInsertBuilder{}).Once()

	q.On("NewTradeBatchInsertBuilder").Return(&history.MockTradeBatchInsertBuilder{})

	return []interface{}{mockAccountSignersBatchInsertBuilder,
		mockOperationsBatchInsertBuilder,
		mockTransactionsBatchInsertBuilder}
}
