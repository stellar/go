package io

import (
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestLedgerEntryChangeCacheExistingCreated(t *testing.T) {
	suite.Run(t, new(TestLedgerEntryChangeCacheExistingCreatedSuite))
}

// TestLedgerEntryChangeCacheExistingCreatedSuite tests transitions from
// existing CREATED state in the cache.
type TestLedgerEntryChangeCacheExistingCreatedSuite struct {
	suite.Suite
	cache *LedgerEntryChangeCache
}

func (s *TestLedgerEntryChangeCacheExistingCreatedSuite) SetupTest() {
	s.cache = NewLedgerEntryChangeCache()

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
	}
	s.Assert().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()
	s.Assert().Len(changes, 1)
	s.Assert().Equal(changes[0].LedgerEntryChangeType(), xdr.LedgerEntryChangeTypeLedgerEntryCreated)
}

func (s *TestLedgerEntryChangeCacheExistingCreatedSuite) TestChangeCreated() {
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
	}
	s.Assert().EqualError(
		s.cache.AddChange(change),
		"can't create an entry that already exists (ledger key = AAAAAAAAAAC2LgFRDBZ3J52nLm30kq2iMgrO7dYzYAN3hvjtf1IHWg==)",
	)
}

func (s *TestLedgerEntryChangeCacheExistingCreatedSuite) TestChangeUpdated() {
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
	}
	s.Assert().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()
	s.Assert().Len(changes, 1)
	s.Assert().Equal(changes[0].LedgerEntryChangeType(), xdr.LedgerEntryChangeTypeLedgerEntryCreated)
}

func (s *TestLedgerEntryChangeCacheExistingCreatedSuite) TestChangeRemoved() {
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
		Post: nil,
	}
	s.Assert().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()
	s.Assert().Len(changes, 0)
}

func TestLedgerEntryChangeCacheExistingUpdated(t *testing.T) {
	suite.Run(t, new(TestLedgerEntryChangeCacheExistingUpdatedSuite))
}

// TestLedgerEntryChangeCacheExistingUpdatedSuite tests transitions from
// existing UPDATED state in the cache.
type TestLedgerEntryChangeCacheExistingUpdatedSuite struct {
	suite.Suite
	cache *LedgerEntryChangeCache
}

func (s *TestLedgerEntryChangeCacheExistingUpdatedSuite) SetupTest() {
	s.cache = NewLedgerEntryChangeCache()

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
	s.Assert().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()
	s.Assert().Len(changes, 1)
	s.Assert().Equal(changes[0].LedgerEntryChangeType(), xdr.LedgerEntryChangeTypeLedgerEntryUpdated)
}

func (s *TestLedgerEntryChangeCacheExistingUpdatedSuite) TestChangeCreated() {
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
	}
	s.Assert().EqualError(
		s.cache.AddChange(change),
		"can't create an entry that already exists (ledger key = AAAAAAAAAAC2LgFRDBZ3J52nLm30kq2iMgrO7dYzYAN3hvjtf1IHWg==)",
	)
}

func (s *TestLedgerEntryChangeCacheExistingUpdatedSuite) TestChangeUpdated() {
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
	}
	s.Assert().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()
	s.Assert().Len(changes, 1)
	s.Assert().Equal(changes[0].LedgerEntryChangeType(), xdr.LedgerEntryChangeTypeLedgerEntryUpdated)
	s.Assert().Equal(changes[0].Post.LastModifiedLedgerSeq, xdr.Uint32(12))
}

func (s *TestLedgerEntryChangeCacheExistingUpdatedSuite) TestChangeRemoved() {
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
		Post: nil,
	}
	s.Assert().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()
	s.Assert().Len(changes, 1)
	s.Assert().Equal(changes[0].LedgerEntryChangeType(), xdr.LedgerEntryChangeTypeLedgerEntryRemoved)
}

func TestLedgerEntryChangeCacheExistingRemoved(t *testing.T) {
	suite.Run(t, new(TestLedgerEntryChangeCacheExistingRemovedSuite))
}

// TestLedgerEntryChangeCacheExistingRemovedSuite tests transitions from
// existing REMOVED state in the cache.
type TestLedgerEntryChangeCacheExistingRemovedSuite struct {
	suite.Suite
	cache *LedgerEntryChangeCache
}

func (s *TestLedgerEntryChangeCacheExistingRemovedSuite) SetupTest() {
	s.cache = NewLedgerEntryChangeCache()

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
	s.Assert().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()
	s.Assert().Len(changes, 1)
	s.Assert().Equal(changes[0].LedgerEntryChangeType(), xdr.LedgerEntryChangeTypeLedgerEntryRemoved)
}

func (s *TestLedgerEntryChangeCacheExistingRemovedSuite) TestChangeCreated() {
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
	}
	s.Assert().NoError(s.cache.AddChange(change))
	changes := s.cache.GetChanges()
	s.Assert().Len(changes, 1)
	s.Assert().Equal(changes[0].LedgerEntryChangeType(), xdr.LedgerEntryChangeTypeLedgerEntryUpdated)
	s.Assert().Equal(changes[0].Post.LastModifiedLedgerSeq, xdr.Uint32(12))
}

func (s *TestLedgerEntryChangeCacheExistingRemovedSuite) TestChangeUpdated() {
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
	}
	s.Assert().EqualError(
		s.cache.AddChange(change),
		"can't update an entry that was previously removed (ledger key = AAAAAAAAAAC2LgFRDBZ3J52nLm30kq2iMgrO7dYzYAN3hvjtf1IHWg==)",
	)
}

func (s *TestLedgerEntryChangeCacheExistingRemovedSuite) TestChangeRemoved() {
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
		Post: nil,
	}
	s.Assert().EqualError(
		s.cache.AddChange(change),
		"can't remove an entry that was previously removed (ledger key = AAAAAAAAAAC2LgFRDBZ3J52nLm30kq2iMgrO7dYzYAN3hvjtf1IHWg==)",
	)
}

// TestLedgerEntryChangeCacheSquashMultiplePayments simulates sending multiple
// payments between two accounts. Ledger cache should squash multiple changes
// into just two.
// GAJ2T6NQ6TDZRVRSNWM3JC7L3TG4H7UBCVK3GUHKP3TQ5NQ3LM4JGBTJ sends money
// GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML receives money
func TestLedgerEntryChangeCacheSquashMultiplePayments(t *testing.T) {
	cache := NewLedgerEntryChangeCache()

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
		}
		assert.NoError(t, cache.AddChange(change))
	}

	changes := cache.GetChanges()
	assert.Len(t, changes, 2)
	for _, change := range changes {
		assert.Equal(t, change.LedgerEntryChangeType(), xdr.LedgerEntryChangeTypeLedgerEntryUpdated)
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
