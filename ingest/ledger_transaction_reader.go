package ingest

import (
	"context"
	"encoding/hex"
	"io"

	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LedgerTransactionReader reads transactions for a given ledger sequence from a
// backend. Use NewTransactionReader to create a new instance.
type LedgerTransactionReader struct {
	lcm             xdr.LedgerCloseMeta                  // read-only
	envelopesByHash map[xdr.Hash]xdr.TransactionEnvelope // set once

	txByIdx map[int]LedgerTransaction // cache
	readIdx int                       // tracks iteration & seeking
}

// NewLedgerTransactionReader creates a new TransactionReader instance. Note
// that TransactionReader is not thread safe and should not be shared by
// multiple goroutines.
func NewLedgerTransactionReader(
	ctx context.Context,
	backend ledgerbackend.LedgerBackend,
	networkPassphrase string,
	sequence uint32,
) (*LedgerTransactionReader, error) {
	ledgerCloseMeta, err := backend.GetLedger(ctx, sequence)
	if err != nil {
		return nil, errors.Wrap(err, "error getting ledger from the backend")
	}

	return NewLedgerTransactionReaderFromLedgerCloseMeta(networkPassphrase, ledgerCloseMeta)
}

// NewLedgerTransactionReaderFromLedgerCloseMeta creates a new TransactionReader
// instance from xdr.LedgerCloseMeta. Note that TransactionReader is not thread
// safe and should not be shared by multiple goroutines.
func NewLedgerTransactionReaderFromLedgerCloseMeta(
	networkPassphrase string,
	ledgerCloseMeta xdr.LedgerCloseMeta,
) (*LedgerTransactionReader, error) {
	reader := &LedgerTransactionReader{
		lcm:             ledgerCloseMeta,
		txByIdx:         make(map[int]LedgerTransaction, ledgerCloseMeta.CountTransactions()),
		envelopesByHash: make(map[xdr.Hash]xdr.TransactionEnvelope, ledgerCloseMeta.CountTransactions()),
		readIdx:         0,
	}

	if err := reader.storeTransactions(networkPassphrase); err != nil {
		return nil, errors.Wrap(err, "error extracting transactions from ledger close meta")
	}
	return reader, nil
}

// GetSequence returns the sequence number of the ledger data stored by this object.
func (reader *LedgerTransactionReader) GetSequence() uint32 {
	return reader.lcm.LedgerSequence()
}

// GetHeader returns the XDR Header data associated with the stored ledger.
func (reader *LedgerTransactionReader) GetHeader() xdr.LedgerHeaderHistoryEntry {
	return reader.lcm.LedgerHeaderHistoryEntry()
}

// Read returns the next transaction in the ledger, ordered by tx number, each time
// it is called. When there are no more transactions to return, an EOF error is returned.
func (reader *LedgerTransactionReader) Read() (LedgerTransaction, error) {
	i := reader.readIdx
	if i >= reader.lcm.CountTransactions() {
		return LedgerTransaction{}, io.EOF
	}
	reader.readIdx++

	hash := reader.lcm.TransactionHash(i)
	envelope, ok := reader.envelopesByHash[hash]
	if !ok {
		hexHash := hex.EncodeToString(hash[:])
		return LedgerTransaction{}, errors.Errorf("unknown tx hash in LedgerCloseMeta: %v", hexHash)
	}

	// We check the version only if FeeProcessing are non empty because some backends
	// (like HistoryArchiveBackend) do not return meta.
	if reader.lcm.ProtocolVersion() < 10 && reader.lcm.TxApplyProcessing(i).V < 2 &&
		len(reader.lcm.FeeProcessing(i)) > 0 {
		return LedgerTransaction{}, errors.New(
			"TransactionMeta.V=2 is required in protocol version older than version 10. " +
				"Please process ledgers again using the latest stellar-core version.",
		)
	}

	if ledgerTx, ok := reader.txByIdx[i]; ok {
		return ledgerTx, nil
	}
	// generate and cache if not found
	ledgerTx := LedgerTransaction{
		Index:         uint32(i + 1), // Transactions start at '1'
		Envelope:      envelope,
		Result:        reader.lcm.TransactionResultPair(i),
		UnsafeMeta:    reader.lcm.TxApplyProcessing(i),
		FeeChanges:    reader.lcm.FeeProcessing(i),
		LedgerVersion: uint32(reader.lcm.LedgerHeaderHistoryEntry().Header.LedgerVersion),
	}
	reader.txByIdx[i] = ledgerTx
	return ledgerTx, nil
}

// Rewind resets the reader back to the first transaction in the ledger
func (reader *LedgerTransactionReader) Rewind() {
	reader.Seek(0)
}

// Seek sets the reader back to a specific transaction in the ledger
func (reader *LedgerTransactionReader) Seek(index int) error {
	if index >= reader.lcm.CountTransactions() || index < 0 {
		return io.EOF
	}

	reader.readIdx = index
	return nil
}

// storeHashes creates a mapping between hashes and envelopes in order to
// correctly provide a per-transaction view on-the-fly when Read() is called.
func (reader *LedgerTransactionReader) storeTransactions(networkPassphrase string) error {
	// See https://github.com/stellar/go/pull/2720: envelopes in the meta (which
	// just come straight from the agreed-upon transaction set) are not in the
	// same order as the actual list of metas (which are sorted by hash), so we
	// need to hash the envelopes *first* to properly associate them with their
	// metas.
	for i, tx := range reader.lcm.TransactionEnvelopes() {
		hash, err := network.HashTransactionInEnvelope(tx, networkPassphrase)
		if err != nil {
			return errors.Wrapf(err, "could not hash transaction %d in TxSet", i)
		}
		reader.envelopesByHash[xdr.Hash(hash)] = tx
	}

	return nil
}

// Close should be called when reading is finished. This is especially
// helpful when there are still some transactions available so reader can stop
// streaming them.
func (reader *LedgerTransactionReader) Close() error {
	reader.envelopesByHash = nil
	reader.txByIdx = nil
	return nil
}
