package io

import (
	"io"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
)

// ChangeReader provides convenient, streaming access to a sequence of Changes.
type ChangeReader interface {
	// Read should return the next `Change` in the leader. If there are no more
	// changes left it should return an `io.EOF` error.
	Read() (Change, error)
	// Close should be called when reading is finished. This is especially
	// helpful when there are still some changes available so reader can stop
	// streaming them.
	Close() error
}

type ledgerChangeReaderState int

const (
	feeChangesState ledgerChangeReaderState = iota
	metaChangesState
	upgradeChangesState
)

// LedgerChangeReader is a ChangeReader which returns Changes from Stellar Core
// for a single ledger
type LedgerChangeReader struct {
	*LedgerTransactionReader
	state        ledgerChangeReaderState
	pending      []Change
	pendingIndex int
	upgradeIndex int
}

// Ensure LedgerChangeReader implements ChangeReader
var _ ChangeReader = (*LedgerChangeReader)(nil)

// NewLedgerChangeReader constructs a new LedgerChangeReader instance bound to the given ledger.
// Note that the returned LedgerChangeReader is not thread safe and should not be shared
// by multiple goroutines.
func NewLedgerChangeReader(backend ledgerbackend.LedgerBackend, sequence uint32) (*LedgerChangeReader, error) {
	transactionReader, err := NewLedgerTransactionReader(backend, sequence)
	if err != nil {
		return nil, err
	}

	return &LedgerChangeReader{
		LedgerTransactionReader: transactionReader,
		state:                   feeChangesState,
	}, nil
}

// Read returns the next change in the stream.
// If there are no changes remaining io.EOF is returned as an error.
func (r *LedgerChangeReader) Read() (Change, error) {
	if r.pendingIndex < len(r.pending) {
		next := r.pending[r.pendingIndex]
		r.pendingIndex++
		if r.pendingIndex == len(r.pending) {
			r.pendingIndex = 0
			r.pending = r.pending[:0]
		}
		return next, nil
	}

	switch r.state {
	case feeChangesState, metaChangesState:
		tx, err := r.LedgerTransactionReader.Read()
		if err != nil {
			if err == io.EOF {
				// If done streaming fee changes rewind to stream meta changes
				if r.state == feeChangesState {
					r.LedgerTransactionReader.Rewind()
				}
				r.state++
				return r.Read()
			}
			return Change{}, err
		}

		switch r.state {
		case feeChangesState:
			r.pending = append(r.pending, tx.GetFeeChanges()...)
		case metaChangesState:
			metaChanges, err := tx.GetChanges()
			if err != nil {
				return Change{}, err
			}
			r.pending = append(r.pending, metaChanges...)
		}
		return r.Read()
	case upgradeChangesState:
		// Get upgrade changes
		if r.upgradeIndex < len(r.LedgerTransactionReader.LedgerReader.ledgerCloseMeta.UpgradesMeta) {
			changes := GetChangesFromLedgerEntryChanges(
				r.LedgerTransactionReader.LedgerReader.ledgerCloseMeta.UpgradesMeta[r.upgradeIndex],
			)
			r.pending = append(r.pending, changes...)
			r.upgradeIndex++
			return r.Read()
		}
	}

	return Change{}, io.EOF
}

// Close should be called when reading is finished.
func (r *LedgerChangeReader) Close() error {
	r.pending = nil
	return r.LedgerTransactionReader.Close()
}
