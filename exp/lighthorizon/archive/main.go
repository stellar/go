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
	GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error)
	Close() error
	NewLedgerTransactionReaderFromLedgerCloseMeta(networkPassphrase string, ledgerCloseMeta xdr.LedgerCloseMeta) (LedgerTransactionReader, error)
	GetTransactionParticipants(transaction LedgerTransaction) (map[string]struct{}, error)
	GetOperationParticipants(transaction LedgerTransaction, operation xdr.Operation, opIndex int) (map[string]struct{}, error)
}
