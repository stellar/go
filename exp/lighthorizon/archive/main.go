package archive

import (
	"context"

	"github.com/stellar/go/support/collections/set"
	"github.com/stellar/go/xdr"
)

// checkpointsToLookup defines a number of checkpoints to check when filling
// a list of objects up to a requested limit. In the old ledgers in pubnet
// many ledgers or even checkpoints were empty. This means that when building
// a list of 200 operations ex. starting at first ledger, lighthorizon will
// have to download many ledgers until it's able to fill the list completely.
// This can be solved by keeping an index/list of empty ledgers.
// TODO: make this configurable.
//
//lint:ignore U1000 Ignore unused temporarily
const checkpointsToLookup = 1

// LightHorizon data model
type LedgerTransaction struct {
	Index      uint32
	Envelope   xdr.TransactionEnvelope
	Result     xdr.TransactionResultPair
	FeeChanges xdr.LedgerEntryChanges
	UnsafeMeta xdr.TransactionMeta
}

type LedgerTransactionReader interface {
	Read() (LedgerTransaction, error)
}

// Archive here only has the methods LightHorizon cares about, to make caching/wrapping easier
type Archive interface {

	// GetLedger - takes a caller context and a sequence number and returns the meta data
	// for the ledger corresponding to the sequence number. If there is any error, it will
	// return nil and the error.
	GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error)

	// Close - will release any resources used for this archive instance and should be
	// called at end of usage of archive.
	Close() error

	// NewLedgerTransactionReaderFromLedgerCloseMeta - takes the passphrase for the blockchain network
	// and the LedgerCloseMeta(meta data) and returns a reader that can be used to obtain a LedgerTransaction model
	// from the meta data. If there is any error, it will return nil and the error.
	NewLedgerTransactionReaderFromLedgerCloseMeta(networkPassphrase string, ledgerCloseMeta xdr.LedgerCloseMeta) (LedgerTransactionReader, error)

	// GetTransactionParticipants - takes a LedgerTransaction and returns a set of all
	// participants(accounts) in the transaction. If there is any error, it will return nil and the error.
	GetTransactionParticipants(tx LedgerTransaction) (set.Set[string], error)

	// GetOperationParticipants - takes a LedgerTransaction, the Operation within the transaction, and
	// the 0 based index of the operation within the transaction. It will return a set of all participants(accounts)
	// in the operation. If there is any error, it will return nil and the error.
	GetOperationParticipants(tx LedgerTransaction, op xdr.Operation, opIndex int) (set.Set[string], error)
}
