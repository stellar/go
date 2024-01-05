package ingest

import (
	"bytes"
	"sort"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// Change is a developer friendly representation of LedgerEntryChanges.
// It also provides some helper functions to quickly check if a given
// change has occurred in an entry.
//
// If an entry is created: Pre is nil and Post is not nil.
// If an entry is updated: Pre is not nil and Post is not nil.
// If an entry is removed: Pre is not nil and Post is nil.
type Change struct {
	Type xdr.LedgerEntryType
	Pre  *xdr.LedgerEntry
	Post *xdr.LedgerEntry
}

func (c *Change) ledgerKey() (xdr.LedgerKey, error) {
	if c.Pre != nil {
		return c.Pre.LedgerKey()
	}
	return c.Post.LedgerKey()
}

// GetChangesFromLedgerEntryChanges transforms LedgerEntryChanges to []Change.
// Each `update` and `removed` is preceded with `state` and `create` changes
// are alone, without `state`. The transformation we're doing is to move each
// change (state/update, state/removed or create) to an array of pre/post pairs.
// Then:
// - for create, pre is null and post is a new entry,
// - for update, pre is previous state and post is the current state,
// - for removed, pre is previous state and post is null.
//
// stellar-core source:
// https://github.com/stellar/stellar-core/blob/e584b43/src/ledger/LedgerTxn.cpp#L582
func GetChangesFromLedgerEntryChanges(ledgerEntryChanges xdr.LedgerEntryChanges) []Change {
	changes := make([]Change, 0, len(ledgerEntryChanges))
	for i, entryChange := range ledgerEntryChanges {
		switch entryChange.Type {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			created := entryChange.MustCreated()
			changes = append(changes, Change{
				Type: created.Data.Type,
				Pre:  nil,
				Post: &created,
			})
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			state := ledgerEntryChanges[i-1].MustState()
			updated := entryChange.MustUpdated()
			changes = append(changes, Change{
				Type: state.Data.Type,
				Pre:  &state,
				Post: &updated,
			})
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			state := ledgerEntryChanges[i-1].MustState()
			changes = append(changes, Change{
				Type: state.Data.Type,
				Pre:  &state,
				Post: nil,
			})
		case xdr.LedgerEntryChangeTypeLedgerEntryState:
			continue
		default:
			panic("Invalid LedgerEntryChangeType")
		}
	}

	sortChanges(changes)
	return changes
}

type sortableChanges struct {
	changes    []Change
	ledgerKeys [][]byte
}

func newSortableChanges(changes []Change) sortableChanges {
	ledgerKeys := make([][]byte, len(changes))
	for i, c := range changes {
		lk, err := c.ledgerKey()
		if err != nil {
			panic(err)
		}
		lkBytes, err := lk.MarshalBinary()
		if err != nil {
			panic(err)
		}
		ledgerKeys[i] = lkBytes
	}
	return sortableChanges{
		changes:    changes,
		ledgerKeys: ledgerKeys,
	}
}

func (s sortableChanges) Len() int {
	return len(s.changes)
}

func (s sortableChanges) Less(i, j int) bool {
	return bytes.Compare(s.ledgerKeys[i], s.ledgerKeys[j]) < 0
}

func (s sortableChanges) Swap(i, j int) {
	s.changes[i], s.changes[j] = s.changes[j], s.changes[i]
	s.ledgerKeys[i], s.ledgerKeys[j] = s.ledgerKeys[j], s.ledgerKeys[i]
}

// sortChanges is applied on a list of changes to ensure that LedgerEntryChanges
// from Tx Meta are ingested in a deterministic order.
// The changes are sorted by ledger key. It is unexpected for there to be
// multiple changes with the same ledger key in a LedgerEntryChanges group,
// but if that is the case, we fall back to the original ordering of the changes
// by using a stable sorting algorithm.
func sortChanges(changes []Change) {
	sort.Stable(newSortableChanges(changes))
}

// LedgerEntryChangeType returns type in terms of LedgerEntryChangeType.
func (c *Change) LedgerEntryChangeType() xdr.LedgerEntryChangeType {
	switch {
	case c.Pre == nil && c.Post != nil:
		return xdr.LedgerEntryChangeTypeLedgerEntryCreated
	case c.Pre != nil && c.Post == nil:
		return xdr.LedgerEntryChangeTypeLedgerEntryRemoved
	case c.Pre != nil && c.Post != nil:
		return xdr.LedgerEntryChangeTypeLedgerEntryUpdated
	default:
		panic("Invalid state of Change (Pre == nil && Post == nil)")
	}
}

// getLiquidityPool gets the most recent state of the LiquidityPool that exists or existed.
func (c *Change) getLiquidityPool() (*xdr.LiquidityPoolEntry, error) {
	var entry *xdr.LiquidityPoolEntry
	if c.Pre != nil {
		entry = c.Pre.Data.LiquidityPool
	}
	if c.Post != nil {
		entry = c.Post.Data.LiquidityPool
	}
	if entry == nil {
		return &xdr.LiquidityPoolEntry{}, errors.New("this change does not include a liquidity pool")
	}
	return entry, nil
}

// GetLiquidityPoolType returns the liquidity pool type.
func (c *Change) GetLiquidityPoolType() (xdr.LiquidityPoolType, error) {
	lp, err := c.getLiquidityPool()
	if err != nil {
		return xdr.LiquidityPoolType(0), err
	}
	return lp.Body.Type, nil
}

// AccountChangedExceptSigners returns true if account has changed WITHOUT
// checking the signers (except master key weight!). In other words, if the only
// change is connected to signers, this function will return false.
func (c *Change) AccountChangedExceptSigners() (bool, error) {
	if c.Type != xdr.LedgerEntryTypeAccount {
		panic("This should not be called on changes other than Account changes")
	}

	// New account
	if c.Pre == nil {
		return true, nil
	}

	// Account merged
	// c.Pre != nil at this point.
	if c.Post == nil {
		return true, nil
	}

	// c.Pre != nil && c.Post != nil at this point.
	if c.Pre.LastModifiedLedgerSeq != c.Post.LastModifiedLedgerSeq {
		return true, nil
	}

	// Don't use short assignment statement (:=) to ensure variables below
	// are not pointers (if `xdr` package changes in the future)!
	var preAccountEntry, postAccountEntry xdr.AccountEntry
	preAccountEntry = c.Pre.Data.MustAccount()
	postAccountEntry = c.Post.Data.MustAccount()

	// preAccountEntry and postAccountEntry are copies so it's fine to
	// modify them here, EXCEPT pointers inside them!
	if preAccountEntry.Ext.V == 0 {
		preAccountEntry.Ext.V = 1
		preAccountEntry.Ext.V1 = &xdr.AccountEntryExtensionV1{
			Liabilities: xdr.Liabilities{
				Buying:  0,
				Selling: 0,
			},
		}
	}

	preAccountEntry.Signers = nil

	if postAccountEntry.Ext.V == 0 {
		postAccountEntry.Ext.V = 1
		postAccountEntry.Ext.V1 = &xdr.AccountEntryExtensionV1{
			Liabilities: xdr.Liabilities{
				Buying:  0,
				Selling: 0,
			},
		}
	}

	postAccountEntry.Signers = nil

	preBinary, err := preAccountEntry.MarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "Error running preAccountEntry.MarshalBinary")
	}

	postBinary, err := postAccountEntry.MarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "Error running postAccountEntry.MarshalBinary")
	}

	return !bytes.Equal(preBinary, postBinary), nil
}

// AccountSignersChanged returns true if account signers have changed.
// Notice: this will return true on master key changes too!
func (c *Change) AccountSignersChanged() bool {
	if c.Type != xdr.LedgerEntryTypeAccount {
		panic("This should not be called on changes other than Account changes")
	}

	// New account so new master key (which is also a signer)
	if c.Pre == nil {
		return true
	}

	// Account merged. Account being merge can still have signers.
	// c.Pre != nil at this point.
	if c.Post == nil {
		return true
	}

	// c.Pre != nil && c.Post != nil at this point.
	preAccountEntry := c.Pre.Data.MustAccount()
	postAccountEntry := c.Post.Data.MustAccount()

	preSigners := preAccountEntry.SignerSummary()
	postSigners := postAccountEntry.SignerSummary()

	if len(preSigners) != len(postSigners) {
		return true
	}

	for postSigner, postWeight := range postSigners {
		preWeight, exist := preSigners[postSigner]
		if !exist {
			return true
		}

		if preWeight != postWeight {
			return true
		}
	}

	preSignerSponsors := preAccountEntry.SignerSponsoringIDs()
	postSignerSponsors := postAccountEntry.SignerSponsoringIDs()

	if len(preSignerSponsors) != len(postSignerSponsors) {
		return true
	}

	for i := 0; i < len(preSignerSponsors); i++ {
		preSponsor := preSignerSponsors[i]
		postSponsor := postSignerSponsors[i]

		if preSponsor == nil && postSponsor != nil {
			return true
		} else if preSponsor != nil && postSponsor == nil {
			return true
		} else if preSponsor != nil && postSponsor != nil {
			preSponsorAccountID := xdr.AccountId(*preSponsor)
			preSponsorAddress := preSponsorAccountID.Address()

			postSponsorAccountID := xdr.AccountId(*postSponsor)
			postSponsorAddress := postSponsorAccountID.Address()

			if preSponsorAddress != postSponsorAddress {
				return true
			}
		}
	}

	return false
}
