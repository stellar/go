package ingest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/xdr"
)

func TestTransformEntry_ClaimableBalance(t *testing.T) {

	input := xdr.LedgerEntry{
		LastModifiedLedgerSeq: 20,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeClaimableBalance,
			ClaimableBalance: &xdr.ClaimableBalanceEntry{
				Claimants: []xdr.Claimant{
					{
						Type: xdr.ClaimantTypeClaimantTypeV0,
						V0: &xdr.ClaimantV0{
							Destination: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
						},
					},
					{
						Type: xdr.ClaimantTypeClaimantTypeV0,
						V0: &xdr.ClaimantV0{
							Destination: xdr.MustAddress("GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A"),
						},
					},
				},
				Ext: xdr.ClaimableBalanceEntryExt{
					V: 1,
					V1: &xdr.ClaimableBalanceEntryExtensionV1{
						Flags: 0,
					},
				},
			},
		},
	}

	expectedOutput := xdr.LedgerEntry{
		LastModifiedLedgerSeq: 20,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeClaimableBalance,
			ClaimableBalance: &xdr.ClaimableBalanceEntry{
				Claimants: []xdr.Claimant{
					{
						Type: xdr.ClaimantTypeClaimantTypeV0,
						V0: &xdr.ClaimantV0{
							Destination: xdr.MustAddress("GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A"),
						},
					},
					{
						Type: xdr.ClaimantTypeClaimantTypeV0,
						V0: &xdr.ClaimantV0{
							Destination: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
						},
					},
				},
				Ext: xdr.ClaimableBalanceEntryExt{
					V: 0,
				},
			},
		},
		Ext: xdr.LedgerEntryExt{
			V:  1,
			V1: &xdr.LedgerEntryExtensionV1{},
		},
	}
	_, output := transformEntry(input)

	assert.Equal(t, expectedOutput, output)

}
