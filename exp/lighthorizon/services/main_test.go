package services

import (
	"context"
	"io"
	"testing"

	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	accountId      = "GDCXSQPVE45DVGT2ZRFFIIHSJ2EJED65W6AELGWIDRMPMWNXCEBJ4FKX"
	checkpointFreq = int32(historyarchive.DefaultCheckpointFrequency)
)

func TestItGetsTransactionsByAccount(tt *testing.T) {
	ctx := context.Background()
	txService := newTransactionService(ctx)

	// l=1586045, t=1, o=1
	// cursor = 6812011404988417, checkpoint=24781
	ledgerSeq := uint32(1586045)
	cursor := toid.New(int32(ledgerSeq), 1, 1).ToInt64()
	require.Equal(tt, 24782, int(index.GetCheckpointNumber(ledgerSeq))) // sanity
	require.Equal(tt, int64(6812011404988417), cursor)                  // sanity

	// this should start at next checkpoint
	txs, err := txService.GetTransactionsByAccount(ctx, cursor, 1, accountId)
	require.NoError(tt, err)
	require.Len(tt, txs, 1)
	require.Equal(tt, txs[0].LedgerHeader.LedgerSeq, xdr.Uint32(1586113))
	require.Equal(tt, txs[0].TxIndex, int32(2))
}

func TestItGetsTransactionsByAccountAndPageLimit(tt *testing.T) {
	ctx := context.Background()
	txService := newTransactionService(ctx)

	checkpoint := int32(24781)
	cursor := toid.New(checkpoint*checkpointFreq, 1, 1).ToInt64()
	require.Equal(tt, int64(6812011404988417), cursor)

	// this should start at next checkpoint
	txs, err := txService.GetTransactionsByAccount(ctx, cursor, 5, accountId)
	require.NoError(tt, err)
	require.Len(tt, txs, 2)
	require.Equal(tt, txs[0].LedgerHeader.LedgerSeq, xdr.Uint32(1586113))
	require.Equal(tt, txs[0].TxIndex, int32(2))
	require.Equal(tt, txs[1].LedgerHeader.LedgerSeq, xdr.Uint32(1586114))
	require.Equal(tt, txs[1].TxIndex, int32(1))
}

func TestItGetsOperationsByAccount(tt *testing.T) {
	ctx := context.Background()
	opsService := newOperationService(ctx)

	// l=1586045, t=1, o=1
	// cursor = 6812011404988417, checkpoint=24781
	cursor := int64(6812011404988417)

	// this should start at next checkpoint
	ops, err := opsService.GetOperationsByAccount(ctx, cursor, 1, accountId)
	require.NoError(tt, err)
	require.Len(tt, ops, 1)
	require.Equal(tt, ops[0].LedgerHeader.LedgerSeq, xdr.Uint32(1586113))
	require.Equal(tt, ops[0].TxIndex, int32(2))
}

func TestItGetsOperationsByAccountAndPageLimit(tt *testing.T) {
	ctx := context.Background()
	opsService := newOperationService(ctx)

	// cursor = 6812011404988417, checkpoint=24781
	cursor := toid.New(1586045, 1, 1).ToInt64()
	require.Equal(tt, int64(6812011404988417), cursor)

	// this should start at next checkpoint
	ops, err := opsService.GetOperationsByAccount(ctx, cursor, 5, accountId)
	require.NoError(tt, err)
	require.Len(tt, ops, 2)
	require.Equal(tt, ops[0].LedgerHeader.LedgerSeq, xdr.Uint32(1586113))
	require.Equal(tt, ops[0].TxIndex, int32(2))
	require.Equal(tt, ops[1].LedgerHeader.LedgerSeq, xdr.Uint32(1586114))
	require.Equal(tt, ops[1].TxIndex, int32(1))
}

func mockArchiveAndIndex(ctx context.Context, passphrase string) (archive.Archive, index.Store) {
	mockArchive := &archive.MockArchive{}
	mockReaderLedger1 := &archive.MockLedgerTransactionReader{}
	mockReaderLedger2 := &archive.MockLedgerTransactionReader{}
	mockReaderLedger3 := &archive.MockLedgerTransactionReader{}
	mockReaderLedgerTheRest := &archive.MockLedgerTransactionReader{}

	ledgerSeq := 1586112
	expectedLedger1 := testLedger(ledgerSeq)
	expectedLedger2 := testLedger(ledgerSeq + 1)
	expectedLedger3 := testLedger(ledgerSeq + 2)

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

	mockArchive.
		On("GetLedger", ctx, uint32(1586112)).Return(expectedLedger1, nil).
		On("GetLedger", ctx, uint32(1586113)).Return(expectedLedger2, nil).
		On("GetLedger", ctx, uint32(1586114)).Return(expectedLedger3, nil).
		On("GetLedger", ctx, mock.Anything).Return(xdr.LedgerCloseMeta{}, nil)

	mockArchive.
		On("NewLedgerTransactionReaderFromLedgerCloseMeta", passphrase, expectedLedger1).Return(mockReaderLedger1, nil).
		On("NewLedgerTransactionReaderFromLedgerCloseMeta", passphrase, expectedLedger2).Return(mockReaderLedger2, nil).
		On("NewLedgerTransactionReaderFromLedgerCloseMeta", passphrase, expectedLedger3).Return(mockReaderLedger3, nil).
		On("NewLedgerTransactionReaderFromLedgerCloseMeta", passphrase, mock.Anything).Return(mockReaderLedgerTheRest, nil)

	partialParticipants := make(map[string]struct{})
	partialParticipants[source.Address()] = struct{}{}

	allParticipants := make(map[string]struct{})
	allParticipants[source.Address()] = struct{}{}
	allParticipants[source2.Address()] = struct{}{}

	mockArchive.
		On("GetTransactionParticipants", expectedLedger1Tx1).Return(partialParticipants, nil).
		On("GetTransactionParticipants", expectedLedger1Tx2).Return(partialParticipants, nil).
		On("GetTransactionParticipants", expectedLedger2Tx1).Return(partialParticipants, nil).
		On("GetTransactionParticipants", expectedLedger2Tx2).Return(allParticipants, nil).
		On("GetTransactionParticipants", expectedLedger3Tx1).Return(allParticipants, nil).
		On("GetTransactionParticipants", expectedLedger3Tx2).Return(partialParticipants, nil)

	mockArchive.
		On("GetOperationParticipants", expectedLedger1Tx1, mock.Anything, int(0)).Return(partialParticipants, nil).
		On("GetOperationParticipants", expectedLedger1Tx1, mock.Anything, int(1)).Return(partialParticipants, nil).
		On("GetOperationParticipants", expectedLedger1Tx2, mock.Anything, int(0)).Return(partialParticipants, nil).
		On("GetOperationParticipants", expectedLedger2Tx1, mock.Anything, int(0)).Return(partialParticipants, nil).
		On("GetOperationParticipants", expectedLedger2Tx2, mock.Anything, int(0)).Return(allParticipants, nil).
		On("GetOperationParticipants", expectedLedger3Tx1, mock.Anything, int(0)).Return(allParticipants, nil).
		On("GetOperationParticipants", expectedLedger3Tx2, mock.Anything, int(0)).Return(partialParticipants, nil)

	mockReaderLedger1.
		On("Read").Return(expectedLedger1Tx1, nil).Once().
		On("Read").Return(expectedLedger1Tx2, nil).Once().
		On("Read").Return(archive.LedgerTransaction{}, io.EOF).Once()

	mockReaderLedger2.
		On("Read").Return(expectedLedger2Tx1, nil).Once().
		On("Read").Return(expectedLedger2Tx2, nil).Once().
		On("Read").Return(archive.LedgerTransaction{}, io.EOF).Once()

	mockReaderLedger3.
		On("Read").Return(expectedLedger3Tx1, nil).Once().
		On("Read").Return(expectedLedger3Tx2, nil).Once().
		On("Read").Return(archive.LedgerTransaction{}, io.EOF).Once()

	mockReaderLedgerTheRest.
		On("Read").Return(archive.LedgerTransaction{}, io.EOF)

	checkpoint := index.GetCheckpointNumber(uint32(ledgerSeq))
	mockStore := &index.MockStore{}
	mockStore.
		On("NextActive", accountId, mock.Anything, uint32(checkpoint)).Return(uint32(checkpoint+1), nil).
		On("NextActive", accountId, mock.Anything, uint32(checkpoint+1)).Return(uint32(checkpoint+2), nil).
		On("NextActive", accountId, mock.Anything, uint32(checkpoint+2)).Return(uint32(0), io.EOF)

	return mockArchive, mockStore
}

func testLedger(seq int) xdr.LedgerCloseMeta {
	return xdr.LedgerCloseMeta{
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					LedgerSeq: xdr.Uint32(seq),
				},
			},
		},
	}
}

func testLedgerTx(source xdr.AccountId, txIndex uint32, bumpTos ...int) archive.LedgerTransaction {
	ops := []xdr.Operation{}
	for _, bumpTo := range bumpTos {
		ops = append(ops, xdr.Operation{
			Body: xdr.OperationBody{
				BumpSequenceOp: &xdr.BumpSequenceOp{
					BumpTo: xdr.SequenceNumber(bumpTo),
				},
			},
		})
	}

	tx := archive.LedgerTransaction{
		Envelope: xdr.TransactionEnvelope{
			Type: xdr.EnvelopeTypeEnvelopeTypeTx,
			V1: &xdr.TransactionV1Envelope{
				Tx: xdr.Transaction{
					SourceAccount: source.ToMuxedAccount(),
					Fee:           xdr.Uint32(1),
					Operations:    ops,
				},
			},
		},
		Index: txIndex,
	}

	return tx
}

func newTransactionService(ctx context.Context) TransactionsService {
	passphrase := "White New England clam chowder"
	archive, store := mockArchiveAndIndex(ctx, passphrase)
	return TransactionsService{
		Config: Config{
			Archive:    archive,
			IndexStore: store,
			Passphrase: passphrase,
		},
	}
}

func newOperationService(ctx context.Context) OperationsService {
	passphrase := "White New England clam chowder"
	archive, store := mockArchiveAndIndex(ctx, passphrase)
	return OperationsService{
		Config: Config{
			Archive:    archive,
			IndexStore: store,
			Passphrase: passphrase,
		},
	}
}
