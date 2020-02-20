package io

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/xdr"
)

const (
	feeAddress     = "GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A"
	metaAddress    = "GACMZD5VJXTRLKVET72CETCYKELPNCOTTBDC6DHFEUPLG5DHEK534JQX"
	upgradeAddress = "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2"
)

func TestNewLedgerChangeReaderFails(t *testing.T) {
	mock := &ledgerbackend.MockDatabaseBackend{}
	seq := uint32(123)
	mock.On("GetLedger", seq).Return(
		true,
		ledgerbackend.LedgerCloseMeta{},
		fmt.Errorf("ledger error"),
	).Once()
	_, err := NewLedgerChangeReader(context.Background(), seq, mock)
	assert.EqualError(
		t,
		err,
		"error reading ledger from backend: ledger error",
	)
}

func TestNewLedgerChangeReaderLedgerDoesNotExist(t *testing.T) {
	mock := &ledgerbackend.MockDatabaseBackend{}
	seq := uint32(123)
	mock.On("GetLedger", seq).Return(
		false,
		ledgerbackend.LedgerCloseMeta{},
		nil,
	).Once()
	_, err := NewLedgerChangeReader(context.Background(), seq, mock)
	assert.Equal(
		t,
		err,
		ErrNotFound,
	)
}

func TestNewLedgerChangeReaderSucceeds(t *testing.T) {
	mock := &ledgerbackend.MockDatabaseBackend{}
	seq := uint32(123)

	header := xdr.LedgerHeaderHistoryEntry{
		Hash: xdr.Hash{1, 2, 3},
		Header: xdr.LedgerHeader{
			LedgerVersion: 7,
		},
	}

	mock.On("GetLedger", seq).Return(
		true,
		ledgerbackend.LedgerCloseMeta{
			LedgerHeader: header,
		},
		nil,
	).Once()

	reader, err := NewLedgerChangeReader(context.Background(), seq, mock)
	assert.NoError(t, err)

	assert.Equal(t, reader.GetHeader(), header)
}

func buildChange(account string, balance int64) xdr.LedgerEntryChange {
	return xdr.LedgerEntryChange{
		Type: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
		Created: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress(account),
					Balance:   xdr.Int64(balance),
				},
			},
		},
	}
}

type balanceEntry struct {
	address string
	balance int64
}

func parseChange(change Change) balanceEntry {
	address := change.Post.Data.Account.AccountId.Address()
	balance := int64(change.Post.Data.Account.Balance)

	return balanceEntry{address, balance}
}

func assertChangesEqual(
	t *testing.T,
	sequence uint32,
	backend ledgerbackend.LedgerBackend,
	expected []balanceEntry,
) {
	reader, err := NewLedgerChangeReader(context.Background(), sequence, backend)
	assert.NoError(t, err)

	changes := []balanceEntry{}
	for {
		change, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		changes = append(changes, parseChange(change))
	}

	assert.Equal(t, expected, changes)
}

func TestLedgerChangeReaderOrder(t *testing.T) {
	mock := &ledgerbackend.MockDatabaseBackend{}
	seq := uint32(123)

	ledger := ledgerbackend.LedgerCloseMeta{
		TransactionResult: []xdr.TransactionResultPair{
			xdr.TransactionResultPair{},
			xdr.TransactionResultPair{},
		},
		TransactionEnvelope: []xdr.TransactionEnvelope{
			xdr.TransactionEnvelope{},
			xdr.TransactionEnvelope{},
		},
		TransactionMeta: []xdr.TransactionMeta{
			xdr.TransactionMeta{
				V: 1,
				V1: &xdr.TransactionMetaV1{
					Operations: []xdr.OperationMeta{
						{
							Changes: xdr.LedgerEntryChanges{
								buildChange(
									metaAddress,
									300,
								),
								buildChange(
									metaAddress,
									400,
								),
							},
						},
					},
				},
			},
			xdr.TransactionMeta{
				V: 2,
				V2: &xdr.TransactionMetaV2{
					TxChangesBefore: xdr.LedgerEntryChanges{
						buildChange(metaAddress, 600),
					},
					Operations: []xdr.OperationMeta{
						{
							Changes: xdr.LedgerEntryChanges{
								buildChange(metaAddress, 700),
							},
						},
					},
					TxChangesAfter: xdr.LedgerEntryChanges{
						buildChange(metaAddress, 800),
						buildChange(metaAddress, 900),
					},
				},
			},
		},
		TransactionFeeChanges: []xdr.LedgerEntryChanges{
			xdr.LedgerEntryChanges{
				buildChange(feeAddress, 100),
				buildChange(feeAddress, 200),
			},
			xdr.LedgerEntryChanges{
				buildChange(feeAddress, 300),
			},
		},
		UpgradesMeta: []xdr.LedgerEntryChanges{
			xdr.LedgerEntryChanges{
				buildChange(upgradeAddress, 2),
			},
			xdr.LedgerEntryChanges{
				buildChange(upgradeAddress, 3),
			},
		},
	}
	mock.On("GetLedger", seq).Return(true, ledger, nil).Once()

	assertChangesEqual(t, seq, mock, []balanceEntry{
		{feeAddress, 100},
		{feeAddress, 200},
		{feeAddress, 300},
		{metaAddress, 300},
		{metaAddress, 400},
		{metaAddress, 600},
		{metaAddress, 700},
		{metaAddress, 800},
		{metaAddress, 900},
		{upgradeAddress, 2},
		{upgradeAddress, 3},
	})
	mock.AssertExpectations(t)

	ledger.TransactionFeeChanges = []xdr.LedgerEntryChanges{
		xdr.LedgerEntryChanges{}, xdr.LedgerEntryChanges{},
	}
	mock.On("GetLedger", seq).Return(true, ledger, nil).Once()

	assertChangesEqual(t, seq, mock, []balanceEntry{
		{metaAddress, 300},
		{metaAddress, 400},
		{metaAddress, 600},
		{metaAddress, 700},
		{metaAddress, 800},
		{metaAddress, 900},
		{upgradeAddress, 2},
		{upgradeAddress, 3},
	})
	mock.AssertExpectations(t)

	ledger.UpgradesMeta = []xdr.LedgerEntryChanges{
		xdr.LedgerEntryChanges{}, xdr.LedgerEntryChanges{},
	}
	mock.On("GetLedger", seq).Return(true, ledger, nil).Once()

	assertChangesEqual(t, seq, mock, []balanceEntry{
		{metaAddress, 300},
		{metaAddress, 400},
		{metaAddress, 600},
		{metaAddress, 700},
		{metaAddress, 800},
		{metaAddress, 900},
	})
	mock.AssertExpectations(t)

	ledger.TransactionMeta = []xdr.TransactionMeta{
		xdr.TransactionMeta{
			V: 1,
			V1: &xdr.TransactionMetaV1{
				Operations: []xdr.OperationMeta{},
			},
		},
		xdr.TransactionMeta{
			V: 1,
			V1: &xdr.TransactionMetaV1{
				Operations: []xdr.OperationMeta{},
			},
		},
	}
	mock.On("GetLedger", seq).Return(true, ledger, nil).Once()

	assertChangesEqual(t, seq, mock, []balanceEntry{})
	mock.AssertExpectations(t)
}

func TestLedgerChangeReaderContext(t *testing.T) {
	mock := &ledgerbackend.MockDatabaseBackend{}
	seq := uint32(123)

	ledger := ledgerbackend.LedgerCloseMeta{
		TransactionResult: []xdr.TransactionResultPair{
			xdr.TransactionResultPair{},
			xdr.TransactionResultPair{},
		},
		TransactionEnvelope: []xdr.TransactionEnvelope{
			xdr.TransactionEnvelope{},
			xdr.TransactionEnvelope{},
		},
		TransactionMeta: []xdr.TransactionMeta{
			xdr.TransactionMeta{
				V: 1,
				V1: &xdr.TransactionMetaV1{
					Operations: []xdr.OperationMeta{},
				},
			},
			xdr.TransactionMeta{
				V: 1,
				V1: &xdr.TransactionMetaV1{
					Operations: []xdr.OperationMeta{},
				},
			},
		},
		TransactionFeeChanges: []xdr.LedgerEntryChanges{
			xdr.LedgerEntryChanges{
				buildChange(feeAddress, 100),
			},
			xdr.LedgerEntryChanges{
				buildChange(feeAddress, 300),
			},
		},
		UpgradesMeta: []xdr.LedgerEntryChanges{
			xdr.LedgerEntryChanges{
				buildChange(upgradeAddress, 2),
			},
			xdr.LedgerEntryChanges{
				buildChange(upgradeAddress, 3),
			},
		},
	}

	mock.On("GetLedger", seq).Return(true, ledger, nil).Once()
	ctx, cancel := context.WithCancel(context.Background())
	reader, err := NewLedgerChangeReader(ctx, seq, mock)
	mock.AssertExpectations(t)
	assert.NoError(t, err)

	cancel()
	_, err = reader.Read()
	assert.Equal(t, context.Canceled, err)

	mock.On("GetLedger", seq).Return(true, ledger, nil).Once()
	ctx, cancel = context.WithCancel(context.Background())
	reader, err = NewLedgerChangeReader(ctx, seq, mock)
	mock.AssertExpectations(t)
	assert.NoError(t, err)

	change, err := reader.Read()
	assert.Equal(t, balanceEntry{feeAddress, 100}, parseChange(change))
	assert.NoError(t, err)

	cancel()
	_, err = reader.Read()
	assert.Equal(t, context.Canceled, err)
}
