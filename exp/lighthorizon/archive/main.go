package archive

import (
	"context"
	"io"

	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/network"
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
const checkpointsToLookup = 1

// Archive here only has the methods we care about, to make caching/wrapping easier
type Archive interface {
	GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error)
}

type Wrapper struct {
	Archive
}

func (a *Wrapper) GetOperations(cursor int64, limit int64) ([]common.Operation, error) {
	parsedID := toid.Parse(cursor)
	ledgerSequence := uint32(parsedID.LedgerSequence)

	log.Debugf("Searching op %d", cursor)
	log.Debugf("Getting ledgers starting at %d", ledgerSequence)

	ops := []common.Operation{}
	appending := false
	ctx := context.Background()

	for {
		log.Debugf("Checking ledger %d", ledgerSequence)
		ledger, err := a.GetLedger(ctx, ledgerSequence)
		if err != nil {
			return nil, err
		}

		reader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(network.PublicNetworkPassphrase, ledger)
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

			for operationOrder := range tx.Envelope.Operations() {
				currID := toid.New(int32(ledgerSequence), transactionOrder+1, int32(operationOrder+1)).ToInt64()

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
						OpIndex:      int32(operationOrder),
						TxIndex:      int32(transactionOrder),
					})
				}

				if int64(len(ops)) == limit {
					return ops, nil
				}
			}

			transactionOrder++
		}

		ledgerSequence++
	}
}

func (a *Wrapper) GetTransactions(cursor int64, limit int64) ([]common.Transaction, error) {
	parsedID := toid.Parse(cursor)
	ledgerSequence := uint32(parsedID.LedgerSequence)

	log.Debugf("Searching tx %d", cursor)
	log.Debugf("Getting ledgers starting at %d", ledgerSequence)

	txns := []common.Transaction{}
	appending := false

	ctx := context.Background()

	for {
		log.Debugf("Checking ledger %d", ledgerSequence)
		ledger, err := a.GetLedger(ctx, ledgerSequence)
		if err != nil {
			return nil, err
		}

		reader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(network.PublicNetworkPassphrase, ledger)
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

			currID := toid.New(int32(ledgerSequence), transactionOrder+1, 1).ToInt64()

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

			transactionOrder++
		}

		ledgerSequence++
	}
}
