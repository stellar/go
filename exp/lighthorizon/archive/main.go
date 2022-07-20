package archive

import (
	"context"
	"github.com/stellar/go/xdr"
)

// checkpointsToLookup defines a number of checkpoints to check when filling
// a list of objects up to a requested limit. In the old ledgers in pubnet
// many ledgers or even checkpoints were empty. This means that when building
// a list of 200 operations ex. starting at first ledger, lighthorizon will
// have to download many ledgers until it's able to fill the list completely.
// This can be solved by keeping an index/list of empty ledgers.
// TODO: make this configurable.
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

	//GetLedger - retreive a ledger's meta data
	//
	//ctx               - the caller's request context
	//ledgerCloseMeta   - the sequence number of ledger to fetch
	//
	//returns error or meta data for requested ledger
	GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error)

	// Close - releases any resources used for this archive instance.
	Close() error

	// NewLedgerTransactionReaderFromLedgerCloseMeta - get a reader for ledger meta data
	//
	// networkPassphrase - the network passphrase
	// ledgerCloseMeta   - the meta data for a ledger
	//
	// returns error or LedgerTransactionReader
	NewLedgerTransactionReaderFromLedgerCloseMeta(networkPassphrase string, ledgerCloseMeta xdr.LedgerCloseMeta) (LedgerTransactionReader, error)

	// GetTransactionParticipants - get set of all participants(accounts) in a transaction
	//
	// transaction - the ledger transaction
	//
	// returns error or map with keys of participant account id's and value of empty struct
	GetTransactionParticipants(transaction LedgerTransaction) (map[string]struct{}, error)

	// GetOperationParticipants - get set of all participants(accounts) in a operation
	//
	// transaction - the ledger transaction
	// operation   - the operation within this transaction
	// opIndex     - the 0 based index of the operation within the transaction
	//
	// returns error or map with keys of participant account id's and value of empty struct
	GetOperationParticipants(transaction LedgerTransaction, operation xdr.Operation, opIndex int) (map[string]struct{}, error)
}
