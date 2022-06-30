package archive

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/require"
)

func TestItGetsSequentialOperationsForLimitBeyondEnd(tt *testing.T) {
	// l=1586111, t=1, o=1
	ctx := context.Background()
	cursor := int64(6812294872829953)
	passphrase := "Red New England clam chowder"
	archiveWrapper := Wrapper{Archive: mockArchiveFixture(ctx, passphrase), Passphrase: passphrase}
	ops, err := archiveWrapper.GetOperations(ctx, cursor, 5)
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
	archiveWrapper := Wrapper{Archive: mockArchiveFixture(ctx, passphrase), Passphrase: passphrase}
	ops, err := archiveWrapper.GetOperations(ctx, cursor, 2)
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
	archiveWrapper := Wrapper{Archive: mockArchiveFixture(ctx, passphrase), Passphrase: passphrase}
	txs, err := archiveWrapper.GetTransactions(ctx, cursor, 5)
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
	archiveWrapper := Wrapper{Archive: mockArchiveFixture(ctx, passphrase), Passphrase: passphrase}
	txs, err := archiveWrapper.GetTransactions(ctx, cursor, 1)
	require.NoError(tt, err)
	require.Len(tt, txs, 1)
	require.Equal(tt, txs[0].LedgerHeader.LedgerSeq, xdr.Uint32(1586111))
	require.Equal(tt, txs[0].TxIndex, int32(2))
}

func mockArchiveFixture(ctx context.Context, passphrase string) *MockArchive {
	mockArchive := &MockArchive{}
	mockReaderLedger1 := &MockLedgerTransactionReader{}
	mockReaderLedger2 := &MockLedgerTransactionReader{}

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
	mockReaderLedger1.On("Read").Return(LedgerTransaction{}, io.EOF).Once()
	mockReaderLedger2.On("Read").Return(expectedLedger2Transaction1, nil).Once()
	mockReaderLedger2.On("Read").Return(LedgerTransaction{}, io.EOF).Once()
	return mockArchive
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

func testLedgerTx(source xdr.AccountId, bumpTos ...int) LedgerTransaction {

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

	tx := LedgerTransaction{
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
