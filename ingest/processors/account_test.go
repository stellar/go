package processors

import (
	"fmt"
	"testing"
	"time"

	"github.com/guregu/null"
	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/xdr"
)

func TestTransformAccount(t *testing.T) {
	type inputStruct struct {
		ledgerChange ingest.Change
	}

	type transformTest struct {
		input      inputStruct
		wantOutput AccountOutput
		wantErr    error
	}

	hardCodedInput := makeAccountTestInput()
	hardCodedOutput := makeAccountTestOutput()

	tests := []transformTest{
		{
			inputStruct{ingest.Change{
				Type: xdr.LedgerEntryTypeOffer,
				Pre:  nil,
				Post: &xdr.LedgerEntry{
					Data: xdr.LedgerEntryData{
						Type: xdr.LedgerEntryTypeOffer,
					},
				},
			},
			},
			AccountOutput{}, fmt.Errorf("could not extract account data from ledger entry; actual type is LedgerEntryTypeOffer"),
		},
		{
			inputStruct{wrapAccountEntry(xdr.AccountEntry{
				AccountId: genericAccountID,
				Balance:   -1,
			}, 0),
			},
			AccountOutput{}, fmt.Errorf("balance is negative (-1) for account: %s", genericAccountAddress),
		},
		{
			inputStruct{wrapAccountEntry(xdr.AccountEntry{
				AccountId: genericAccountID,
				Ext: xdr.AccountEntryExt{
					V: 1,
					V1: &xdr.AccountEntryExtensionV1{
						Liabilities: xdr.Liabilities{
							Buying: -1,
						},
					},
				},
			}, 0),
			},
			AccountOutput{}, fmt.Errorf("the buying liabilities count is negative (-1) for account: %s", genericAccountAddress),
		},
		{
			inputStruct{wrapAccountEntry(xdr.AccountEntry{
				AccountId: genericAccountID,
				Ext: xdr.AccountEntryExt{
					V: 1,
					V1: &xdr.AccountEntryExtensionV1{
						Liabilities: xdr.Liabilities{
							Selling: -2,
						},
					},
				},
			}, 0),
			},
			AccountOutput{}, fmt.Errorf("the selling liabilities count is negative (-2) for account: %s", genericAccountAddress),
		},
		{
			inputStruct{wrapAccountEntry(xdr.AccountEntry{
				AccountId: genericAccountID,
				SeqNum:    -3,
			}, 0),
			},
			AccountOutput{}, fmt.Errorf("account sequence number is negative (-3) for account: %s", genericAccountAddress),
		},
		{
			inputStruct{
				hardCodedInput,
			},
			hardCodedOutput, nil,
		},
	}

	for _, test := range tests {
		header := xdr.LedgerHeaderHistoryEntry{
			Header: xdr.LedgerHeader{
				ScpValue: xdr.StellarValue{
					CloseTime: 1000,
				},
				LedgerSeq: 10,
			},
		}
		actualOutput, actualError := TransformAccount(test.input.ledgerChange, header)
		assert.Equal(t, test.wantErr, actualError)
		assert.Equal(t, test.wantOutput, actualOutput)
	}
}

func wrapAccountEntry(accountEntry xdr.AccountEntry, lastModified int) ingest.Change {
	return ingest.Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: xdr.Uint32(lastModified),
			Data: xdr.LedgerEntryData{
				Type:    xdr.LedgerEntryTypeAccount,
				Account: &accountEntry,
			},
		},
	}
}

func makeAccountTestInput() ingest.Change {

	ledgerEntry := xdr.LedgerEntry{
		LastModifiedLedgerSeq: 30705278,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeAccount,
			Account: &xdr.AccountEntry{
				AccountId:     testAccount1ID,
				Balance:       10959979,
				SeqNum:        117801117454198833,
				NumSubEntries: 141,
				InflationDest: &testAccount2ID,
				Flags:         4,
				HomeDomain:    "examplehome.com",
				Thresholds:    xdr.Thresholds([4]byte{2, 1, 3, 5}),
				Ext: xdr.AccountEntryExt{
					V: 1,
					V1: &xdr.AccountEntryExtensionV1{
						Liabilities: xdr.Liabilities{
							Buying:  1000,
							Selling: 1500,
						},
						Ext: xdr.AccountEntryExtensionV1Ext{
							V: 2,
							V2: &xdr.AccountEntryExtensionV2{
								NumSponsored:  3,
								NumSponsoring: 1,
							},
						},
					},
				},
			},
		},
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: &testAccount3ID,
			},
		},
	}
	return ingest.Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  &ledgerEntry,
		Post: nil,
	}
}

func makeAccountTestOutput() AccountOutput {
	return AccountOutput{
		AccountID:            testAccount1Address,
		Balance:              1.0959979,
		BuyingLiabilities:    0.0001,
		SellingLiabilities:   0.00015,
		SequenceNumber:       117801117454198833,
		NumSubentries:        141,
		InflationDestination: testAccount2Address,
		Flags:                4,
		HomeDomain:           "examplehome.com",
		MasterWeight:         2,
		ThresholdLow:         1,
		ThresholdMedium:      3,
		ThresholdHigh:        5,
		Sponsor:              null.StringFrom(testAccount3Address),
		NumSponsored:         3,
		NumSponsoring:        1,
		LastModifiedLedger:   30705278,
		LedgerEntryChange:    2,
		Deleted:              true,
		LedgerSequence:       10,
		ClosedAt:             time.Date(1970, time.January, 1, 0, 16, 40, 0, time.UTC),
	}
}
