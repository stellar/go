package io

import (
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestChangeAccountChangedExceptSignersInvalidType(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeOffer,
	}

	assert.Panics(t, func() {
		change.AccountChangedExceptSigners()
	})
}

func TestChangeAccountChangedExceptSignersLastModifiedLedgerSeq(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 11,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
	}
	changed, err := change.AccountChangedExceptSigners()
	assert.NoError(t, err)
	assert.True(t, changed)
}

func TestChangeAccountChangedExceptSignersNoPre(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
	}
	changed, err := change.AccountChangedExceptSigners()
	assert.NoError(t, err)
	assert.True(t, changed)
}

func TestChangeAccountChangedExceptSignersNoPost(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		Post: nil,
	}
	changed, err := change.AccountChangedExceptSigners()
	assert.NoError(t, err)
	assert.True(t, changed)
}

func TestChangeAccountChangedExceptSignersMasterKeyRemoved(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					// Master weight = 1
					Thresholds: [4]byte{1, 1, 1, 1},
				},
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					// Master weight = 0
					Thresholds: [4]byte{0, 1, 1, 1},
				},
			},
		},
	}

	changed, err := change.AccountChangedExceptSigners()
	assert.NoError(t, err)
	assert.True(t, changed)
}

func TestChangeAccountChangedExceptSignersSignerChange(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					Signers: []xdr.Signer{
						xdr.Signer{
							Key:    xdr.MustSigner("GCCCU34WDY2RATQTOOQKY6SZWU6J5DONY42SWGW2CIXGW4LICAGNRZKX"),
							Weight: 1,
						},
					},
				},
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					Signers: []xdr.Signer{
						xdr.Signer{
							Key:    xdr.MustSigner("GCCCU34WDY2RATQTOOQKY6SZWU6J5DONY42SWGW2CIXGW4LICAGNRZKX"),
							Weight: 2,
						},
					},
				},
			},
		},
	}

	changed, err := change.AccountChangedExceptSigners()
	assert.NoError(t, err)
	assert.False(t, changed)
}

func TestChangeAccountChangedExceptSignersNoChanges(t *testing.T) {
	inflationDest := xdr.MustAddress("GBAH2GBLJB54JAROJ3FVO4ZTTJJI3XKOBTMJOZFUJ3UHYIVNJTLJUYFY")
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId:     xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					Balance:       1000,
					SeqNum:        432894732,
					NumSubEntries: 2,
					InflationDest: &inflationDest,
					Flags:         4,
					HomeDomain:    "stellar.org",
					Thresholds:    [4]byte{1, 1, 1, 1},
					Signers: []xdr.Signer{
						xdr.Signer{
							Key:    xdr.MustSigner("GCCCU34WDY2RATQTOOQKY6SZWU6J5DONY42SWGW2CIXGW4LICAGNRZKX"),
							Weight: 1,
						},
					},
					Ext: xdr.AccountEntryExt{
						V: 1,
						V1: &xdr.AccountEntryV1{
							Liabilities: xdr.Liabilities{
								Buying:  10,
								Selling: 20,
							},
						},
					},
				},
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId:     xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					Balance:       1000,
					SeqNum:        432894732,
					NumSubEntries: 2,
					InflationDest: &inflationDest,
					Flags:         4,
					HomeDomain:    "stellar.org",
					Thresholds:    [4]byte{1, 1, 1, 1},
					Signers: []xdr.Signer{
						xdr.Signer{
							Key:    xdr.MustSigner("GCCCU34WDY2RATQTOOQKY6SZWU6J5DONY42SWGW2CIXGW4LICAGNRZKX"),
							Weight: 1,
						},
					},
					Ext: xdr.AccountEntryExt{
						V: 1,
						V1: &xdr.AccountEntryV1{
							Liabilities: xdr.Liabilities{
								Buying:  10,
								Selling: 20,
							},
						},
					},
				},
			},
		},
	}

	changed, err := change.AccountChangedExceptSigners()
	assert.NoError(t, err)
	assert.False(t, changed)

	// Make sure pre and post not modified
	assert.NotNil(t, change.Pre.Data.Account.Signers)
	assert.Len(t, change.Pre.Data.Account.Signers, 1)

	assert.NotNil(t, change.Post.Data.Account.Signers)
	assert.Len(t, change.Post.Data.Account.Signers, 1)
}

func TestChangeAccountSignersChangedInvalidType(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeOffer,
	}

	assert.Panics(t, func() {
		change.AccountSignersChanged()
	})
}

func TestChangeAccountSignersChangedNoPre(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
	}

	assert.True(t, change.AccountSignersChanged())
}

func TestChangeAccountSignersChangedNoPostMasterKey(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					// Master weight = 1
					Thresholds: [4]byte{1, 1, 1, 1},
				},
			},
		},
		Post: nil,
	}

	assert.True(t, change.AccountSignersChanged())
}

func TestChangeAccountSignersChangedNoPostNoMasterKey(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					// Master weight = 0
					Thresholds: [4]byte{0, 1, 1, 1},
				},
			},
		},
		Post: nil,
	}

	// Account being merge can still have signers so they will be removed.
	assert.True(t, change.AccountSignersChanged())
}

func TestChangeAccountSignersChangedMasterKeyRemoved(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					// Master weight = 1
					Thresholds: [4]byte{1, 1, 1, 1},
				},
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					// Master weight = 0
					Thresholds: [4]byte{0, 1, 1, 1},
				},
			},
		},
	}

	assert.True(t, change.AccountSignersChanged())
}

func TestChangeAccountSignersChangedMasterKeyAdded(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					// Master weight = 0
					Thresholds: [4]byte{0, 1, 1, 1},
				},
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					// Master weight = 1
					Thresholds: [4]byte{1, 1, 1, 1},
				},
			},
		},
	}

	assert.True(t, change.AccountSignersChanged())
}

func TestChangeAccountSignersChangedSignerAdded(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					Signers:   []xdr.Signer{},
				},
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					Signers: []xdr.Signer{
						xdr.Signer{
							Key:    xdr.MustSigner("GCCCU34WDY2RATQTOOQKY6SZWU6J5DONY42SWGW2CIXGW4LICAGNRZKX"),
							Weight: 1,
						},
					},
				},
			},
		},
	}

	assert.True(t, change.AccountSignersChanged())
}

func TestChangeAccountSignersChangedSignerRemoved(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					Signers: []xdr.Signer{
						xdr.Signer{
							Key:    xdr.MustSigner("GCCCU34WDY2RATQTOOQKY6SZWU6J5DONY42SWGW2CIXGW4LICAGNRZKX"),
							Weight: 1,
						},
					},
				},
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					Signers:   []xdr.Signer{},
				},
			},
		},
	}

	assert.True(t, change.AccountSignersChanged())
}

func TestChangeAccountSignersChangedSignerWeightChanged(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					Signers: []xdr.Signer{
						xdr.Signer{
							Key:    xdr.MustSigner("GCCCU34WDY2RATQTOOQKY6SZWU6J5DONY42SWGW2CIXGW4LICAGNRZKX"),
							Weight: 1,
						},
					},
				},
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					Signers: []xdr.Signer{
						xdr.Signer{
							Key:    xdr.MustSigner("GCCCU34WDY2RATQTOOQKY6SZWU6J5DONY42SWGW2CIXGW4LICAGNRZKX"),
							Weight: 2,
						},
					},
				},
			},
		},
	}

	assert.True(t, change.AccountSignersChanged())
}
