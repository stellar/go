package io

import (
	"bytes"

	"github.com/stellar/go/support/errors"
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
	Pre  *xdr.LedgerEntry
	Post *xdr.LedgerEntry
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
		preAccountEntry.Ext.V1 = &xdr.AccountEntryV1{
			Liabilities: xdr.Liabilities{
				Buying:  0,
				Selling: 0,
			},
		}
	}

	preAccountEntry.Signers = nil

	if postAccountEntry.Ext.V == 0 {
		postAccountEntry.Ext.V = 1
		postAccountEntry.Ext.V1 = &xdr.AccountEntryV1{
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

	return false
}

// TxResultCode returns the transaction result code
func (t *LedgerTransaction) TxResultCode() xdr.TransactionResultCode {
	return t.Result.Result.Result.Code
}

// Successful returns true if the transaction succeeded
func (t *LedgerTransaction) Successful() bool {
	return t.TxResultCode() == xdr.TransactionResultCodeTxSuccess
}

func (t *LedgerTransaction) txInternalError() bool {
	return t.TxResultCode() == xdr.TransactionResultCodeTxInternalError
}

// GetFeeChanges returns a developer friendly representation of LedgerEntryChanges
// connected to fees.
func (t *LedgerTransaction) GetFeeChanges() []Change {
	return getChangesFromLedgerEntryChanges(t.FeeChanges)
}

// GetChanges returns a developer friendly representation of LedgerEntryChanges.
// It contains transaction changes and operation changes in that order. If the
// transaction failed with TxInternalError, operations and txChangesAfter are
// omitted. It doesn't support legacy TransactionMeta.V=0.
func (t *LedgerTransaction) GetChanges() ([]Change, error) {
	var changes []Change

	// Transaction meta
	switch t.Meta.V {
	case 0:
		return changes, errors.New("TransactionMeta.V=0 not supported")
	case 1:
		v1Meta := t.Meta.MustV1()
		txChanges := getChangesFromLedgerEntryChanges(v1Meta.TxChanges)
		changes = append(changes, txChanges...)

		// Ignore operations meta if txInternalError https://github.com/stellar/go/issues/2111
		if t.txInternalError() {
			return changes, nil
		}

		for _, operationMeta := range v1Meta.Operations {
			opChanges := getChangesFromLedgerEntryChanges(
				operationMeta.Changes,
			)
			changes = append(changes, opChanges...)
		}

	case 2:
		v2Meta := t.Meta.MustV2()
		txChangesBefore := getChangesFromLedgerEntryChanges(v2Meta.TxChangesBefore)
		changes = append(changes, txChangesBefore...)

		// Ignore operations meta and txChangesAfter if txInternalError
		// https://github.com/stellar/go/issues/2111
		if t.txInternalError() {
			return changes, nil
		}

		for _, operationMeta := range v2Meta.Operations {
			opChanges := getChangesFromLedgerEntryChanges(
				operationMeta.Changes,
			)
			changes = append(changes, opChanges...)
		}

		txChangesAfter := getChangesFromLedgerEntryChanges(v2Meta.TxChangesAfter)
		changes = append(changes, txChangesAfter...)
	default:
		return changes, errors.New("Unsupported TransactionMeta version")
	}

	return changes, nil
}

// GetOperationChanges returns a developer friendly representation of LedgerEntryChanges.
// It contains only operation changes.
func (t *LedgerTransaction) GetOperationChanges(operationIndex uint32) ([]Change, error) {
	changes := []Change{}

	// Transaction meta
	switch t.Meta.V {
	case 0:
		return changes, errors.New("TransactionMeta.V=0 not supported")
	case 1:
		// Ignore operations meta if txInternalError https://github.com/stellar/go/issues/2111
		if t.txInternalError() {
			return changes, nil
		}

		v1Meta := t.Meta.MustV1()
		changes = operationChanges(v1Meta.Operations, operationIndex)
	case 2:
		// Ignore operations meta if txInternalError https://github.com/stellar/go/issues/2111
		if t.txInternalError() {
			return changes, nil
		}

		v2Meta := t.Meta.MustV2()
		changes = operationChanges(v2Meta.Operations, operationIndex)
	default:
		return changes, errors.New("Unsupported TransactionMeta version")
	}

	return changes, nil
}

func operationChanges(ops []xdr.OperationMeta, index uint32) []Change {
	if len(ops) == 0 || int(index) >= len(ops) {
		return []Change{}
	}

	operationMeta := ops[index]
	return getChangesFromLedgerEntryChanges(
		operationMeta.Changes,
	)
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

	return changes
}
