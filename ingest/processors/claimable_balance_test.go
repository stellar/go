package processors

import (
	"testing"
	"time"

	"github.com/guregu/null"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestTransformClaimableBalance(t *testing.T) {
	type inputStruct struct {
		ingest ingest.Change
	}
	type transformTest struct {
		input      inputStruct
		wantOutput ClaimableBalanceOutput
		wantErr    error
	}
	inputChange := makeClaimableBalanceTestInput()
	output := makeClaimableBalanceTestOutput()

	input := inputStruct{
		inputChange,
	}

	tests := []transformTest{
		{
			input:      input,
			wantOutput: output,
			wantErr:    nil,
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
		actualOutput, actualError := TransformClaimableBalance(test.input.ingest, header)
		assert.Equal(t, test.wantErr, actualError)
		assert.Equal(t, test.wantOutput, actualOutput)
	}
}

func makeClaimableBalanceTestInput() ingest.Change {
	ledgerEntry := xdr.LedgerEntry{
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: &xdr.AccountId{
					Type:    0,
					Ed25519: &xdr.Uint256{1, 2, 3, 4, 5, 6, 7, 8, 9},
				},
				Ext: xdr.LedgerEntryExtensionV1Ext{
					V: 1,
				},
			},
		},
		LastModifiedLedgerSeq: 30705278,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeClaimableBalance,
			ClaimableBalance: &xdr.ClaimableBalanceEntry{
				BalanceId: genericClaimableBalance,
				Claimants: []xdr.Claimant{
					{
						Type: 0,
						V0: &xdr.ClaimantV0{
							Destination: testAccount1ID,
						},
					},
				},
				Asset: xdr.Asset{
					Type: xdr.AssetTypeAssetTypeCreditAlphanum12,
					AlphaNum12: &xdr.AlphaNum12{
						AssetCode: xdr.AssetCode12{1, 2, 3, 4, 5, 6, 7, 8, 9},
						Issuer:    testAccount3ID,
					},
				},
				Amount: 9990000000,
				Ext: xdr.ClaimableBalanceEntryExt{
					V: 1,
					V1: &xdr.ClaimableBalanceEntryExtensionV1{
						Ext: xdr.ClaimableBalanceEntryExtensionV1Ext{
							V: 1,
						},
						Flags: 10,
					},
				},
			},
		},
	}
	return ingest.Change{
		Type: xdr.LedgerEntryTypeClaimableBalance,
		Pre:  &ledgerEntry,
		Post: nil,
	}
}

func makeClaimableBalanceTestOutput() ClaimableBalanceOutput {
	return ClaimableBalanceOutput{
		BalanceID: "000000000102030405060708090000000000000000000000000000000000000000000000",
		Claimants: []Claimant{
			{
				Destination: "GCEODJVUUVYVFD5KT4TOEDTMXQ76OPFOQC2EMYYMLPXQCUVPOB6XRWPQ",
				Predicate: xdr.ClaimPredicate{
					Type: xdr.ClaimPredicateTypeClaimPredicateUnconditional,
				},
			},
		},
		AssetIssuer:        "GBT4YAEGJQ5YSFUMNKX6BPBUOCPNAIOFAVZOF6MIME2CECBMEIUXFZZN",
		AssetType:          "credit_alphanum12",
		AssetCode:          "\x01\x02\x03\x04\x05\x06\a\b\t",
		AssetAmount:        999,
		AssetID:            -4023078858747574648,
		Sponsor:            null.StringFrom("GAAQEAYEAUDAOCAJAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABO3W"),
		Flags:              10,
		LastModifiedLedger: 30705278,
		LedgerEntryChange:  2,
		Deleted:            true,
		LedgerSequence:     10,
		ClosedAt:           time.Date(1970, time.January, 1, 0, 16, 40, 0, time.UTC),
	}
}
