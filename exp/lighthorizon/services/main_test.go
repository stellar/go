package services

import (
	"context"
	"io"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/exp/lighthorizon/ingester"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	passphrase     = "White New England clam chowder"
	accountId      = "GDCXSQPVE45DVGT2ZRFFIIHSJ2EJED65W6AELGWIDRMPMWNXCEBJ4FKX"
	startLedgerSeq = 1586112
)

func TestItGetsTransactionsByAccount(t *testing.T) {
	ctx := context.Background()

	// this is in the checkpoint range prior to the first active checkpoint
	ledgerSeq := checkpointMgr.PrevCheckpoint(uint32(startLedgerSeq))
	cursor := toid.New(int32(ledgerSeq), 1, 1).ToInt64()

	t.Run("first", func(tt *testing.T) {
		txService := newTransactionService(ctx)

		txs, err := txService.GetTransactionsByAccount(ctx, cursor, 1, accountId)
		require.NoError(tt, err)
		require.Len(tt, txs, 1)
		require.Equal(tt, txs[0].LedgerHeader.LedgerSeq, xdr.Uint32(1586113))
		require.EqualValues(tt, txs[0].TxIndex, 2)
	})

	t.Run("without cursor", func(tt *testing.T) {
		txService := newTransactionService(ctx)

		txs, err := txService.GetTransactionsByAccount(ctx, 0, 1, accountId)
		require.NoError(tt, err)
		require.Len(tt, txs, 1)
		require.Equal(tt, txs[0].LedgerHeader.LedgerSeq, xdr.Uint32(1586113))
		require.EqualValues(tt, txs[0].TxIndex, 2)
	})

	t.Run("with limit", func(tt *testing.T) {
		txService := newTransactionService(ctx)

		txs, err := txService.GetTransactionsByAccount(ctx, cursor, 5, accountId)
		require.NoError(tt, err)
		require.Len(tt, txs, 2)
		require.Equal(tt, txs[0].LedgerHeader.LedgerSeq, xdr.Uint32(1586113))
		require.EqualValues(tt, txs[0].TxIndex, 2)
		require.Equal(tt, txs[1].LedgerHeader.LedgerSeq, xdr.Uint32(1586114))
		require.EqualValues(tt, txs[1].TxIndex, 1)
	})
}

func TestItGetsOperationsByAccount(t *testing.T) {
	ctx := context.Background()

	// this is in the checkpoint range prior to the first active checkpoint
	ledgerSeq := checkpointMgr.PrevCheckpoint(uint32(startLedgerSeq))
	cursor := toid.New(int32(ledgerSeq), 1, 1).ToInt64()

	t.Run("first", func(tt *testing.T) {
		opsService := newOperationService(ctx)

		// this should start at next checkpoint
		ops, err := opsService.GetOperationsByAccount(ctx, cursor, 1, accountId)
		require.NoError(tt, err)
		require.Len(tt, ops, 1)
		require.Equal(tt, ops[0].LedgerHeader.LedgerSeq, xdr.Uint32(1586113))
		require.Equal(tt, ops[0].TxIndex, int32(2))

	})

	t.Run("with limit", func(tt *testing.T) {
		opsService := newOperationService(ctx)

		// this should start at next checkpoint
		ops, err := opsService.GetOperationsByAccount(ctx, cursor, 5, accountId)
		require.NoError(tt, err)
		require.Len(tt, ops, 2)
		require.Equal(tt, ops[0].LedgerHeader.LedgerSeq, xdr.Uint32(1586113))
		require.Equal(tt, ops[0].TxIndex, int32(2))
		require.Equal(tt, ops[1].LedgerHeader.LedgerSeq, xdr.Uint32(1586114))
		require.Equal(tt, ops[1].TxIndex, int32(1))
	})
}

func mockArchiveAndIndex(ctx context.Context) (ingester.Ingester, index.Store) {
	mockArchive := &ingester.MockIngester{}
	mockReaderLedger1 := &ingester.MockLedgerTransactionReader{}
	mockReaderLedger2 := &ingester.MockLedgerTransactionReader{}
	mockReaderLedger3 := &ingester.MockLedgerTransactionReader{}
	mockReaderLedgerTheRest := &ingester.MockLedgerTransactionReader{}

	expectedLedger1 := testLedger(startLedgerSeq)
	expectedLedger2 := testLedger(startLedgerSeq + 1)
	expectedLedger3 := testLedger(startLedgerSeq + 2)

	// throw an irrelevant account in there to make sure it's filtered
	source := xdr.MustAddress("GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU")
	source2 := xdr.MustAddress(accountId)

	// assert results iterate sequentially across ops-tx-ledgers
	expectedLedger1Tx1 := testLedgerTx(source, 1, 34, 35)
	expectedLedger1Tx2 := testLedgerTx(source, 2, 34)
	expectedLedger2Tx1 := testLedgerTx(source, 1, 34)
	expectedLedger2Tx2 := testLedgerTx(source2, 2, 34)
	expectedLedger3Tx1 := testLedgerTx(source2, 1, 34)
	expectedLedger3Tx2 := testLedgerTx(source, 2, 34)

	mockReaderLedger1.
		On("Read").Return(expectedLedger1Tx1, nil).Once().
		On("Read").Return(expectedLedger1Tx2, nil).Once().
		On("Read").Return(ingester.LedgerTransaction{}, io.EOF).Once()

	mockReaderLedger2.
		On("Read").Return(expectedLedger2Tx1, nil).Once().
		On("Read").Return(expectedLedger2Tx2, nil).Once().
		On("Read").Return(ingester.LedgerTransaction{}, io.EOF).Once()

	mockReaderLedger3.
		On("Read").Return(expectedLedger3Tx1, nil).Once().
		On("Read").Return(expectedLedger3Tx2, nil).Once().
		On("Read").Return(ingester.LedgerTransaction{}, io.EOF).Once()

	mockReaderLedgerTheRest.
		On("Read").Return(ingester.LedgerTransaction{}, io.EOF)

	mockArchive.
		On("GetLedger", mock.Anything, uint32(1586112)).Return(expectedLedger1, nil).
		On("GetLedger", mock.Anything, uint32(1586113)).Return(expectedLedger2, nil).
		On("GetLedger", mock.Anything, uint32(1586114)).Return(expectedLedger3, nil).
		On("GetLedger", mock.Anything, mock.AnythingOfType("uint32")).
		Return(xdr.SerializedLedgerCloseMeta{}, nil)

	mockArchive.
		On("NewLedgerTransactionReader", expectedLedger1).Return(mockReaderLedger1, nil).Once().
		On("NewLedgerTransactionReader", expectedLedger2).Return(mockReaderLedger2, nil).Once().
		On("NewLedgerTransactionReader", expectedLedger3).Return(mockReaderLedger3, nil).Once().
		On("NewLedgerTransactionReader", mock.AnythingOfType("xdr.SerializedLedgerCloseMeta")).
		Return(mockReaderLedgerTheRest, nil).
		On("PrepareRange", mock.Anything, mock.Anything).Return(nil)

	// should be 24784
	activeChk := uint32(index.GetCheckpointNumber(uint32(startLedgerSeq)))
	mockStore := &index.MockStore{}
	mockStore.
		On("NextActive", accountId, mock.Anything, uint32(0)).Return(activeChk, nil).     // start
		On("NextActive", accountId, mock.Anything, activeChk-1).Return(activeChk, nil).   // prev
		On("NextActive", accountId, mock.Anything, activeChk).Return(activeChk, nil).     // curr
		On("NextActive", accountId, mock.Anything, activeChk+1).Return(uint32(0), io.EOF) // next

	return mockArchive, mockStore
}

func testLedger(seq int) xdr.SerializedLedgerCloseMeta {
	return xdr.SerializedLedgerCloseMeta{
		V: 0,
		V0: &xdr.LedgerCloseMeta{
			V0: &xdr.LedgerCloseMetaV0{
				LedgerHeader: xdr.LedgerHeaderHistoryEntry{
					Header: xdr.LedgerHeader{
						LedgerSeq: xdr.Uint32(seq),
					},
				},
			},
		},
	}
}

func testLedgerTx(source xdr.AccountId, txIndex uint32, bumpTos ...int) ingester.LedgerTransaction {
	code := xdr.TransactionResultCodeTxSuccess

	operations := []xdr.Operation{}
	for _, bumpTo := range bumpTos {
		operations = append(operations, xdr.Operation{
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeBumpSequence,
				BumpSequenceOp: &xdr.BumpSequenceOp{
					BumpTo: xdr.SequenceNumber(bumpTo),
				},
			},
		})
	}

	return ingester.LedgerTransaction{
		LedgerTransaction: &ingest.LedgerTransaction{
			Result: xdr.TransactionResultPair{
				TransactionHash: xdr.Hash{},
				Result: xdr.TransactionResult{
					Result: xdr.TransactionResultResult{
						Code:            code,
						InnerResultPair: &xdr.InnerTransactionResultPair{},
						Results:         &[]xdr.OperationResult{},
					},
				},
			},
			Envelope: xdr.TransactionEnvelope{
				Type: xdr.EnvelopeTypeEnvelopeTypeTx,
				V1: &xdr.TransactionV1Envelope{
					Tx: xdr.Transaction{
						SourceAccount: source.ToMuxedAccount(),
						Operations:    operations,
					},
				},
			},
			UnsafeMeta: xdr.TransactionMeta{
				V: 2,
				V2: &xdr.TransactionMetaV2{
					Operations: make([]xdr.OperationMeta, len(bumpTos)),
				},
			},
			Index: txIndex,
		},
	}
}

func newTransactionService(ctx context.Context) TransactionService {
	ingest, store := mockArchiveAndIndex(ctx)
	return &TransactionRepository{
		Config: Config{
			Ingester:   ingest,
			IndexStore: store,
			Passphrase: passphrase,
			Metrics:    NewMetrics(prometheus.NewRegistry()),
		},
	}
}

func newOperationService(ctx context.Context) OperationService {
	ingest, store := mockArchiveAndIndex(ctx)
	return &OperationRepository{
		Config: Config{
			Ingester:   ingest,
			IndexStore: store,
			Passphrase: passphrase,
			Metrics:    NewMetrics(prometheus.NewRegistry()),
		},
	}
}
