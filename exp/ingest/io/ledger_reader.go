package io

import (
	"io"
	"sync"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// DBLedgerReader is a database-backed implementation of the io.LedgerReader interface.
type DBLedgerReader struct {
	sequence       uint32
	backend        ledgerbackend.LedgerBackend
	header         xdr.LedgerHeaderHistoryEntry
	transactions   []LedgerTransaction
	upgradeChanges []Change
	initOnce       sync.Once
	readMutex      sync.Mutex
	readIdx        int
	upgradeReadIdx int
}

// Ensure DBLedgerReader implements LedgerReader
var _ LedgerReader = (*DBLedgerReader)(nil)

// NewDBLedgerReader is a factory method for LedgerReader.
func NewDBLedgerReader(sequence uint32, backend ledgerbackend.LedgerBackend) (*DBLedgerReader, error) {
	reader := &DBLedgerReader{
		sequence: sequence,
		backend:  backend,
	}

	var err error
	reader.initOnce.Do(func() { err = reader.init() })
	if err != nil {
		return nil, err
	}

	return reader, nil
}

// GetSequence returns the sequence number of the ledger data stored by this object.
func (dblrc *DBLedgerReader) GetSequence() uint32 {
	return dblrc.sequence
}

// GetHeader returns the XDR Header data associated with the stored ledger.
func (dblrc *DBLedgerReader) GetHeader() xdr.LedgerHeaderHistoryEntry {
	var err error
	dblrc.initOnce.Do(func() { err = dblrc.init() })
	if err != nil {
		// TODO, object should be initialized in constructor.
		// Not returning error here, makes this much simpler.
		panic(err)
	}
	return dblrc.header
}

// Read returns the next transaction in the ledger, ordered by tx number, each time it is called. When there
// are no more transactions to return, an EOF error is returned.
func (dblrc *DBLedgerReader) Read() (LedgerTransaction, error) {
	var err error
	dblrc.initOnce.Do(func() { err = dblrc.init() })
	if err != nil {
		return LedgerTransaction{}, err
	}

	// Protect all accesses to dblrc.readIdx
	dblrc.readMutex.Lock()
	defer dblrc.readMutex.Unlock()

	if dblrc.readIdx < len(dblrc.transactions) {
		dblrc.readIdx++
		return dblrc.transactions[dblrc.readIdx-1], nil
	}
	return LedgerTransaction{}, io.EOF
}

// ReadUpgradeChange returns the next upgrade change in the ledger, each time it is called. When there
// are no more upgrades to return, an EOF error is returned.
func (dblrc *DBLedgerReader) ReadUpgradeChange() (Change, error) {
	var err error
	dblrc.initOnce.Do(func() { err = dblrc.init() })
	if err != nil {
		return Change{}, err
	}

	// Protect all accesses to dblrc.readIdx
	dblrc.readMutex.Lock()
	defer dblrc.readMutex.Unlock()

	if dblrc.upgradeReadIdx < len(dblrc.transactions) {
		dblrc.upgradeReadIdx++
		return dblrc.upgradeChanges[dblrc.upgradeReadIdx-1], nil
	}
	return Change{}, io.EOF
}

// Close moves the read pointer so that subsequent calls to Read() will return EOF.
func (dblrc *DBLedgerReader) Close() error {
	dblrc.readMutex.Lock()
	dblrc.readIdx = len(dblrc.transactions)
	dblrc.readMutex.Unlock()

	return nil
}

// Init pulls data from the backend to set this object up for use.
func (dblrc *DBLedgerReader) init() error {
	exists, ledgerCloseMeta, err := dblrc.backend.GetLedger(dblrc.sequence)

	if err != nil {
		return errors.Wrap(err, "error reading ledger from backend")
	}
	if !exists {
		return ErrNotFound
	}

	dblrc.header = ledgerCloseMeta.LedgerHeader

	dblrc.storeTransactions(ledgerCloseMeta)
	for _, upgradeChanges := range ledgerCloseMeta.UpgradesMeta {
		changes := getChangesFromLedgerEntryChanges(upgradeChanges)
		dblrc.upgradeChanges = append(dblrc.upgradeChanges, changes...)
	}

	return nil
}

// storeTransactions maps the close meta data into a slice of LedgerTransaction structs, to provide
// a per-transaction view of the data when Read() is called.
func (dblrc *DBLedgerReader) storeTransactions(lcm ledgerbackend.LedgerCloseMeta) {
	for i := range lcm.TransactionEnvelope {
		dblrc.transactions = append(dblrc.transactions, LedgerTransaction{
			Index:      uint32(i + 1), // Transactions start at '1'
			Envelope:   lcm.TransactionEnvelope[i],
			Result:     lcm.TransactionResult[i],
			Meta:       lcm.TransactionMeta[i],
			FeeChanges: lcm.TransactionFeeChanges[i],
		})
	}
}
