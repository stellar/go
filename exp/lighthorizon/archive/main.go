package archive

import (
	"io"

	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/network"
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
const checkpointsToLookup = 1

type Wrapper struct {
	*historyarchive.Archive
}

func (a *Wrapper) GetOperations(cursor int64, limit int64) ([]common.Operation, error) {
	parsedID := toid.Parse(cursor)
	ledgerSequence := uint32(parsedID.LedgerSequence)

	log.Debugf("Searching op %d", cursor)
	log.Debugf("Getting ledgers starting at %d", ledgerSequence)

	ops := []common.Operation{}
	appending := false

	ledgers, err := a.Archive.GetLedgers(ledgerSequence, ledgerSequence+64*checkpointsToLookup)
	if err != nil {
		return nil, err
	}

	for {
		log.Debugf("Checking ledger %d", ledgerSequence)
		ledger, ok := ledgers[ledgerSequence]
		if !ok {
			return nil, errors.Errorf("could not reach limit in %d checkpoints (ledger not found)", checkpointsToLookup)
		}

		resultMeta := make([]xdr.TransactionResultMeta, len(ledger.TransactionResult.TxResultSet.Results))
		for i, result := range ledger.TransactionResult.TxResultSet.Results {
			resultMeta[i].Result = result
		}

		closeMeta := xdr.LedgerCloseMeta{
			V0: &xdr.LedgerCloseMetaV0{
				LedgerHeader: ledger.Header,
				TxSet:        ledger.Transaction.TxSet,
				TxProcessing: resultMeta,
			},
		}

		reader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(network.PublicNetworkPassphrase, closeMeta)
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

			for operationOrder, _ := range tx.Envelope.Operations() {
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
						LedgerHeader:        &ledger.Header.Header,
						Index:               int32(operationOrder),
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

func (a *Wrapper) getTxReaderForSingleLedgerFromArchive(ledgerSequence uint32) (*ingest.LedgerTransactionReader, error) {
	ledgers, err := a.Archive.GetLedgers(ledgerSequence, ledgerSequence)
	if err != nil {
		return nil, err
	}

	ledger, ok := ledgers[ledgerSequence]
	if !ok {
		return nil, errors.New("ledger not found")
	}

	resultMeta := make([]xdr.TransactionResultMeta, len(ledger.TransactionResult.TxResultSet.Results))
	for i, result := range ledger.TransactionResult.TxResultSet.Results {
		resultMeta[i].Result = result
	}

	closeMeta := xdr.LedgerCloseMeta{
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: ledger.Header,
			TxSet:        ledger.Transaction.TxSet,
			TxProcessing: resultMeta,
		},
	}

	reader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(network.PublicNetworkPassphrase, closeMeta)
	if err != nil {
		return nil, err
	}

	return reader, nil
}
