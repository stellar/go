package services

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestItGetsSequentialOperationsForLimitBeyondEnd(tt *testing.T) {
	// l=1586111, t=1, o=1
	ctx := context.Background()
	cursor := int64(6812294872829953)
	passphrase := "Red New England clam chowder"
	archive := mockArchiveFixture(ctx, passphrase)
	opsService := OperationsService{
		Archive:    archive,
		IndexStore: &index.MockStore{},
		Passphrase: passphrase,
	}
	ops, err := opsService.GetOperations(ctx, cursor, 5)
	require.NoError(tt, err)
	require.Len(tt, ops, 3)
	require.Equal(tt, ops[0].LedgerHeader.LedgerSeq, xdr.Uint32(1586111))
	require.Equal(tt, ops[0].TxIndex, int32(1))
	require.Equal(tt, ops[0].OpIndex, int32(2))
	require.Equal(tt, ops[1].LedgerHeader.LedgerSeq, xdr.Uint32(1586111))
	require.Equal(tt, ops[1].TxIndex, int32(2))
	require.Equal(tt, ops[1].OpIndex, int32(1))
	require.Equal(tt, ops[2].LedgerHeader.LedgerSeq, xdr.Uint32(1586112))
	require.Equal(tt, ops[2].TxIndex, int32(1))
	require.Equal(tt, ops[2].OpIndex, int32(1))
}

func TestItGetsSequentialOperationsForLimitBeforeEnd(tt *testing.T) {
	// l=1586111, t=1, o=1
	ctx := context.Background()
	cursor := int64(6812294872829953)
	passphrase := "White New England clam chowder"
	archive := mockArchiveFixture(ctx, passphrase)
	opsService := OperationsService{
		Archive:    archive,
		IndexStore: &index.MockStore{},
		Passphrase: passphrase,
	}
	ops, err := opsService.GetOperations(ctx, cursor, 2)
	require.NoError(tt, err)
	require.Len(tt, ops, 2)
	require.Equal(tt, ops[0].LedgerHeader.LedgerSeq, xdr.Uint32(1586111))
	require.Equal(tt, ops[0].TxIndex, int32(1))
	require.Equal(tt, ops[0].OpIndex, int32(2))
	require.Equal(tt, ops[1].LedgerHeader.LedgerSeq, xdr.Uint32(1586111))
	require.Equal(tt, ops[1].TxIndex, int32(2))
	require.Equal(tt, ops[1].OpIndex, int32(1))
}

func TestItGetsSequentialTransactionsForLimitBeyondEnd(tt *testing.T) {
	// l=1586111, t=1, o=1
	ctx := context.Background()
	cursor := int64(6812294872829953)
	passphrase := "White New England clam chowder"
	archive := mockArchiveFixture(ctx, passphrase)
	txService := TransactionsService{
		Archive:    archive,
		IndexStore: &index.MockStore{},
		Passphrase: passphrase,
	}
	txs, err := txService.GetTransactions(ctx, cursor, 5)
	require.NoError(tt, err)
	require.Len(tt, txs, 2)
	require.Equal(tt, txs[0].LedgerHeader.LedgerSeq, xdr.Uint32(1586111))
	require.Equal(tt, txs[0].TxIndex, int32(2))
	require.Equal(tt, txs[1].LedgerHeader.LedgerSeq, xdr.Uint32(1586112))
	require.Equal(tt, txs[1].TxIndex, int32(1))
}

func TestItGetsSequentialTransactionsForLimitBeforeEnd(tt *testing.T) {
	// l=1586111, t=1, o=1
	ctx := context.Background()
	cursor := int64(6812294872829953)
	passphrase := "White New England clam chowder"
	archive := mockArchiveFixture(ctx, passphrase)
	txService := TransactionsService{
		Archive:    archive,
		IndexStore: &index.MockStore{},
		Passphrase: passphrase,
	}
	txs, err := txService.GetTransactions(ctx, cursor, 1)
	require.NoError(tt, err)
	require.Len(tt, txs, 1)
	require.Equal(tt, txs[0].LedgerHeader.LedgerSeq, xdr.Uint32(1586111))
	require.Equal(tt, txs[0].TxIndex, int32(2))
}

func TestItGetsTransactionsByAccount(tt *testing.T) {
	// l=1586111, t=1, o=1
	// cursor = 6812294872829953, checkpoint=24782
	cursor := int64(6812294872829953)
	ctx := context.Background()
	passphrase := "White New England clam chowder"
	archive, store := mockArchiveAndIndex(ctx, passphrase)
	txService := TransactionsService{
		Archive:    archive,
		IndexStore: store,
		Passphrase: passphrase,
	}
	accountId := "GDCXSQPVE45DVGT2ZRFFIIHSJ2EJED65W6AELGWIDRMPMWNXCEBJ4FKX"
	// this should start at next checkpoint
	txs, err := txService.GetTransactionsByAccount(ctx, cursor, 1, accountId)
	require.NoError(tt, err)
	require.Len(tt, txs, 1)
	require.Equal(tt, txs[0].LedgerHeader.LedgerSeq, xdr.Uint32(1586113))
	require.Equal(tt, txs[0].TxIndex, int32(2))
}

func TestItGetsTransactionsByAccountAndPageLimit(tt *testing.T) {
	// l=1586111, t=1, o=1
	// cursor = 6812294872829953, checkpoint=24782
	cursor := int64(6812294872829953)
	ctx := context.Background()
	passphrase := "White New England clam chowder"
	archive, store := mockArchiveAndIndex(ctx, passphrase)
	txService := TransactionsService{
		Archive:    archive,
		IndexStore: store,
		Passphrase: passphrase,
	}
	accountId := "GDCXSQPVE45DVGT2ZRFFIIHSJ2EJED65W6AELGWIDRMPMWNXCEBJ4FKX"
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
	// l=1586111, t=1, o=1
	// cursor = 6812294872829953, checkpoint=24782
	cursor := int64(6812294872829953)
	ctx := context.Background()
	passphrase := "White New England clam chowder"
	archive, store := mockArchiveAndIndex(ctx, passphrase)
	opsService := OperationsService{
		Archive:    archive,
		IndexStore: store,
		Passphrase: passphrase,
	}
	accountId := "GDCXSQPVE45DVGT2ZRFFIIHSJ2EJED65W6AELGWIDRMPMWNXCEBJ4FKX"
	// this should start at next checkpoint
	ops, err := opsService.GetOperationsByAccount(ctx, cursor, 1, accountId)
	require.NoError(tt, err)
	require.Len(tt, ops, 1)
	require.Equal(tt, ops[0].LedgerHeader.LedgerSeq, xdr.Uint32(1586113))
	require.Equal(tt, ops[0].TxIndex, int32(2))
}

func TestItGetsOperationsByAccountAndPageLimit(tt *testing.T) {
	// l=1586111, t=1, o=1
	// cursor = 6812294872829953, checkpoint=24782
	cursor := int64(6812294872829953)
	ctx := context.Background()
	passphrase := "White New England clam chowder"
	archive, store := mockArchiveAndIndex(ctx, passphrase)
	opsService := OperationsService{
		Archive:    archive,
		IndexStore: store,
		Passphrase: passphrase,
	}
	accountId := "GDCXSQPVE45DVGT2ZRFFIIHSJ2EJED65W6AELGWIDRMPMWNXCEBJ4FKX"
	// this should start at next checkpoint
	ops, err := opsService.GetOperationsByAccount(ctx, cursor, 5, accountId)
	require.NoError(tt, err)
	require.Len(tt, ops, 2)
	require.Equal(tt, ops[0].LedgerHeader.LedgerSeq, xdr.Uint32(1586113))
	require.Equal(tt, ops[0].TxIndex, int32(2))
	require.Equal(tt, ops[1].LedgerHeader.LedgerSeq, xdr.Uint32(1586114))
	require.Equal(tt, ops[1].TxIndex, int32(1))
}

func mockArchiveFixture(ctx context.Context, passphrase string) archive.Archive {

	mockArchive := &archive.MockArchive{}
	mockReaderLedger1 := &archive.MockLedgerTransactionReader{}
	mockReaderLedger2 := &archive.MockLedgerTransactionReader{}

	expectedLedger1 := testLedger(1586111)
	expectedLedger2 := testLedger(1586112)
	source := xdr.MustAddress("GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU")
	// assert results iterate sequentially across ops-tx-ledgers
	expectedLedger1Transaction1 := testLedgerTx(source, 34, 34)
	expectedLedger1Transaction2 := testLedgerTx(source, 34)
	expectedLedger2Transaction1 := testLedgerTx(source, 34)

	mockArchive.On("GetLedger", ctx, uint32(1586111)).Return(expectedLedger1, nil)
	mockArchive.On("GetLedger", ctx, uint32(1586112)).Return(expectedLedger2, nil)
	mockArchive.On("GetLedger", ctx, uint32(1586113)).Return(xdr.LedgerCloseMeta{}, fmt.Errorf("ledger not found"))
	mockArchive.On("NewLedgerTransactionReaderFromLedgerCloseMeta", passphrase, expectedLedger1).Return(mockReaderLedger1, nil)
	mockArchive.On("NewLedgerTransactionReaderFromLedgerCloseMeta", passphrase, expectedLedger2).Return(mockReaderLedger2, nil)
	mockReaderLedger1.On("Read").Return(expectedLedger1Transaction1, nil).Once()
	mockReaderLedger1.On("Read").Return(expectedLedger1Transaction2, nil).Once()
	mockReaderLedger1.On("Read").Return(archive.LedgerTransaction{}, io.EOF).Once()
	mockReaderLedger2.On("Read").Return(expectedLedger2Transaction1, nil).Once()
	mockReaderLedger2.On("Read").Return(archive.LedgerTransaction{}, io.EOF).Once()

	return mockArchive
}

func mockArchiveAndIndex(ctx context.Context, passphrase string) (archive.Archive, index.Store) {

	mockArchive := &archive.MockArchive{}
	mockReaderLedger1 := &archive.MockLedgerTransactionReader{}
	mockReaderLedger2 := &archive.MockLedgerTransactionReader{}
	mockReaderLedger3 := &archive.MockLedgerTransactionReader{}
	mockReaderLedgerTheRest := &archive.MockLedgerTransactionReader{}

	expectedLedger1 := testLedger(1586112)
	expectedLedger2 := testLedger(1586113)
	expectedLedger3 := testLedger(1586114)
	source := xdr.MustAddress("GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU")
	source2 := xdr.MustAddress("GDCXSQPVE45DVGT2ZRFFIIHSJ2EJED65W6AELGWIDRMPMWNXCEBJ4FKX")
	// assert results iterate sequentially across ops-tx-ledgers
	expectedLedger1Transaction1 := testLedgerTx(source, 34, 34)
	expectedLedger1Transaction2 := testLedgerTx(source, 34)
	expectedLedger2Transaction1 := testLedgerTx(source, 34)
	expectedLedger2Transaction2 := testLedgerTx(source2, 34)
	expectedLedger3Transaction1 := testLedgerTx(source2, 34)
	expectedLedger3Transaction2 := testLedgerTx(source, 34)

	mockArchive.On("GetLedger", ctx, uint32(1586112)).Return(expectedLedger1, nil)
	mockArchive.On("GetLedger", ctx, uint32(1586113)).Return(expectedLedger2, nil)
	mockArchive.On("GetLedger", ctx, uint32(1586114)).Return(expectedLedger3, nil)
	mockArchive.On("GetLedger", ctx, mock.Anything).Return(xdr.LedgerCloseMeta{}, nil)

	mockArchive.On("NewLedgerTransactionReaderFromLedgerCloseMeta", passphrase, expectedLedger1).Return(mockReaderLedger1, nil)
	mockArchive.On("NewLedgerTransactionReaderFromLedgerCloseMeta", passphrase, expectedLedger2).Return(mockReaderLedger2, nil)
	mockArchive.On("NewLedgerTransactionReaderFromLedgerCloseMeta", passphrase, expectedLedger3).Return(mockReaderLedger3, nil)
	mockArchive.On("NewLedgerTransactionReaderFromLedgerCloseMeta", passphrase, mock.Anything).Return(mockReaderLedgerTheRest, nil)

	partialParticipants := make(map[string]struct{})
	partialParticipants[source.Address()] = struct{}{}
	allParticipants := make(map[string]struct{})
	allParticipants[source.Address()] = struct{}{}
	allParticipants[source2.Address()] = struct{}{}

	mockArchive.On("GetTransactionParticipants", expectedLedger1Transaction1).Return(partialParticipants, nil)
	mockArchive.On("GetTransactionParticipants", expectedLedger1Transaction2).Return(partialParticipants, nil)
	mockArchive.On("GetTransactionParticipants", expectedLedger2Transaction1).Return(partialParticipants, nil)
	mockArchive.On("GetTransactionParticipants", expectedLedger2Transaction2).Return(allParticipants, nil)
	mockArchive.On("GetTransactionParticipants", expectedLedger3Transaction1).Return(allParticipants, nil)
	mockArchive.On("GetTransactionParticipants", expectedLedger3Transaction2).Return(partialParticipants, nil)

	mockArchive.On("GetOperationParticipants", expectedLedger1Transaction1, mock.Anything, int(1)).Return(partialParticipants, nil)
	mockArchive.On("GetOperationParticipants", expectedLedger1Transaction1, mock.Anything, int(2)).Return(partialParticipants, nil)
	mockArchive.On("GetOperationParticipants", expectedLedger1Transaction2, mock.Anything, int(1)).Return(partialParticipants, nil)
	mockArchive.On("GetOperationParticipants", expectedLedger2Transaction1, mock.Anything, int(1)).Return(partialParticipants, nil)
	mockArchive.On("GetOperationParticipants", expectedLedger2Transaction2, mock.Anything, int(1)).Return(allParticipants, nil)
	mockArchive.On("GetOperationParticipants", expectedLedger3Transaction1, mock.Anything, int(1)).Return(allParticipants, nil)
	mockArchive.On("GetOperationParticipants", expectedLedger3Transaction2, mock.Anything, int(1)).Return(partialParticipants, nil)

	mockReaderLedger1.On("Read").Return(expectedLedger1Transaction1, nil).Once()
	mockReaderLedger1.On("Read").Return(expectedLedger1Transaction2, nil).Once()
	mockReaderLedger1.On("Read").Return(archive.LedgerTransaction{}, io.EOF).Once()
	mockReaderLedger2.On("Read").Return(expectedLedger2Transaction1, nil).Once()
	mockReaderLedger2.On("Read").Return(expectedLedger2Transaction2, nil).Once()
	mockReaderLedger2.On("Read").Return(archive.LedgerTransaction{}, io.EOF).Once()
	mockReaderLedger3.On("Read").Return(expectedLedger3Transaction1, nil).Once()
	mockReaderLedger3.On("Read").Return(expectedLedger3Transaction2, nil).Once()
	mockReaderLedger3.On("Read").Return(archive.LedgerTransaction{}, io.EOF).Once()
	mockReaderLedgerTheRest.On("Read").Return(archive.LedgerTransaction{}, io.EOF)

	mockStore := &index.MockStore{}
	mockStore.On("NextActive", "GDCXSQPVE45DVGT2ZRFFIIHSJ2EJED65W6AELGWIDRMPMWNXCEBJ4FKX", mock.Anything, uint32(24782)).Return(uint32(24783), nil)
	mockStore.On("NextActive", "GDCXSQPVE45DVGT2ZRFFIIHSJ2EJED65W6AELGWIDRMPMWNXCEBJ4FKX", mock.Anything, uint32(24783)).Return(uint32(0), io.EOF)

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

func testLedgerTx(source xdr.AccountId, bumpTos ...int) archive.LedgerTransaction {

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
	}

	return tx
}