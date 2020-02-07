package expingest

import (
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
)

type fakeLedgerBackend struct {
	numTransactions       int
	changesPerTransaction int
}

func (fakeLedgerBackend) GetLatestLedgerSequence() (uint32, error) {
	return 1, nil
}

func fakeChange(account string, balance int64) xdr.LedgerEntryChange {
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

func (f fakeLedgerBackend) GetLedger(sequence uint32) (bool, ledgerbackend.LedgerCloseMeta, error) {
	ledgerCloseMeta := ledgerbackend.LedgerCloseMeta{
		LedgerHeader: xdr.LedgerHeaderHistoryEntry{
			Hash: xdr.Hash{1, 2, 3, 4, 5, 6},
			Header: xdr.LedgerHeader{
				LedgerVersion: 7,
				LedgerSeq:     xdr.Uint32(1),
			},
		},
		TransactionResult:     make([]xdr.TransactionResultPair, f.numTransactions),
		TransactionEnvelope:   make([]xdr.TransactionEnvelope, f.numTransactions),
		TransactionMeta:       make([]xdr.TransactionMeta, f.numTransactions),
		TransactionFeeChanges: make([]xdr.LedgerEntryChanges, f.numTransactions),
		UpgradesMeta:          []xdr.LedgerEntryChanges{},
	}
	for i := 0; i < f.numTransactions; i++ {
		var results []xdr.OperationResult
		ledgerCloseMeta.TransactionResult[i] = xdr.TransactionResultPair{
			TransactionHash: xdr.Hash{1, byte(i % 256), byte((i / 256) % 256)},
			Result: xdr.TransactionResult{
				Result: xdr.TransactionResultResult{
					Code:    xdr.TransactionResultCodeTxSuccess,
					Results: &results,
				},
			},
		}
		ledgerCloseMeta.TransactionEnvelope[i] = xdr.TransactionEnvelope{
			Tx: xdr.Transaction{
				SourceAccount: xdr.MustAddress("GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A"),
			},
		}
		ledgerCloseMeta.TransactionMeta[i] = xdr.TransactionMeta{
			V: 1,
			V1: &xdr.TransactionMetaV1{
				Operations: []xdr.OperationMeta{},
			},
		}

		feeChanges := xdr.LedgerEntryChanges{}
		for j := 0; j < f.changesPerTransaction; j++ {
			feeChanges = append(feeChanges, fakeChange(keypair.MustRandom().Address(), 100))
		}
		ledgerCloseMeta.TransactionFeeChanges[i] = feeChanges
	}

	return true, ledgerCloseMeta, nil
}

func (fakeLedgerBackend) Close() error {
	return nil
}
