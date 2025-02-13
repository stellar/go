package ingest

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/stellar/go/xdr"
)

func TestChangeCompactorExistingCreated(t *testing.T) {
	suite.Run(t, new(TestChangeCompactorExistingCreatedSuite))
}

// TestChangeCompactorExistingCreatedSuite tests transitions from
// existing CREATED state in the cache.
type TestChangeCompactorExistingCreatedSuite struct {
	suite.Suite
	cache *ChangeCompactor
}

func (s *TestChangeCompactorExistingCreatedSuite) SetupTest() {
	s.cache = NewChangeCompactor(NewChangeCompactorDefaultConfig())

	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 11,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
	}
	s.Assert().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()
	s.Assert().Len(changes, 1)
	s.Assert().Equal(changes[0].ChangeType, xdr.LedgerEntryChangeTypeLedgerEntryCreated)
}

func (s *TestChangeCompactorExistingCreatedSuite) TestChangeCreated() {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 12,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
	}
	s.Assert().EqualError(
		s.cache.AddChange(change),
		"can't create an entry that already exists (ledger key = AAAAAAAAAAC2LgFRDBZ3J52nLm30kq2iMgrO7dYzYAN3hvjtf1IHWg==)",
	)
}

func (s *TestChangeCompactorExistingCreatedSuite) TestChangeUpdated() {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 11,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 12,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
	}
	s.Assert().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()
	s.Assert().Len(changes, 1)
	s.Assert().Equal(changes[0].ChangeType, xdr.LedgerEntryChangeTypeLedgerEntryCreated)
}

func (s *TestChangeCompactorExistingCreatedSuite) TestChangeRemoved() {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 11,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		Post:       nil,
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
	}
	s.Assert().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()
	s.Assert().Len(changes, 0)
}

func (s *TestChangeCompactorExistingCreatedSuite) TestChangeRestored() {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 11,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryRestored,
	}
	s.Assert().EqualError(
		s.cache.AddChange(change),
		"can't restore an entry that is already active (ledger key = AAAAAAAAAAC2LgFRDBZ3J52nLm30kq2iMgrO7dYzYAN3hvjtf1IHWg==)",
	)
}

func TestLedgerEntryChangeCacheExistingUpdated(t *testing.T) {
	suite.Run(t, new(TestChangeCompactorExistingUpdatedSuite))
}

// TestChangeCompactorExistingUpdatedSuite tests transitions from existing
// UPDATED state in the cache.
type TestChangeCompactorExistingUpdatedSuite struct {
	suite.Suite
	cache *ChangeCompactor
}

func (s *TestChangeCompactorExistingUpdatedSuite) SetupTest() {
	s.cache = NewChangeCompactor(NewChangeCompactorDefaultConfig())

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
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
	}
	s.Assert().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()
	s.Assert().Len(changes, 1)
	s.Assert().Equal(changes[0].ChangeType, xdr.LedgerEntryChangeTypeLedgerEntryUpdated)
}

func (s *TestChangeCompactorExistingUpdatedSuite) TestChangeCreated() {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 12,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
	}
	s.Assert().EqualError(
		s.cache.AddChange(change),
		"can't create an entry that already exists (ledger key = AAAAAAAAAAC2LgFRDBZ3J52nLm30kq2iMgrO7dYzYAN3hvjtf1IHWg==)",
	)
}

func (s *TestChangeCompactorExistingUpdatedSuite) TestChangeUpdated() {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 11,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 12,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
	}
	s.Assert().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()
	s.Assert().Len(changes, 1)
	s.Assert().Equal(changes[0].ChangeType, xdr.LedgerEntryChangeTypeLedgerEntryUpdated)
	s.Assert().Equal(changes[0].Post.LastModifiedLedgerSeq, xdr.Uint32(12))
}

func (s *TestChangeCompactorExistingUpdatedSuite) TestChangeRemoved() {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 11,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		Post:       nil,
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
	}
	s.Assert().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()
	s.Assert().Len(changes, 1)
	s.Assert().Equal(changes[0].ChangeType, xdr.LedgerEntryChangeTypeLedgerEntryRemoved)
}

func (s *TestChangeCompactorExistingUpdatedSuite) TestChangeRestored() {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 11,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryRestored,
	}
	s.Assert().EqualError(
		s.cache.AddChange(change),
		"can't restore an entry that is already active (ledger key = AAAAAAAAAAC2LgFRDBZ3J52nLm30kq2iMgrO7dYzYAN3hvjtf1IHWg==)",
	)
}

func TestChangeCompactorExistingRemoved(t *testing.T) {
	suite.Run(t, new(TestChangeCompactorExistingRemovedSuite))
}

// TestChangeCompactorExistingRemovedSuite tests transitions from existing
// REMOVED state in the cache.
type TestChangeCompactorExistingRemovedSuite struct {
	suite.Suite
	cache *ChangeCompactor
}

func (s *TestChangeCompactorExistingRemovedSuite) SetupTest() {
	s.cache = NewChangeCompactor(NewChangeCompactorDefaultConfig())

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
		Post:       nil,
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
	}
	s.Assert().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()
	s.Assert().Len(changes, 1)
	s.Assert().Equal(changes[0].ChangeType, xdr.LedgerEntryChangeTypeLedgerEntryRemoved)
}

func (s *TestChangeCompactorExistingRemovedSuite) TestChangeCreated() {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 12,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
	}
	s.Assert().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()
	s.Assert().Len(changes, 1)
	s.Assert().Equal(changes[0].ChangeType, xdr.LedgerEntryChangeTypeLedgerEntryUpdated)
	s.Assert().Equal(changes[0].Post.LastModifiedLedgerSeq, xdr.Uint32(12))
}

func (s *TestChangeCompactorExistingRemovedSuite) TestChangeUpdated() {
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
			LastModifiedLedgerSeq: 12,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
	}
	s.Assert().EqualError(
		s.cache.AddChange(change),
		"can't update an entry that was previously removed (ledger key = AAAAAAAAAAC2LgFRDBZ3J52nLm30kq2iMgrO7dYzYAN3hvjtf1IHWg==)",
	)
}

func (s *TestChangeCompactorExistingRemovedSuite) TestChangeRemoved() {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 11,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		Post:       nil,
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
	}
	s.Assert().EqualError(
		s.cache.AddChange(change),
		"can't remove an entry that was previously removed (ledger key = AAAAAAAAAAC2LgFRDBZ3J52nLm30kq2iMgrO7dYzYAN3hvjtf1IHWg==)",
	)
}

func (s *TestChangeCompactorExistingRemovedSuite) TestChangeRestored() {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 11,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryRestored,
	}
	s.Assert().EqualError(
		s.cache.AddChange(change),
		"can't restore an entry that is already active (ledger key = AAAAAAAAAAC2LgFRDBZ3J52nLm30kq2iMgrO7dYzYAN3hvjtf1IHWg==)",
	)

}

func TestChangeCompactorExistingRestored(t *testing.T) {
	for _, emitRemoved := range []bool{true, false} {
		s := new(TestChangeCompactorExistingRestoredSuite)
		s.emitArchivedEntryRemovedChange = emitRemoved
		suite.Run(t, s)
	}
}

// TestChangeCompactorExistingRestoredSuite tests transitions from existing
// RESTORED state in the cache.
type TestChangeCompactorExistingRestoredSuite struct {
	suite.Suite
	cache                          *ChangeCompactor
	contractDataEntry              xdr.LedgerEntry
	emitArchivedEntryRemovedChange bool
}

func (s *TestChangeCompactorExistingRestoredSuite) SetupTest() {
	s.cache = NewChangeCompactor(&ChangeCompactorConfig{EmitArchivedEntryRemovedChange: s.emitArchivedEntryRemovedChange})
	val := true
	s.contractDataEntry = xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeContractData,
			ContractData: &xdr.ContractDataEntry{
				Contract: xdr.ScAddress{
					Type:       xdr.ScAddressTypeScAddressTypeContract,
					ContractId: &xdr.Hash{0xca, 0xfe},
				},
				Key:        xdr.ScVal{Type: xdr.ScValTypeScvBool, B: &val},
				Durability: xdr.ContractDataDurabilityPersistent,
				Val:        xdr.ScVal{Type: xdr.ScValTypeScvBool, B: &val},
			},
		},
	}

	change := Change{
		Type:       xdr.LedgerEntryTypeContractData,
		Post:       &s.contractDataEntry,
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryRestored,
	}

	s.Require().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()
	s.Assert().Len(changes, 1)
	s.Assert().Equal(xdr.LedgerEntryChangeTypeLedgerEntryRestored, changes[0].ChangeType)
	s.Assert().EqualValues(&s.contractDataEntry, changes[0].Post)
}

func (s *TestChangeCompactorExistingRestoredSuite) getLedgerKeyString(entry *xdr.LedgerEntry) string {
	lk, err := entry.LedgerKey()
	s.Require().NoError(err)
	ledgerKey, err := xdr.NewEncodingBuffer().UnsafeMarshalBinary(lk)
	s.Require().NoError(err)
	return base64.StdEncoding.EncodeToString(ledgerKey)
}

func (s *TestChangeCompactorExistingRestoredSuite) TestChangeCreated() {
	change := Change{
		Type:       xdr.LedgerEntryTypeContractData,
		Pre:        nil,
		Post:       &s.contractDataEntry,
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
	}
	s.Assert().EqualError(
		s.cache.AddChange(change),
		fmt.Sprintf("can't create an entry that already exists (ledger key = %s)",
			s.getLedgerKeyString(&s.contractDataEntry),
		),
	)
}

func (s *TestChangeCompactorExistingRestoredSuite) TestChangeUpdated() {
	modified := s.contractDataEntry
	modified.LastModifiedLedgerSeq = 2
	change := Change{
		Type:       xdr.LedgerEntryTypeContractData,
		Pre:        &s.contractDataEntry,
		Post:       &modified,
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
	}

	s.Assert().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()
	s.Assert().Len(changes, 1)
	s.Assert().Equal(xdr.LedgerEntryChangeTypeLedgerEntryRestored, changes[0].ChangeType)
	s.Assert().EqualValues(&modified, changes[0].Post)
}

func (s *TestChangeCompactorExistingRestoredSuite) TestChangeRemoved() {
	change := Change{
		Type:       xdr.LedgerEntryTypeContractData,
		Pre:        &s.contractDataEntry,
		Post:       nil,
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
	}

	s.Assert().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()

	if s.cache.config.EmitArchivedEntryRemovedChange {
		s.Assert().Len(changes, 1)
		s.Assert().Equal(xdr.LedgerEntryChangeTypeLedgerEntryRemoved, changes[0].ChangeType)
		s.Assert().EqualValues(&s.contractDataEntry, changes[0].Pre)
	} else {
		s.Assert().Len(changes, 0)
	}
}

func (s *TestChangeCompactorExistingRestoredSuite) TestChangeRestored() {
	change := Change{
		Type:       xdr.LedgerEntryTypeContractData,
		Pre:        nil,
		Post:       &s.contractDataEntry,
		ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryRestored,
	}
	s.Assert().EqualError(
		s.cache.AddChange(change),
		fmt.Sprintf("can't restore an entry that is already active (ledger key = %s)",
			s.getLedgerKeyString(&s.contractDataEntry),
		),
	)
}

// TestChangeCompactorSquashMultiplePayments simulates sending multiple payments
// between two accounts. Ledger cache should squash multiple changes into just
// two.
//
// GAJ2T6NQ6TDZRVRSNWM3JC7L3TG4H7UBCVK3GUHKP3TQ5NQ3LM4JGBTJ sends money
// GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML receives money
func TestChangeCompactorSquashMultiplePayments(t *testing.T) {
	cache := NewChangeCompactor(NewChangeCompactorDefaultConfig())

	for i := 1; i <= 1000; i++ {
		change := Change{
			Type: xdr.LedgerEntryTypeAccount,
			Pre: &xdr.LedgerEntry{
				LastModifiedLedgerSeq: 10,
				Data: xdr.LedgerEntryData{
					Type: xdr.LedgerEntryTypeAccount,
					Account: &xdr.AccountEntry{
						AccountId: xdr.MustAddress("GAJ2T6NQ6TDZRVRSNWM3JC7L3TG4H7UBCVK3GUHKP3TQ5NQ3LM4JGBTJ"),
						Balance:   xdr.Int64(2000 - i + 1),
					},
				},
			},
			Post: &xdr.LedgerEntry{
				LastModifiedLedgerSeq: 12,
				Data: xdr.LedgerEntryData{
					Type: xdr.LedgerEntryTypeAccount,
					Account: &xdr.AccountEntry{
						AccountId: xdr.MustAddress("GAJ2T6NQ6TDZRVRSNWM3JC7L3TG4H7UBCVK3GUHKP3TQ5NQ3LM4JGBTJ"),
						Balance:   xdr.Int64(2000 - i),
					},
				},
			},
			ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
		}
		assert.NoError(t, cache.AddChange(change))

		change = Change{
			Type: xdr.LedgerEntryTypeAccount,
			Pre: &xdr.LedgerEntry{
				LastModifiedLedgerSeq: 10,
				Data: xdr.LedgerEntryData{
					Type: xdr.LedgerEntryTypeAccount,
					Account: &xdr.AccountEntry{
						AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
						Balance:   xdr.Int64(2000 + i - 1),
					},
				},
			},
			Post: &xdr.LedgerEntry{
				LastModifiedLedgerSeq: 12,
				Data: xdr.LedgerEntryData{
					Type: xdr.LedgerEntryTypeAccount,
					Account: &xdr.AccountEntry{
						AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
						Balance:   xdr.Int64(2000 + i),
					},
				},
			},
			ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
		}
		assert.NoError(t, cache.AddChange(change))
	}

	changes := cache.GetChanges()
	assert.Len(t, changes, 2)
	for _, change := range changes {
		assert.Equal(t, change.ChangeType, xdr.LedgerEntryChangeTypeLedgerEntryUpdated)
		account := change.Post.Data.MustAccount()
		switch account.AccountId.Address() {
		case "GAJ2T6NQ6TDZRVRSNWM3JC7L3TG4H7UBCVK3GUHKP3TQ5NQ3LM4JGBTJ":
			assert.Equal(t, account.Balance, xdr.Int64(1000))
		case "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML":
			assert.Equal(t, account.Balance, xdr.Int64(3000))
		default:
			assert.Fail(t, "Invalid account")
		}
	}
}
