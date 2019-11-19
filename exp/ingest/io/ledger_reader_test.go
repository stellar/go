package io

import (
	"testing"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestCloseTime(t *testing.T) {
	backend := &ledgerbackend.MockDatabaseBackend{}
	expectedCloseTime := int64(1234)
	backend.On("GetLedger", uint32(1)).Return(
		true,
		ledgerbackend.LedgerCloseMeta{
			CloseTime: expectedCloseTime,
		},
		nil,
	).Once()
	reader, err := NewDBLedgerReader(1, backend)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	backend.AssertExpectations(t)
	assert.Equal(t, expectedCloseTime, reader.CloseTime())
}

func txResultPair(code xdr.TransactionResultCode) xdr.TransactionResultPair {
	return xdr.TransactionResultPair{Result: xdr.TransactionResult{
		Result: xdr.TransactionResultResult{Code: code},
	}}
}

func TestLedgerReaderCounts(t *testing.T) {
	backend := &ledgerbackend.MockDatabaseBackend{}
	backend.On("GetLedger", uint32(1)).Return(true, ledgerbackend.LedgerCloseMeta{}, nil).Once()
	reader, err := NewDBLedgerReader(1, backend)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	backend.AssertExpectations(t)

	assert.Equal(t, 0, reader.SuccessfulTransactionCount())
	assert.Equal(t, 0, reader.FailedTransactionCount())
	assert.Equal(t, 0, reader.SuccessfulLedgerOperationCount())

	backend.On("GetLedger", uint32(1)).Return(
		true,
		ledgerbackend.LedgerCloseMeta{
			TransactionEnvelope: []xdr.TransactionEnvelope{
				xdr.TransactionEnvelope{
					Tx: xdr.Transaction{
						Operations: []xdr.Operation{
							xdr.Operation{},
						},
					},
				},
				xdr.TransactionEnvelope{
					Tx: xdr.Transaction{
						Operations: []xdr.Operation{
							xdr.Operation{},
							xdr.Operation{},
						},
					},
				},
				xdr.TransactionEnvelope{
					Tx: xdr.Transaction{
						Operations: []xdr.Operation{
							xdr.Operation{},
							xdr.Operation{},
							xdr.Operation{},
						},
					},
				},
			},
			TransactionMeta: []xdr.TransactionMeta{
				xdr.TransactionMeta{},
				xdr.TransactionMeta{},
				xdr.TransactionMeta{},
			},
			TransactionResult: []xdr.TransactionResultPair{
				txResultPair(xdr.TransactionResultCodeTxSuccess),
				txResultPair(xdr.TransactionResultCodeTxBadAuth),
				txResultPair(xdr.TransactionResultCodeTxSuccess),
			},
			TransactionFeeChanges: []xdr.LedgerEntryChanges{
				xdr.LedgerEntryChanges{},
				xdr.LedgerEntryChanges{},
				xdr.LedgerEntryChanges{},
			},
		},
		nil,
	).Once()
	reader, err = NewDBLedgerReader(1, backend)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	backend.AssertExpectations(t)

	assert.Equal(t, 2, reader.SuccessfulTransactionCount())
	assert.Equal(t, 1, reader.FailedTransactionCount())
	assert.Equal(t, 4, reader.SuccessfulLedgerOperationCount())
}
