package archive

import (
	"context"
	"io"

	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/toid"
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
}

type Wrapper struct {
	Archive
	Passphrase string
}

func (a *Wrapper) GetOperations(ctx context.Context, cursor int64, limit int64) ([]common.Operation, error) {
	parsedID := toid.Parse(cursor)
	ledgerSequence := uint32(parsedID.LedgerSequence)
	if ledgerSequence < 2 {
		ledgerSequence = 2
	}

	log.Debugf("Searching op %d", cursor)
	log.Debugf("Getting ledgers starting at %d", ledgerSequence)

	ops := []common.Operation{}
	appending := false

	for {
		log.Debugf("Checking ledger %d", ledgerSequence)
		ledger, err := a.GetLedger(ctx, ledgerSequence)
		if err != nil {
			return ops, nil
		}

		reader, err := a.NewLedgerTransactionReaderFromLedgerCloseMeta(a.Passphrase, ledger)
		if err != nil {
			return nil, errors.Wrapf(err, "error in ledger %d", ledgerSequence)
		}

		transactionOrder := int32(0)
		for {
			tx, err := reader.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}

			transactionOrder++
			for operationOrder := range tx.Envelope.Operations() {
				currID := toid.New(int32(ledgerSequence), transactionOrder, int32(operationOrder+1)).ToInt64()

				if currID >= cursor {
					appending = true
					if currID == cursor {
						continue
					}
				}

				if appending {
					ops = append(ops, common.Operation{
						TransactionEnvelope: &tx.Envelope,
						TransactionResult:   &tx.Result.Result,
						// TODO: Use a method to get the header
						LedgerHeader: &ledger.V0.LedgerHeader.Header,
						OpIndex:      int32(operationOrder + 1),
						TxIndex:      int32(transactionOrder),
					})
				}

				if int64(len(ops)) == limit {
					return ops, nil
				}
			}
		}

		ledgerSequence++
	}
}

func (a *Wrapper) GetTransactions(ctx context.Context, cursor int64, limit int64) ([]common.Transaction, error) {
	parsedID := toid.Parse(cursor)
	ledgerSequence := uint32(parsedID.LedgerSequence)
	if ledgerSequence < 2 {
		ledgerSequence = 2
	}

	log.Debugf("Searching tx %d starting at", cursor)
	log.Debugf("Getting ledgers starting at %d", ledgerSequence)

	txns := []common.Transaction{}
	appending := false

	for {
		log.Debugf("Checking ledger %d", ledgerSequence)
		ledger, err := a.GetLedger(ctx, ledgerSequence)
		if err != nil {
			// no 'NotFound' distinction on err, treat all as not found.
			return txns, nil
		}

		reader, err := a.NewLedgerTransactionReaderFromLedgerCloseMeta(a.Passphrase, ledger)
		if err != nil {
			return nil, err
		}

		transactionOrder := int32(0)
		for {
			tx, err := reader.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}

			transactionOrder++
			currID := toid.New(int32(ledgerSequence), transactionOrder, 1).ToInt64()

			if currID >= cursor {
				appending = true
				if currID == cursor {
					continue
				}
			}

			if appending {
				txns = append(txns, common.Transaction{
					TransactionEnvelope: &tx.Envelope,
					TransactionResult:   &tx.Result.Result,
					// TODO: Use a method to get the header
					LedgerHeader: &ledger.V0.LedgerHeader.Header,
					TxIndex:      int32(transactionOrder),
				})
			}

			if int64(len(txns)) == limit {
				return txns, nil
			}

			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
		}

		ledgerSequence++
	}
}
