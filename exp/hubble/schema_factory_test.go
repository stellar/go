// +build go1.13

package hubble

import (
	"fmt"
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

const wantAddress = "GBFLTCDLOE6YQ74B66RH3S2UW5I2MKZ5VLTM75F4YMIWUIXRIFVNRNIF"

var accountIDTests = []struct {
	testName      string
	state         *accountState
	wantAccountID string
	change        xdr.LedgerEntryChange
}{
	{"FromState",
		&accountState{address: wantAddress},
		wantAddress,
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type: xdr.LedgerEntryTypeAccount,
				},
			},
		},
	},
	{"FromChange",
		nil,
		wantAddress,
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type: xdr.LedgerEntryTypeAccount,
					Account: &xdr.AccountEntry{
						AccountId: xdr.MustAddress(wantAddress),
					},
				},
			},
		},
	},
}

func TestMakeAccount(t *testing.T) {
	for _, tt := range accountIDTests {
		t.Run(tt.testName, func(t *testing.T) {
			var (
				gotAccountID string
				err          error
			)
			if tt.state == nil {
				gotAccountID, err = makeAccountID(&tt.change)
			} else {
				gotAccountID, err = makeAccountID(&tt.change, *tt.state)
			}
			if err != nil {
				t.Error(err)
			}
			if !assert.Equal(t, tt.wantAccountID, gotAccountID) {
				t.Fatalf("got account id %s, want account id %s", tt.wantAccountID, gotAccountID)
			}
		})
	}
}

var seqnumTests = []struct {
	testName   string
	wantSeqnum uint32
	change     xdr.LedgerEntryChange
}{
	{"Changed",
		2947523,
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				LastModifiedLedgerSeq: xdr.Uint32(2947523),
				Data: xdr.LedgerEntryData{
					Type:    xdr.LedgerEntryTypeAccount,
					Account: &xdr.AccountEntry{},
				},
			},
		},
	},
	{"Removed",
		0,
		xdr.LedgerEntryChange{
			Type:  xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
			State: &xdr.LedgerEntry{},
		},
	},
}

func TestMakeSeqnum(t *testing.T) {
	for _, tt := range seqnumTests {
		t.Run(string(tt.testName), func(t *testing.T) {
			state := accountState{seqnum: 11}
			gotSeqnum, err := makeSeqnum(&state, &tt.change)
			if err != nil {
				t.Error(err)
			}
			if !assert.Equal(t, tt.wantSeqnum, gotSeqnum) {
				t.Fatalf("got seqnum %d, want seqnum %d", gotSeqnum, tt.wantSeqnum)
			}
		})
	}
}

var accountEntryTests = []struct {
	testName  string
	wantEntry *xdr.AccountEntry
	change    xdr.LedgerEntryChange
}{
	{"NotAccount",
		nil,
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type: xdr.LedgerEntryTypeData,
					Data: &xdr.DataEntry{
						AccountId: xdr.MustAddress("GBFLTCDLOE6YQ74B66RH3S2UW5I2MKZ5VLTM75F4YMIWUIXRIFVNRNIF"),
						DataName:  xdr.String64("name"),
						DataValue: xdr.DataValue([]byte("value")),
					},
				},
			},
		},
	},
	{"Removed",
		nil,
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
			Removed: &xdr.LedgerKey{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.LedgerKeyAccount{
					AccountId: xdr.MustAddress("GBFLTCDLOE6YQ74B66RH3S2UW5I2MKZ5VLTM75F4YMIWUIXRIFVNRNIF"),
				},
			},
		},
	},
	{"NotRemoved",
		&xdr.AccountEntry{AccountId: xdr.MustAddress(wantAddress)},
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type: xdr.LedgerEntryTypeAccount,
					Account: &xdr.AccountEntry{
						AccountId: xdr.MustAddress(wantAddress),
					},
				},
			},
		},
	},
}

func TestGetAccountEntry(t *testing.T) {
	for _, tt := range accountEntryTests {
		t.Run(string(tt.testName), func(t *testing.T) {
			gotEntry, err := getAccountEntry(&tt.change)
			if err != nil {
				t.Error(err)
			}
			if tt.wantEntry == nil {
				if gotEntry != nil {
					t.Fatal("got account entry non-nil, want account entry nil")
				}
			} else {
				gotAddress := gotEntry.AccountId.Address()
				wantAddress := tt.wantEntry.AccountId.Address()
				if !assert.Equal(t, wantAddress, gotAddress) {
					t.Fatalf("got address %s, want address %s", gotAddress, wantAddress)
				}
			}
		})
	}
}

var balanceTests = []struct {
	testName    string
	wantBalance uint32
	change      xdr.LedgerEntryChange
}{
	{"NotChanged",
		uint32(111),
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
			Removed: &xdr.LedgerKey{
				Type:    xdr.LedgerEntryTypeAccount,
				Account: &xdr.LedgerKeyAccount{},
			},
		},
	},
	{"Changed",
		uint32(222),
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type: xdr.LedgerEntryTypeAccount,
					Account: &xdr.AccountEntry{
						Balance: xdr.Int64(222),
					},
				},
			},
		},
	},
}

func TestMakeBalance(t *testing.T) {
	for _, tt := range balanceTests {
		t.Run(tt.testName, func(t *testing.T) {
			state := accountState{balance: uint32(111)}
			gotBalance, err := makeBalance(&state, &tt.change)
			if err != nil {
				t.Error(err)
			}
			if !assert.Equal(t, tt.wantBalance, gotBalance) {
				t.Fatalf("got balance %d, want balance %d", gotBalance, tt.wantBalance)
			}
		})
	}
}

var signersTests = []struct {
	testName    string
	wantSigners []signer
	change      xdr.LedgerEntryChange
}{
	{"NotAccount",
		[]signer{{address: wantAddress, weight: uint32(1)}},
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
			Removed: &xdr.LedgerKey{
				Type:    xdr.LedgerEntryTypeAccount,
				Account: &xdr.LedgerKeyAccount{},
			},
		}},
	{"NotChanged",
		[]signer{{address: wantAddress, weight: uint32(1)}},
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type: xdr.LedgerEntryTypeAccount,
					Account: &xdr.AccountEntry{
						Signers: []xdr.Signer{{
							Key:    xdr.MustSigner(wantAddress),
							Weight: xdr.Uint32(1),
						}},
					},
				},
			},
		},
	},
}

func TestSigners(t *testing.T) {
	for _, tt := range signersTests {
		t.Run(tt.testName, func(t *testing.T) {
			originalSigners := []signer{{address: wantAddress, weight: uint32(1)}}
			state := accountState{signers: originalSigners}
			gotSigners, err := makeSigners(&state, &tt.change)
			if err != nil {
				t.Error(err)
			}
			if tt.wantSigners == nil {
				if gotSigners != nil {
					t.Fatalf("got signers %v, want signers nil", gotSigners)
				}
			} else {
				if !assert.Equal(t, gotSigners, tt.wantSigners) {
					t.Fatalf("got signers %v, want signers %v", gotSigners, tt.wantSigners)
				}
			}
		})
	}
}

var trustlineKey = fmt.Sprintf("credit_alphanum4/USD/%s", wantAddress)

var trustlinesTests = []struct {
	testName       string
	wantTrustlines map[string]trustline
	change         xdr.LedgerEntryChange
}{
	{"NotTrustline",
		map[string]trustline{
			trustlineKey: trustline{
				asset:   trustlineKey,
				balance: uint32(0),
				limit:   uint32(100),
			}},
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type:    xdr.LedgerEntryTypeAccount,
					Account: &xdr.AccountEntry{},
				},
			},
		},
	},
	{"Removed",
		map[string]trustline{},
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
			Removed: &xdr.LedgerKey{
				Type: xdr.LedgerEntryTypeTrustline,
				TrustLine: &xdr.LedgerKeyTrustLine{
					AccountId: xdr.MustAddress(wantAddress),
					Asset:     xdr.MustNewCreditAsset("USD", wantAddress),
				},
			},
		},
	},
	{"Changed",
		map[string]trustline{
			trustlineKey: trustline{
				asset:   trustlineKey,
				balance: uint32(20),
				limit:   uint32(100),
			}},
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type: xdr.LedgerEntryTypeTrustline,
					TrustLine: &xdr.TrustLineEntry{
						AccountId: xdr.MustAddress(wantAddress),
						Asset:     xdr.MustNewCreditAsset("USD", wantAddress),
						Balance:   xdr.Int64(20),
						Limit:     xdr.Int64(100),
					},
				},
			},
		},
	},
}

func TestTrustlines(t *testing.T) {
	for _, tt := range trustlinesTests {
		t.Run(tt.testName, func(t *testing.T) {
			originalTrustlines := map[string]trustline{
				trustlineKey: trustline{
					asset:   trustlineKey,
					balance: uint32(0),
					limit:   uint32(100),
				},
			}
			state := accountState{
				trustlines: originalTrustlines,
			}
			gotTrustlines, err := makeTrustlines(&state, &tt.change)
			if err != nil {
				t.Error(err)
			}
			if !assert.Equal(t, tt.wantTrustlines, gotTrustlines) {
				t.Fatalf("got trustlines %v, want trustlines %v", gotTrustlines, tt.wantTrustlines)
			}
		})
	}
}

var originalData = map[string][]byte{
	"key": []byte("value"),
}
var dataTests = []struct {
	testName string
	wantData map[string][]byte
	change   xdr.LedgerEntryChange
}{
	{"NotData",
		originalData,
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type:    xdr.LedgerEntryTypeAccount,
					Account: &xdr.AccountEntry{},
				},
			},
		},
	},
	{"Removed",
		map[string][]byte{},
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
			Removed: &xdr.LedgerKey{
				Type: xdr.LedgerEntryTypeData,
				Data: &xdr.LedgerKeyData{
					DataName: xdr.String64("key"),
				},
			},
		},
	},
	{"Changed",
		map[string][]byte{"key": []byte("newValue")},
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type: xdr.LedgerEntryTypeData,
					Data: &xdr.DataEntry{
						DataName:  xdr.String64("key"),
						DataValue: xdr.DataValue("newValue"),
					},
				},
			},
		},
	},
}

func TestData(t *testing.T) {
	for _, tt := range dataTests {
		t.Run(tt.testName, func(t *testing.T) {

			state := accountState{data: originalData}
			gotData, err := makeData(&state, &tt.change)
			if err != nil {
				t.Error(err)
			}
			if !assert.Equal(t, tt.wantData, gotData) {
				t.Fatalf("got data %v, want data %v", gotData, tt.wantData)
			}
		})
	}
}

var originalOffers = map[uint32]offer{1: offer{id: 1}}
var offerTests = []struct {
	testName   string
	wantOffers map[uint32]offer
	change     xdr.LedgerEntryChange
}{
	{"NotOffers",
		originalOffers,
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type:    xdr.LedgerEntryTypeAccount,
					Account: &xdr.AccountEntry{},
				},
			},
		},
	},
	{"Removed",
		map[uint32]offer{},
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
			Removed: &xdr.LedgerKey{
				Type: xdr.LedgerEntryTypeOffer,
				Offer: &xdr.LedgerKeyOffer{
					OfferId: xdr.Int64(1),
				},
			},
		},
	},
	{"Changed",
		map[uint32]offer{1: offer{id: 1}, 2: offer{id: 2, seller: wantAddress, selling: "native", buying: "native"}},
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type: xdr.LedgerEntryTypeOffer,
					Offer: &xdr.OfferEntry{
						OfferId:  xdr.Int64(2),
						SellerId: xdr.MustAddress(wantAddress),
					},
				},
			},
		},
	},
}

func TestOffers(t *testing.T) {
	for _, tt := range offerTests {
		t.Run(tt.testName, func(t *testing.T) {
			originalOffers := map[uint32]offer{
				1: offer{id: 1},
			}
			state := accountState{offers: originalOffers}
			gotOffers, err := makeOffers(&state, &tt.change)
			if err != nil {
				t.Error(err)
			}
			if !assert.Equal(t, tt.wantOffers, gotOffers) {
				t.Fatalf("got offers %v, want offers %v", gotOffers, tt.wantOffers)
			}
		})
	}
}
