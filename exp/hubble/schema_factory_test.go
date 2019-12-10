// +build go1.13

package hubble

import (
	"fmt"
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

const wantAddress = "GBFLTCDLOE6YQ74B66RH3S2UW5I2MKZ5VLTM75F4YMIWUIXRIFVNRNIF"

func TestMakeNewAccountStateSuccess(t *testing.T) {
	var trustlineKey = fmt.Sprintf("credit_alphanum4/USD/%s", wantAddress)
	var tests = []struct {
		name      string
		state     *accountState
		change    *xdr.LedgerEntryChange
		wantState *accountState
	}{
		{"AccountRemoved",
			&accountState{address: wantAddress},
			&xdr.LedgerEntryChange{
				Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
				Removed: &xdr.LedgerKey{
					Account: &xdr.LedgerKeyAccount{
						AccountId: xdr.MustAddress(wantAddress),
					},
				},
			},
			nil,
		},
		{"SeqnumNotChanged",
			&accountState{seqnum: 11},
			&xdr.LedgerEntryChange{
				Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
				State: &xdr.LedgerEntry{
					LastModifiedLedgerSeq: xdr.Uint32(11),
					Data: xdr.LedgerEntryData{
						Type:    xdr.LedgerEntryTypeAccount,
						Account: &xdr.AccountEntry{},
					},
				},
			},
			&accountState{seqnum: 11},
		},
		{"SeqnumChanged",
			&accountState{seqnum: 11},
			&xdr.LedgerEntryChange{
				Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
				State: &xdr.LedgerEntry{
					LastModifiedLedgerSeq: xdr.Uint32(2947523),
					Data: xdr.LedgerEntryData{
						Type:    xdr.LedgerEntryTypeAccount,
						Account: &xdr.AccountEntry{},
					},
				},
			},
			&accountState{seqnum: 2947523},
		},
		{"BalanceChanged",
			&accountState{balance: 222},
			makeLedgerEntryChangeAccount(&xdr.AccountEntry{Balance: xdr.Int64(111)}),
			&accountState{balance: 111},
		},
		{"SignersNotChanged",
			&accountState{signers: []signer{{address: wantAddress, weight: uint32(1)}}},
			makeLedgerEntryChangeAccount(&xdr.AccountEntry{
				Signers: []xdr.Signer{{
					Key:    xdr.MustSigner(wantAddress),
					Weight: xdr.Uint32(1),
				}},
			}),
			&accountState{signers: []signer{{address: wantAddress, weight: uint32(1)}}},
		},
		{"SignersChanged",
			&accountState{signers: []signer{{address: wantAddress, weight: uint32(1)}}},
			makeLedgerEntryChangeAccount(&xdr.AccountEntry{
				Signers: []xdr.Signer{{
					Key:    xdr.MustSigner(wantAddress),
					Weight: xdr.Uint32(2),
				}},
			}),
			&accountState{signers: []signer{{address: wantAddress, weight: uint32(2)}}},
		},
		{"TrustlinesRemoved",
			&accountState{trustlines: map[string]trustline{
				trustlineKey: trustline{asset: trustlineKey, balance: uint32(0), limit: uint32(100)}},
			},
			&xdr.LedgerEntryChange{
				Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
				Removed: &xdr.LedgerKey{
					Type: xdr.LedgerEntryTypeTrustline,
					TrustLine: &xdr.LedgerKeyTrustLine{
						AccountId: xdr.MustAddress(wantAddress),
						Asset:     xdr.MustNewCreditAsset("USD", wantAddress),
					},
				},
			},
			&accountState{trustlines: map[string]trustline{}},
		},
		{"TrustlinesChanged",
			&accountState{trustlines: map[string]trustline{
				trustlineKey: trustline{asset: trustlineKey, balance: uint32(10), limit: uint32(100)}},
			},
			makeLedgerEntryChangeTrustline(wantAddress, "USD", 20, 100),
			&accountState{trustlines: map[string]trustline{
				trustlineKey: trustline{asset: trustlineKey, balance: uint32(20), limit: uint32(100)}},
			},
		},
		{"DataRemoved",
			&accountState{data: map[string][]byte{"key": []byte("value")}},
			&xdr.LedgerEntryChange{
				Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
				Removed: &xdr.LedgerKey{
					Type: xdr.LedgerEntryTypeData,
					Data: &xdr.LedgerKeyData{
						DataName: xdr.String64("key"),
					},
				},
			},
			&accountState{data: map[string][]byte{}},
		},
		{"DataChanged",
			&accountState{data: map[string][]byte{"key": []byte("old")}},
			makeLedgerEntryChangeData(wantAddress, "key", "new"),
			&accountState{data: map[string][]byte{"key": []byte("new")}},
		},
		{"OffersRemoved",
			&accountState{offers: map[uint32]offer{1: offer{id: 1}}},
			&xdr.LedgerEntryChange{
				Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
				Removed: &xdr.LedgerKey{
					Type: xdr.LedgerEntryTypeOffer,
					Offer: &xdr.LedgerKeyOffer{
						OfferId: xdr.Int64(1),
					},
				},
			},
			&accountState{offers: map[uint32]offer{}},
		},
		{"OffersChanged",
			&accountState{offers: map[uint32]offer{1: offer{id: 1}}},
			makeLedgerEntryChangeOffer(2, wantAddress),
			&accountState{offers: map[uint32]offer{
				1: offer{id: 1}, 2: offer{id: 2, seller: wantAddress, selling: "native", buying: "native"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotState, err := makeNewAccountState(tt.state, tt.change)
			if !assert.NoError(t, err) {
				return
			}
			if !assert.Equal(t, tt.wantState, gotState) {
				return
			}
		})
	}
}

// TODO: Add tests for error cases.
