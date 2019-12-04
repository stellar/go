package expingest

import (
	"sync"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/verify"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LedgerEntryChangeCache is a cache of ledger entry changes that squashes
// data before applying them in a DB. By doing this, it decreases number of DB
// queries sent to a DB to update the current state of the ledger.
// It has integrity checks built in so ex. removing an account that was
// previously removed returns an error. In such case verify.StateError is
// returned.
type LedgerEntryChangeCache struct {
	HistoryQ history.Q

	// ledger key => ledger entry change
	cache map[string]xdr.LedgerEntryChange
	mutex sync.Mutex

	accountsBatch    history.AccountsBatchInsertBuilder
	accountDataBatch history.AccountDataBatchInsertBuilder
	offersBatch      history.OffersBatchInsertBuilder
	trustLinesBatch  history.TrustLinesBatchInsertBuilder
}

// Init initializes LedgerEntryChangeCache. maxBatchSize is a maximum capacity
// of DB insert batch. LedgerEntryChangeCache uses 4 batches internally, for
// each ledger entry type.
func (c *LedgerEntryChangeCache) Init(maxBatchSize int) error {
	c.cache = make(map[string]xdr.LedgerEntryChange)
	c.accountsBatch = c.HistoryQ.NewAccountsBatchInsertBuilder(maxBatchSize)
	c.accountDataBatch = c.HistoryQ.NewAccountDataBatchInsertBuilder(maxBatchSize)
	c.offersBatch = c.HistoryQ.NewOffersBatchInsertBuilder(maxBatchSize)
	c.trustLinesBatch = c.HistoryQ.NewTrustLinesBatchInsertBuilder(maxBatchSize)

	return nil
}

// AddChange adds a change to LedgerEntryChangeCache. All changes are stored
// in memory. The actual DB update is done in Commit() method.
// TODO: it should call Commit() internally if the cache grows too much.
func (c *LedgerEntryChangeCache) AddChange(change io.Change) error {
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
func (c *LedgerEntryChangeCache) addCreatedChange(change io.Change) error {
	ledgerKeyString, err := change.Post.LedgerKey().MarshalBinaryBase64()
	if err != nil {
		return errors.Wrap(err, "Error MarshalBinaryBase64")
	}

	entryChange, exist := c.cache[ledgerKeyString]
	if exist {
		switch entryChange.Type {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			return verify.NewStateError(errors.Errorf(
				"Creating an entry that already exist. Ledger key = %s",
				ledgerKeyString,
			))
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			return verify.NewStateError(errors.Errorf(
				"Creating an entry that already exist. Ledger key = %s",
				ledgerKeyString,
			))
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			// If existing type is removed it means that this entry does exist
			// in a DB so we update entry change.
			c.cache[ledgerKeyString] = xdr.LedgerEntryChange{
				Type:    xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
				Updated: change.Post,
			}
		default:
			return errors.Errorf("Unknown LedgerEntryChangeType: %d", entryChange.Type)
		}
	} else {
		c.cache[ledgerKeyString] = xdr.LedgerEntryChange{
			Type:    xdr.LedgerEntryChangeTypeLedgerEntryCreated,
			Created: change.Post,
		}
	}

	return nil
}

// addUpdatedChange adds a change to the cache, but returns an error if update
// change is unexpected.
func (c *LedgerEntryChangeCache) addUpdatedChange(change io.Change) error {
	ledgerKeyString, err := change.Post.LedgerKey().MarshalBinaryBase64()
	if err != nil {
		return errors.Wrap(err, "Error MarshalBinaryBase64")
	}

	entryChange, exist := c.cache[ledgerKeyString]
	if exist {
		switch entryChange.Type {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			// If existing type is created it means that this entry does not
			// exist in a DB so we update entry change.
			c.cache[ledgerKeyString] = xdr.LedgerEntryChange{
				Type:    xdr.LedgerEntryChangeTypeLedgerEntryCreated,
				Created: change.Post,
			}
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			c.cache[ledgerKeyString] = xdr.LedgerEntryChange{
				Type:    xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
				Updated: change.Post,
			}
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			return verify.NewStateError(errors.Errorf(
				"Updating an entry that was previously removed. Ledger key = %s",
				ledgerKeyString,
			))
		default:
			return errors.Errorf("Unknown LedgerEntryChangeType: %d", entryChange.Type)
		}
	}

	return nil
}

// addRemovedChange adds a change to the cache, but returns an error if remove
// change is unexpected.
func (c *LedgerEntryChangeCache) addRemovedChange(change io.Change) error {
	ledgerKey := change.Pre.LedgerKey()
	ledgerKeyString, err := ledgerKey.MarshalBinaryBase64()
	if err != nil {
		return errors.Wrap(err, "Error MarshalBinaryBase64")
	}

	entryChange, exist := c.cache[ledgerKeyString]
	if exist {
		switch entryChange.Type {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			// If existing type is created it means that this will be no op.
			// Entry was created and is now removed in a single ledger.
			delete(c.cache, ledgerKeyString)
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			c.cache[ledgerKeyString] = xdr.LedgerEntryChange{
				Type:    xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
				Removed: &ledgerKey,
			}
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			return verify.NewStateError(errors.Errorf(
				"Removing an entry that was previously removed. Ledger key = %s",
				ledgerKeyString,
			))
		default:
			return errors.Errorf("Unknown LedgerEntryChangeType: %d", entryChange.Type)
		}
	} else {
		c.cache[ledgerKeyString] = xdr.LedgerEntryChange{
			Type:    xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
			Removed: &ledgerKey,
		}
	}

	return nil
}

// Commit applies changes to a DB.
func (c *LedgerEntryChangeCache) Commit() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, entryChange := range c.cache {
		switch entryChange.Type {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			return c.insertEntry(entryChange)
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			return c.updateEntry(entryChange)
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			return c.removeEntry(entryChange)
		default:
			return errors.Errorf("Unknown LedgerEntryChangeType: %d", entryChange.Type)
		}
	}

	var err error

	// TODO for batch errors check for "unique constraints error" that should
	// result in verify.StateError
	err = c.accountsBatch.Exec()
	if err != nil {
		return errors.Wrap(err, "Error accountsBatch.Exec()")
	}

	err = c.accountDataBatch.Exec()
	if err != nil {
		return errors.Wrap(err, "Error accountDataBatch.Exec()")
	}

	err = c.offersBatch.Exec()
	if err != nil {
		return errors.Wrap(err, "Error offersBatch.Exec()")
	}

	err = c.trustLinesBatch.Exec()
	if err != nil {
		return errors.Wrap(err, "Error trustLinesBatch.Exec()")
	}

	c.cache = make(map[string]xdr.LedgerEntryChange)
	return nil
}

func (c *LedgerEntryChangeCache) insertEntry(entryChange xdr.LedgerEntryChange) error {
	entry := entryChange.MustCreated()

	switch entryChange.EntryType() {
	case xdr.LedgerEntryTypeAccount:
		return c.accountsBatch.Add(entry.Data.MustAccount(), entry.LastModifiedLedgerSeq)
	case xdr.LedgerEntryTypeData:
		return c.accountDataBatch.Add(entry.Data.MustData(), entry.LastModifiedLedgerSeq)
	case xdr.LedgerEntryTypeOffer:
		return c.offersBatch.Add(entry.Data.MustOffer(), entry.LastModifiedLedgerSeq)
	case xdr.LedgerEntryTypeTrustline:
		return c.trustLinesBatch.Add(entry.Data.MustTrustLine(), entry.LastModifiedLedgerSeq)
	default:
		return errors.Errorf("Unknown LedgerEntryType: %d", entryChange.EntryType())
	}
}

func (c *LedgerEntryChangeCache) updateEntry(entryChange xdr.LedgerEntryChange) error {
	entry := entryChange.MustUpdated()
	var rowsAffected int64
	var err error

	switch entryChange.EntryType() {
	case xdr.LedgerEntryTypeAccount:
		rowsAffected, err = c.HistoryQ.UpdateAccount(entry.Data.MustAccount(), entry.LastModifiedLedgerSeq)
	case xdr.LedgerEntryTypeData:
		rowsAffected, err = c.HistoryQ.UpdateAccountData(entry.Data.MustData(), entry.LastModifiedLedgerSeq)
	case xdr.LedgerEntryTypeOffer:
		rowsAffected, err = c.HistoryQ.UpdateOffer(entry.Data.MustOffer(), entry.LastModifiedLedgerSeq)
	case xdr.LedgerEntryTypeTrustline:
		rowsAffected, err = c.HistoryQ.UpdateTrustLine(entry.Data.MustTrustLine(), entry.LastModifiedLedgerSeq)
	default:
		return errors.Errorf("Unknown LedgerEntryType: %d", entryChange.EntryType())
	}

	if err != nil {
		return errors.Wrap(err, "error updating an entry")
	}

	if rowsAffected != 1 {
		entryChangeBase64, _ := entryChange.MarshalBinaryBase64()
		return verify.NewStateError(errors.Errorf(
			"No rows affected when updating using an entry change: %s",
			entryChangeBase64,
		))
	}

	return nil
}

func (c *LedgerEntryChangeCache) removeEntry(entryChange xdr.LedgerEntryChange) error {
	entry := entryChange.MustRemoved()
	var rowsAffected int64
	var err error

	switch entryChange.EntryType() {
	case xdr.LedgerEntryTypeAccount:
		ledgerKeyAccount := entry.LedgerKey().MustAccount()
		rowsAffected, err = c.HistoryQ.RemoveAccount(ledgerKeyAccount.AccountId.Address())
	case xdr.LedgerEntryTypeData:
		rowsAffected, err = c.HistoryQ.RemoveAccountData(entry.LedgerKey().MustData())
	case xdr.LedgerEntryTypeOffer:
		rowsAffected, err = c.HistoryQ.RemoveOffer(entry.LedgerKey().MustOffer().OfferId)
	case xdr.LedgerEntryTypeTrustline:
		rowsAffected, err = c.HistoryQ.RemoveTrustLine(entry.LedgerKey().MustTrustLine())
	default:
		return errors.Errorf("Unknown LedgerEntryType: %d", entryChange.EntryType())
	}

	if err != nil {
		return errors.Wrap(err, "error removing an entry")
	}

	if rowsAffected != 1 {
		entryChangeBase64, _ := entryChange.MarshalBinaryBase64()
		return verify.NewStateError(errors.Errorf(
			"No rows affected when removing using an entry change: %s",
			entryChangeBase64,
		))
	}

	return nil
}
