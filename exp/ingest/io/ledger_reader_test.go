package io

import (
	"testing"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/xdr"
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
	if got := reader.CloseTime(); got != expectedCloseTime {
		t.Fatalf("expected close time of %v but got %v", expectedCloseTime, got)

	}
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

	if got := reader.SuccessfulTransactionCount(); got != 0 {
		t.Fatalf("expected successful transaction count of %v but got %v", 0, got)
	}
	if got := reader.FailedTransactionCount(); got != 0 {
		t.Fatalf("expected failed transaction count of %v but got %v", 0, got)
	}
	if got := reader.SuccessfulLedgerOperationCount(); got != 0 {
		t.Fatalf("expected successful operation count of %v but got %v", 0, got)
	}

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

	if got := reader.SuccessfulTransactionCount(); got != 2 {
		t.Fatalf("expected successful transaction count of %v but got %v", 2, got)
	}
	if got := reader.FailedTransactionCount(); got != 1 {
		t.Fatalf("expected failed transaction count of %v but got %v", 1, got)
	}
	if got := reader.SuccessfulLedgerOperationCount(); got != 4 {
		t.Fatalf("expected successful operation count of %v but got %v", 4, got)
	}
}
