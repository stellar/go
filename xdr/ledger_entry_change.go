package xdr

import "fmt"

// EntryType is a helper to get at the entry type for a change.
func (change *LedgerEntryChange) EntryType() LedgerEntryType {
	return change.LedgerKey().Type
}

// LedgerKey returns the key for the ledger entry that was changed
// in `change`.
func (change *LedgerEntryChange) LedgerKey() LedgerKey {
	switch change.Type {
	case LedgerEntryChangeTypeLedgerEntryCreated:
		change := change.MustCreated()
		return change.LedgerKey()
	case LedgerEntryChangeTypeLedgerEntryRemoved:
		return change.MustRemoved()
	case LedgerEntryChangeTypeLedgerEntryUpdated:
		change := change.MustUpdated()
		return change.LedgerKey()
	case LedgerEntryChangeTypeLedgerEntryState:
		change := change.MustState()
		return change.LedgerKey()
	default:
		panic(fmt.Errorf("Unknown change type: %v", change.Type))
	}
}

// GetLedgerEntry returns the ledger entry that was changed in `change`, and
// an error if that entry cannot be retrieved.
func (change *LedgerEntryChange) GetLedgerEntry() (*LedgerEntry, error) {
	var (
		entry LedgerEntry
		ok    bool
	)
	switch change.Type {
	case LedgerEntryChangeTypeLedgerEntryCreated:
		entry, ok = change.GetCreated()
	case LedgerEntryChangeTypeLedgerEntryState:
		entry, ok = change.GetState()
	case LedgerEntryChangeTypeLedgerEntryUpdated:
		entry, ok = change.GetUpdated()
	case LedgerEntryChangeTypeLedgerEntryRemoved:
		return nil, fmt.Errorf("Entry type %v does not have ledger entry", change.Type)
	default:
		return nil, fmt.Errorf("Unknown change type: %v", change.Type)
	}

	if !ok {
		return nil, fmt.Errorf("Could not get entry of type %v from change", change.Type)
	}
	return &entry, nil
}
