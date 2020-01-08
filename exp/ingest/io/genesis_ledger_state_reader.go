package io

import (
	"io"
	"sync"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
)

// GenesisLedgerStateReader is a streaming ledger entries for genesis ledger
// (ledgerseq = 1) for of the network with the given passphrase.
type GenesisLedgerStateReader struct {
	NetworkPassphrase string

	mutex sync.Mutex
	done  bool
}

// Ensure GenesisLedgerStateReader implements StateReader
var _ StateReader = &GenesisLedgerStateReader{}

// GetSequence returns the sequence of the ledger.
func (r *GenesisLedgerStateReader) GetSequence() uint32 {
	return 1
}

// Read returns a new ledger entry change on each call, returning io.EOF when the stream ends.
func (r *GenesisLedgerStateReader) Read() (xdr.LedgerEntryChange, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.done {
		return xdr.LedgerEntryChange{}, io.EOF
	}

	masterKeyPair := keypair.Master(r.NetworkPassphrase)

	masterAccountEntry := xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeAccount,
			Account: &xdr.AccountEntry{
				AccountId: xdr.MustAddress(masterKeyPair.Address()),
				// 100B
				Balance:    amount.MustParse("100000000000"),
				SeqNum:     0,
				Thresholds: xdr.Thresholds{1, 0, 0, 0},
			},
		},
	}

	r.done = true
	return xdr.LedgerEntryChange{
		Type:  xdr.LedgerEntryChangeTypeLedgerEntryState,
		State: &masterAccountEntry,
	}, nil
}

// Close should be called when reading is finished.
func (r *GenesisLedgerStateReader) Close() error {
	return nil
}
