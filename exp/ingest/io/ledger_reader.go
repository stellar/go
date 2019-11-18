package io

import (
	"io"
	"sync"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// DBLedgerReader is a database-backed implementation of the io.LedgerReader interface.
// Use NewDBLedgerReader to create a new instance.
type DBLedgerReader struct {
	sequence                uint32
	closeTime               int64
	backend                 ledgerbackend.LedgerBackend
	header                  xdr.LedgerHeaderHistoryEntry
	transactions            []LedgerTransaction
	upgradeChanges          []Change
	opCount                 int
	successTxCount          int
	failedTxCount           int
	readMutex               sync.Mutex
	readIdx                 int
	upgradeReadIdx          int
	readUpgradeChangeCalled bool
	ignoreUpgradeChanges    bool
}

// Ensure DBLedgerReader implements LedgerReader
var _ LedgerReader = (*DBLedgerReader)(nil)

// NewDBLedgerReader is a factory method for LedgerReader.
func NewDBLedgerReader(sequence uint32, backend ledgerbackend.LedgerBackend) (*DBLedgerReader, error) {
	reader := &DBLedgerReader{
		sequence: sequence,
		backend:  backend,
	}

	err := reader.init()
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
	return dblrc.header
}

// Read returns the next transaction in the ledger, ordered by tx number, each time it is called. When there
// are no more transactions to return, an EOF error is returned.
func (dblrc *DBLedgerReader) Read() (LedgerTransaction, error) {
	// Protect all accesses to dblrc.readIdx
	dblrc.readMutex.Lock()
	defer dblrc.readMutex.Unlock()

	if dblrc.readIdx < len(dblrc.transactions) {
		dblrc.readIdx++
		return dblrc.transactions[dblrc.readIdx-1], nil
	}
	return LedgerTransaction{}, io.EOF
}

// ReadUpgradeChange returns the next upgrade change in the ledger, each time it
// is called. When there are no more upgrades to return, an EOF error is returned.
func (dblrc *DBLedgerReader) ReadUpgradeChange() (Change, error) {
	// Protect all accesses to dblrc.upgradeReadIdx
	dblrc.readMutex.Lock()
	defer dblrc.readMutex.Unlock()
	dblrc.readUpgradeChangeCalled = true

	if dblrc.upgradeReadIdx < len(dblrc.upgradeChanges) {
		dblrc.upgradeReadIdx++
		return dblrc.upgradeChanges[dblrc.upgradeReadIdx-1], nil
	}
	return Change{}, io.EOF
}

// GetUpgradeChanges returns all ledger upgrade changes.
func (dblrc *DBLedgerReader) GetUpgradeChanges() []Change {
	return dblrc.upgradeChanges
}

func (dblrc *DBLedgerReader) IgnoreUpgradeChanges() {
	dblrc.ignoreUpgradeChanges = true
}

// Close moves the read pointer so that subsequent calls to Read() will return EOF.
func (dblrc *DBLedgerReader) Close() error {
	dblrc.readMutex.Lock()
	dblrc.readIdx = len(dblrc.transactions)
	if !dblrc.ignoreUpgradeChanges &&
		(!dblrc.readUpgradeChangeCalled || dblrc.upgradeReadIdx != len(dblrc.upgradeChanges)) {
		return errors.New("Ledger upgrade changes not read! Use ReadUpgradeChange() method.")
	}
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
	dblrc.closeTime = ledgerCloseMeta.CloseTime

	dblrc.storeTransactions(ledgerCloseMeta)

	for _, upgradeChanges := range ledgerCloseMeta.UpgradesMeta {
		changes := getChangesFromLedgerEntryChanges(upgradeChanges)
		dblrc.upgradeChanges = append(dblrc.upgradeChanges, changes...)
	}

	return nil
}

// SuccessfulLedgerOperationCount returns the count of operations in the current ledger
func (dblrc *DBLedgerReader) SuccessfulLedgerOperationCount() int {
	return dblrc.opCount
}

// SuccessfulTransactionCount returns the count of transactions in the current
// ledger that succeeded.
func (dblrc *DBLedgerReader) SuccessfulTransactionCount() int {
	return dblrc.successTxCount
}

// FailedTransactionCount returns the count of transactions in the current ledger that failed.
func (dblrc *DBLedgerReader) FailedTransactionCount() int {
	return dblrc.FailedTransactionCount()
}

// CloseTime returns the time at which the ledger was applied to Stellar Core.
func (dblrc *DBLedgerReader) CloseTime() int64 {
	return dblrc.closeTime
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

		if lcm.TransactionResult[i].Result.Result.Code == xdr.TransactionResultCodeTxSuccess {
			dblrc.successTxCount++
			dblrc.opCount += len(lcm.TransactionEnvelope[i].Tx.Operations)
		} else {
			dblrc.failedTxCount++
		}
	}
}
