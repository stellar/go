package xdr

import "fmt"

// EntryType is a helper to get the entry type for a change. It
// panics if unable to do so.
func (change *LedgerEntryChange) EntryType() LedgerEntryType {
	return change.LedgerKey().Type
}

// GetEntryType is a helper to get the entry type for a change, and it
// returns an error if unable to do so.
func (change *LedgerEntryChange) GetEntryType() (LedgerEntryType, error) {
	key, err := change.GetLedgerKey()
	if err != nil {
		return LedgerEntryTypeAccount, err
	}
	return key.Type, nil
}

// LedgerKey returns the key for the ledger entry that was changed
// in `change`.
func (change *LedgerEntryChange) LedgerKey() LedgerKey {
	key, err := change.GetLedgerKey()
	if err != nil {
		panic(err)
	}
	return key
}

// GetLedgerKey returns the key for the ledger entry changed in `change`,
// or panics if it is unable to do so.
func (change *LedgerEntryChange) GetLedgerKey() (LedgerKey, error) {
	switch change.Type {
	case LedgerEntryChangeTypeLedgerEntryCreated:
		created, ok := change.GetCreated()
		if !ok {
			return LedgerKey{}, fmt.Errorf("could not get created")
		}
		return created.GetLedgerKey()
	case LedgerEntryChangeTypeLedgerEntryRemoved:
		removed, ok := change.GetRemoved()
		if !ok {
			return LedgerKey{}, fmt.Errorf("could not get removed")
		}
		return removed, nil
	case LedgerEntryChangeTypeLedgerEntryUpdated:
		updated, ok := change.GetUpdated()
		if !ok {
			return LedgerKey{}, fmt.Errorf("could not get updated")
		}
		return updated.GetLedgerKey()
	case LedgerEntryChangeTypeLedgerEntryState:
		state, ok := change.GetState()
		if !ok {
			return LedgerKey{}, fmt.Errorf("could not get state")
		}
		return state.GetLedgerKey()
	default:
		return LedgerKey{}, fmt.Errorf("unknown change type: %v", change.Type)
	}
}

// GetLedgerEntry returns the ledger entry that was changed in `change`, along
// with a boolean indicating whether the entry value was valid.
func (change *LedgerEntryChange) GetLedgerEntry() (LedgerEntry, bool) {
	switch change.Type {
	case LedgerEntryChangeTypeLedgerEntryCreated:
		return change.GetCreated()
	case LedgerEntryChangeTypeLedgerEntryState:
		return change.GetState()
	case LedgerEntryChangeTypeLedgerEntryUpdated:
		return change.GetUpdated()
	case LedgerEntryChangeTypeLedgerEntryRemoved:
		return LedgerEntry{}, false
	default:
		return LedgerEntry{}, false
	}
}
