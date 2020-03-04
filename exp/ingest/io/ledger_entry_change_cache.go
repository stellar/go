package io

import (
	"sync"

	ingesterrors "github.com/stellar/go/exp/ingest/errors"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LedgerEntryChangeCache is a cache of ledger entry changes that squashes all
// changes within a single ledger. By doing this, it decreases number of DB
// queries sent to a DB to update the current state of the ledger.
// It has integrity checks built in so ex. removing an account that was
// previously removed returns an error. In such case verify.StateError is
// returned.
//
// It applies changes to the cache using the following algorithm:
//
// 1. If the change is CREATED it checks if any change connected to given entry
//    is already in the cache. If not, it adds CREATED change. Otherwise, if
//    existing change is:
//    a. CREATED it returns error because we can't add an entry that already
//       exists.
//    b. UPDATED it returns error because we can't add an entry that already
//       exists.
//    c. REMOVED it means that due to previous transitions we want to remove
//       this from a DB what means that it already exists in a DB so we need to
//       update the type of change to UPDATED.
// 2. If the change is UPDATE it checks if any change connected to given entry
//    is already in the cache. If not, it adds UPDATE change. Otherwise, if
//    existing change is:
//    a. CREATED it means that due to previous transitions we want to create
//       this in a DB what means that it doesn't exist in a DB so we need to
//       update the entry but stay with CREATED type.
//    b. UPDATED we simply update it with the new value.
//    c. REMOVED it means that at this point in the ledger the entry is removed
//       so updating it returns an error.
// 3. If the change is REMOVE it checks if any change connected to given entry
//    is already in the cache. If not, it adds REMOVE change. Otherwise, if
//    existing change is:
//    a. CREATED it means that due to previous transitions we want to create
//       this in a DB what means that it doesn't exist in a DB. If it was
//       created and removed in the same ledger it's a noop so we remove entry
//       from the cache.
//    b. UPDATED we simply update it to be a REMOVE change because the UPDATE
//       change means the entry exists in a DB.
//    c. REMOVED it returns error because we can't remove an entry that was
//       already removed.
type LedgerEntryChangeCache struct {
	// ledger key => Change
	cache map[string]Change
	mutex sync.Mutex
}

// NewLedgerEntryChangeCache returns a new LedgerEntryChangeCache.
func NewLedgerEntryChangeCache() *LedgerEntryChangeCache {
	return &LedgerEntryChangeCache{
		cache: make(map[string]Change),
	}
}

// AddChange adds a change to LedgerEntryChangeCache. All changes are stored
// in memory. To get the final, squashed changes call GetChanges.
//
// Please note that the current ledger capacity in pubnet (max 1000 ops/ledger)
// makes LedgerEntryChangeCache safe to use in terms of memory usage. If the
// cache takes too much memory, you apply changes returned by GetChanges and
// create a new LedgerEntryChangeCache object to continue ingestion.
func (c *LedgerEntryChangeCache) AddChange(change Change) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch {
	case change.Pre == nil && change.Post != nil:
		return c.addCreatedChange(change)
	case change.Pre != nil && change.Post != nil:
		return c.addUpdatedChange(change)
	case change.Pre != nil && change.Post == nil:
		return c.addRemovedChange(change)
	default:
		return errors.New("Unknown entry change state")
	}
}

// addCreatedChange adds a change to the cache, but returns an error if create
// change is unexpected.
func (c *LedgerEntryChangeCache) addCreatedChange(change Change) error {
	ledgerKeyString, err := change.Post.LedgerKey().MarshalBinaryBase64()
	if err != nil {
		return errors.Wrap(err, "Error MarshalBinaryBase64")
	}

	existingChange, exist := c.cache[ledgerKeyString]
	if !exist {
		c.cache[ledgerKeyString] = change
		return nil
	}

	switch existingChange.LedgerEntryChangeType() {
	case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
		return ingesterrors.NewStateError(errors.Errorf(
			"can't create an entry that already exists (ledger key = %s)",
			ledgerKeyString,
		))
	case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
		return ingesterrors.NewStateError(errors.Errorf(
			"can't create an entry that already exists (ledger key = %s)",
			ledgerKeyString,
		))
	case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
		// If existing type is removed it means that this entry does exist
		// in a DB so we update entry change.
		c.cache[ledgerKeyString] = Change{
			Type: change.Post.LedgerKey().Type,
			Pre:  existingChange.Pre,
			Post: change.Post,
		}
	default:
		return errors.Errorf("Unknown LedgerEntryChangeType: %d", existingChange.LedgerEntryChangeType())
	}

	return nil
}

// addUpdatedChange adds a change to the cache, but returns an error if update
// change is unexpected.
func (c *LedgerEntryChangeCache) addUpdatedChange(change Change) error {
	ledgerKeyString, err := change.Post.LedgerKey().MarshalBinaryBase64()
	if err != nil {
		return errors.Wrap(err, "Error MarshalBinaryBase64")
	}

	existingChange, exist := c.cache[ledgerKeyString]
	if !exist {
		c.cache[ledgerKeyString] = change
		return nil
	}

	switch existingChange.LedgerEntryChangeType() {
	case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
		// If existing type is created it means that this entry does not
		// exist in a DB so we update entry change.
		c.cache[ledgerKeyString] = Change{
			Type: change.Post.LedgerKey().Type,
			Pre:  existingChange.Pre, // = nil
			Post: change.Post,
		}
	case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
		c.cache[ledgerKeyString] = Change{
			Type: change.Post.LedgerKey().Type,
			Pre:  existingChange.Pre,
			Post: change.Post,
		}
	case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
		return ingesterrors.NewStateError(errors.Errorf(
			"can't update an entry that was previously removed (ledger key = %s)",
			ledgerKeyString,
		))
	default:
		return errors.Errorf("Unknown LedgerEntryChangeType: %d", existingChange.Type)
	}

	return nil
}

// addRemovedChange adds a change to the cache, but returns an error if remove
// change is unexpected.
func (c *LedgerEntryChangeCache) addRemovedChange(change Change) error {
	ledgerKeyString, err := change.Pre.LedgerKey().MarshalBinaryBase64()
	if err != nil {
		return errors.Wrap(err, "Error MarshalBinaryBase64")
	}

	existingChange, exist := c.cache[ledgerKeyString]
	if !exist {
		c.cache[ledgerKeyString] = change
		return nil
	}

	switch existingChange.LedgerEntryChangeType() {
	case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
		// If existing type is created it means that this will be no op.
		// Entry was created and is now removed in a single ledger.
		delete(c.cache, ledgerKeyString)
	case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
		c.cache[ledgerKeyString] = Change{
			Type: change.Pre.LedgerKey().Type,
			Pre:  existingChange.Pre,
			Post: nil,
		}
	case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
		return ingesterrors.NewStateError(errors.Errorf(
			"can't remove an entry that was previously removed (ledger key = %s)",
			ledgerKeyString,
		))
	default:
		return errors.Errorf("Unknown LedgerEntryChangeType: %d", existingChange.Type)
	}

	return nil
}

// GetChanges returns a slice of Changes in the cache. The order of changes is
// random but each change is connected to a separate entry.
func (c *LedgerEntryChangeCache) GetChanges() []Change {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	changes := make([]Change, 0, len(c.cache))

	for _, entryChange := range c.cache {
		changes = append(changes, entryChange)
	}

	return changes
}

// Size returns number of ledger entries in the cache.
func (c *LedgerEntryChangeCache) Size() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return len(c.cache)
}
