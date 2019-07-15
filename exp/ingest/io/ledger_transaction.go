package io

import (
	"github.com/stellar/go/xdr"
)

// Change is a developer friendly representation of LedgerEntryChanges.
// It also provides some helper functions to quickly check if a given
// change has occured in an entry.
//
// If an entry is created: Pre is nil and Post is not nil.
// If an entry is updated: Pre is not nil and Post is not nil.
// If an entry is removed: Pre is not nil and Post is nil.
type Change struct {
	Type xdr.LedgerEntryType
	Pre  *xdr.LedgerEntryData
	Post *xdr.LedgerEntryData
}

func (c *Change) AccountSignersChanged() bool {
	if c.Type != xdr.LedgerEntryTypeAccount {
		panic("This should not be called on changes other than Account changes")
	}

	// Signers must be removed before merging an account and it's
	// impossible to add signers during a creation of a new account.
	if c.Pre == nil || c.Post == nil {
		return false
	}

	if len(c.Pre.MustAccount().Signers) != len(c.Post.MustAccount().Signers) {
		return true
	}

	signers := map[string]uint32{} // signer => weight

	for _, signer := range c.Pre.MustAccount().Signers {
		signers[signer.Key.Address()] = uint32(signer.Weight)
	}

	for _, signer := range c.Post.MustAccount().Signers {
		weight, exist := signers[signer.Key.Address()]
		if !exist {
			return false
		}

		if weight != uint32(signer.Weight) {
			return false
		}
	}

	// TODO should it also change on master key weight changes?

	return false
}

// GetChanges returns a developer friendly representation of LedgerEntryChanges.
// It contains fee changes, transaction changes and operation changes in that
// order.
func (t *LedgerTransaction) GetChanges() []Change {
	// Fee meta
	changes := getChangesFromLedgerEntryChanges(t.FeeChanges)

	// Transaction meta
	v1Meta, ok := t.Meta.GetV1()
	if ok {
		txChanges := getChangesFromLedgerEntryChanges(v1Meta.TxChanges)
		changes = append(changes, txChanges...)
	}

	// Operation meta
	for _, operationMeta := range t.Meta.OperationsMeta() {
		ledgerEntryChanges := operationMeta.Changes
		opChanges := getChangesFromLedgerEntryChanges(ledgerEntryChanges)

		changes = append(changes, opChanges...)
	}

	return changes
}

// getChangesFromLedgerEntryChanges transforms LedgerEntryChanges to []Change.
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
func getChangesFromLedgerEntryChanges(ledgerEntryChanges xdr.LedgerEntryChanges) []Change {
	changes := []Change{}

	for i, entryChange := range ledgerEntryChanges {
		switch entryChange.Type {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			created := entryChange.MustCreated()
			changes = append(changes, Change{
				Type: created.Data.Type,
				Pre:  nil,
				Post: &created.Data,
			})
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			state := ledgerEntryChanges[i-1].MustState()
			updated := entryChange.MustUpdated()
			changes = append(changes, Change{
				Type: state.Data.Type,
				Pre:  &state.Data,
				Post: &updated.Data,
			})
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			state := ledgerEntryChanges[i-1].MustState()
			changes = append(changes, Change{
				Type: state.Data.Type,
				Pre:  &state.Data,
				Post: nil,
			})
		case xdr.LedgerEntryChangeTypeLedgerEntryState:
			continue
		default:
			panic("Invalid LedgerEntryChangeType")
		}
	}

	return changes
}
