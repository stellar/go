package ingest

import (
	"encoding/base64"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// ChangeCompactor is a cache of ledger entry changes that squashes all
// changes within a single ledger. By doing this, it decreases number of DB
// queries sent to a DB to update the current state of the ledger.
// It has integrity checks built in so ex. removing an account that was
// previously removed returns an error. In such case verify.StateError is
// returned.
//
// The ChangeCompactor should not be used when ingesting from history archives
// because the history archive snapshots only contain CREATED changes.
// The ChangeCompactor is suited for compacting ledger entry changes derived
// from LedgerCloseMeta payloads because they typically contain a mix of
// CREATED, UPDATED, and REMOVED ledger entry changes and therefore may benefit
// from compaction.
//
// It applies changes to the cache using the following algorithm:
//
//  1. If the change is CREATED it checks if any change connected to given entry
//     is already in the cache. If not, it adds CREATED change. Otherwise, if
//     existing change is
//     a. CREATED: return an error because we can't add an entry that already exists.
//     b. UPDATED: return an error because we can't add an entry that already exists.
//     c. REMOVED: entry exists in the DB but was marked for removal; change the type
//     to UPDATED and update the new value.
//     d. RESTORED: return an error as the RESTORED change indicates the entry already
//     exists.
//
//  2. If the change is UPDATE it checks if any change connected to given entry
//     is already in the cache. If not, it adds UPDATE change. Otherwise, if
//     existing change is
//     a. CREATED: We want to create this in a DB which means that it doesn't exist
//     in a DB so we need to update the entry but stay with CREATED type.
//     b. UPDATED: update it with the new value.
//     c. REMOVED it means that at this point in the ledger the entry is removed
//     so updating it returns an error.
//     d. RESTORED: update it with the new value but keep the type as RESTORED.
//
//  3. If the change is REMOVED, it checks if any change related to the given entry
//     already exists in the cache. If not, it adds the `REMOVED` change. Otherwise,
//     if existing change is
//     a. CREATED: due to previous transitions we want to create
//     this in a DB which means that it doesn't exist in a DB. If it was created and
//     removed in the same ledger it's a noop so we remove the entry from the cache.
//     b. UPDATED: update it to be a REMOVE change because the UPDATE change means
//     the entry exists in a DB.
//     c. REMOVED: return an error because we can't remove an entry that was already
//     removed.
//     d. RESTORED: if the item was previously restored from an archived state, it means
//     it already exists in the DB, so change it to REMOVED type. If the restored item
//     was evicted, it doesn't exist in the DB, so it's a noop so remove the entry from
//     the cache.
//
//  4. If the change is RESTORED for an evicted entry (pre is nil), it checks if any
//     change related to the given entry already exists in the cache. If not, it adds
//     the RESTORED change. Otherwise, if existing change is
//     a. CREATED: return an error because we can't restore and entry that already exists.
//     b. UPDATED: return an error because we can't restore an entry that already exists.
//     c. REMOVED: entry exists in the DB but was marked for removal; change the
//     type to RESTORED and update the new value.
//     d. RESTORED: return an error as the RESTORED change indicates the entry
//     already exists.
//
//  5. If the change is RESTORED for an archived entry (pre and post not nil), it checks
//     if any change related to the given entry already exists in the cache. If not,
//     it adds the RESTORED change. Otherwise, if existing change is
//     a. CREATED: it means that it doesn't exist in the DB so we need to update the
//     entry but stay with CREATED type.
//     b. UPDATED: update it with the new value and change the type to RESTORED.
//     c. REMOVED: return an error because we can not RESTORE an entry that was already
//     removed.
//     d. RESTORED: update it with the new value.
type ChangeCompactor struct {
	// ledger key => Change
	cache          map[string]Change
	encodingBuffer *xdr.EncodingBuffer
}

// NewChangeCompactor returns a new ChangeCompactor.
func NewChangeCompactor() *ChangeCompactor {
	return &ChangeCompactor{
		cache:          make(map[string]Change),
		encodingBuffer: xdr.NewEncodingBuffer(),
	}
}

// AddChange adds a change to ChangeCompactor. All changes are stored
// in memory. To get the final, squashed changes call GetChanges.
//
// Please note that the current ledger capacity in pubnet (max 1000 ops/ledger)
// makes ChangeCompactor safe to use in terms of memory usage. If the
// cache takes too much memory, you apply changes returned by GetChanges and
// create a new ChangeCompactor object to continue ingestion.
func (c *ChangeCompactor) AddChange(change Change) error {
	switch change.ChangeType {
	case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
		return c.addCreatedChange(change)
	case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
		return c.addUpdatedChange(change)
	case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
		return c.addRemovedChange(change)
	case xdr.LedgerEntryChangeTypeLedgerEntryRestored:
		return c.addRestoredChange(change)
	default:
		return errors.New("Unknown entry change state")
	}
}

// addCreatedChange adds a change to the cache, but returns an error if create
// change is unexpected.
func (c *ChangeCompactor) addCreatedChange(change Change) error {
	// safe, since we later cast to string (causing a copy)
	key, err := change.Post.LedgerKey()
	if err != nil {
		return errors.Wrap(err, "error getting ledger key for new entry")
	}
	ledgerKey, err := c.encodingBuffer.UnsafeMarshalBinary(key)
	if err != nil {
		return errors.Wrap(err, "error marshaling ledger key for new entry")
	}

	ledgerKeyString := string(ledgerKey)

	existingChange, exist := c.cache[ledgerKeyString]
	if !exist {
		c.cache[ledgerKeyString] = change
		return nil
	}

	switch existingChange.LedgerEntryChangeType() {
	case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
		fallthrough
	case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
		fallthrough
	case xdr.LedgerEntryChangeTypeLedgerEntryRestored:
		return NewStateError(errors.Errorf(
			"can't create an entry that already exists (ledger key = %s)",
			base64.StdEncoding.EncodeToString(ledgerKey),
		))
	case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
		// If existing type is removed it means that this entry does exist
		// in a DB so we update entry change.
		c.cache[ledgerKeyString] = Change{
			Type:       key.Type,
			Pre:        existingChange.Pre,
			Post:       change.Post,
			ChangeType: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
		}
	default:
		return errors.Errorf("Unknown LedgerEntryChangeType: %d", existingChange.LedgerEntryChangeType())
	}

	return nil
}

// addUpdatedChange adds a change to the cache, but returns an error if update
// change is unexpected.
func (c *ChangeCompactor) addUpdatedChange(change Change) error {
	// safe, since we later cast to string (causing a copy)
	key, err := change.Post.LedgerKey()
	if err != nil {
		return errors.Wrap(err, "error getting ledger key for updated entry")
	}
	ledgerKey, err := c.encodingBuffer.UnsafeMarshalBinary(key)
	if err != nil {
		return errors.Wrap(err, "error marshaling ledger key for updated entry")
	}

	ledgerKeyString := string(ledgerKey)

	existingChange, exist := c.cache[ledgerKeyString]
	if !exist {
		c.cache[ledgerKeyString] = change
		return nil
	}

	switch existingChange.LedgerEntryChangeType() {
	case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
		fallthrough
	case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
		fallthrough
	case xdr.LedgerEntryChangeTypeLedgerEntryRestored:
		// If existing type is created it means that this entry does not
		// exist in a DB so we update entry change.
		c.cache[ledgerKeyString] = Change{
			Type:       key.Type,
			Pre:        existingChange.Pre, // = nil for created type
			Post:       change.Post,
			ChangeType: existingChange.ChangeType,
		}
	case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
		return NewStateError(errors.Errorf(
			"can't update an entry that was previously removed (ledger key = %s)",
			base64.StdEncoding.EncodeToString(ledgerKey),
		))
	default:
		return errors.Errorf("Unknown LedgerEntryChangeType: %d", existingChange.LedgerEntryChangeType())
	}

	return nil
}

// addRemovedChange adds a change to the cache, but returns an error if remove
// change is unexpected.
func (c *ChangeCompactor) addRemovedChange(change Change) error {
	// safe, since we later cast to string (causing a copy)
	key, err := change.Pre.LedgerKey()
	if err != nil {
		return errors.Wrap(err, "error getting ledger key for removed entry")
	}
	ledgerKey, err := c.encodingBuffer.UnsafeMarshalBinary(key)
	if err != nil {
		return errors.Wrap(err, "error marshaling ledger key for removed entry")
	}

	ledgerKeyString := string(ledgerKey)

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
			Type:       key.Type,
			Pre:        existingChange.Pre,
			Post:       nil,
			ChangeType: change.ChangeType,
		}
	case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
		return NewStateError(errors.Errorf(
			"can't remove an entry that was previously removed (ledger key = %s)",
			base64.StdEncoding.EncodeToString(ledgerKey),
		))
	case xdr.LedgerEntryChangeTypeLedgerEntryRestored:
		if existingChange.Pre == nil {
			// Entry was created and removed in the same ledger; deleting it is effectively a noop.
			delete(c.cache, ledgerKeyString)
		} else {
			// If the entry exists, we mark it as removed by setting Post to nil.
			c.cache[ledgerKeyString] = Change{
				Type:       existingChange.Type,
				Pre:        existingChange.Pre,
				Post:       nil,
				ChangeType: change.ChangeType,
			}
		}
	default:
		return errors.Errorf("Unknown LedgerEntryChangeType: %d", existingChange.LedgerEntryChangeType())
	}

	return nil
}

// addRestoredChange adds a change to the cache, but returns an error if the restore
// change is unexpected.
func (c *ChangeCompactor) addRestoredChange(change Change) error {
	// safe, since we later cast to string (causing a copy)
	key, err := change.Post.LedgerKey()
	if err != nil {
		return errors.Wrap(err, "error getting ledger key for updated entry")
	}
	ledgerKey, err := c.encodingBuffer.UnsafeMarshalBinary(key)
	if err != nil {
		return errors.Wrap(err, "error marshaling ledger key for updated entry")
	}

	ledgerKeyString := string(ledgerKey)

	existingChange, exist := c.cache[ledgerKeyString]
	if !exist {
		c.cache[ledgerKeyString] = change
		return nil
	}
	// If 'Pre' is nil, it indicates that an item previously *evicted* is being restored.
	if change.Pre == nil {
		switch existingChange.ChangeType {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			fallthrough
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			fallthrough
		case xdr.LedgerEntryChangeTypeLedgerEntryRestored:
			return NewStateError(errors.Errorf(
				"can't restore an entry that already exists (ledger key = %s)",
				base64.StdEncoding.EncodeToString(ledgerKey),
			))
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			c.cache[ledgerKeyString] = Change{
				Type:       key.Type,
				Pre:        existingChange.Pre,
				Post:       change.Post,
				ChangeType: change.ChangeType,
			}
		default:
			return errors.Errorf("Unknown LedgerEntryChangeType: %d", existingChange.LedgerEntryChangeType())
		}
	} else {
		// If 'Pre' is not nil, it indicates that an item previously *archived* is being restored.
		switch existingChange.ChangeType {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			c.cache[ledgerKeyString] = Change{
				Type:       key.Type,
				Pre:        nil,
				Post:       change.Post,
				ChangeType: existingChange.ChangeType,
			}
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			fallthrough
		case xdr.LedgerEntryChangeTypeLedgerEntryRestored:
			c.cache[ledgerKeyString] = Change{
				Type:       key.Type,
				Pre:        existingChange.Pre,
				Post:       change.Post,
				ChangeType: change.ChangeType,
			}
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			return NewStateError(errors.Errorf(
				"can't restore an entry that was previously removed (ledger key = %s)", ledgerKey,
			))
		default:
			return errors.Errorf("Unknown LedgerEntryChangeType: %d", existingChange.LedgerEntryChangeType())
		}
	}
	return nil
}

// GetChanges returns a slice of Changes in the cache. The order of changes is
// random but each change is connected to a separate entry.
func (c *ChangeCompactor) GetChanges() []Change {
	changes := make([]Change, 0, len(c.cache))

	for _, entryChange := range c.cache {
		changes = append(changes, entryChange)
	}

	return changes
}

// Size returns number of ledger entries in the cache.
func (c *ChangeCompactor) Size() int {
	return len(c.cache)
}
