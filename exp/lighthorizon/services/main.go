package services

import (
	"context"
	"io"

	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

type LightHorizon struct {
	Operations   OperationsService
	Transactions TransactionsService
}

type Config struct {
	Archive    archive.Archive
	IndexStore index.Store
	Passphrase string
}

type TransactionsService struct {
	TransactionRepository
	Config Config
}

type OperationsService struct {
	OperationsRepository,
	Config Config
}

type OperationsRepository interface {
	GetOperationsByAccount(ctx context.Context, cursor int64, limit uint64, accountId string) ([]common.Operation, error)
}

type TransactionRepository interface {
	GetTransactionsByAccount(ctx context.Context, cursor int64, limit uint64, accountId string) ([]common.Transaction, error)
}

type searchCallback func(archive.LedgerTransaction, *xdr.LedgerHeader) (finished bool, err error)

func (os *OperationsService) GetOperationsByAccount(ctx context.Context, cursor int64, limit uint64, accountId string) ([]common.Operation, error) {
	ops := []common.Operation{}
	opsCallback := func(tx archive.LedgerTransaction, ledgerHeader *xdr.LedgerHeader) (bool, error) {
		for operationOrder, op := range tx.Envelope.Operations() {
			opParticipants, opParticipantErr := os.Config.Archive.GetOperationParticipants(tx, op, operationOrder+1)
			if opParticipantErr != nil {
				return false, opParticipantErr
			}
			if _, foundInOp := opParticipants[accountId]; foundInOp {
				ops = append(ops, common.Operation{
					TransactionEnvelope: &tx.Envelope,
					TransactionResult:   &tx.Result.Result,
					LedgerHeader:        ledgerHeader,
					TxIndex:             int32(tx.Index),
					OpIndex:             int32(operationOrder + 1),
				})
				if uint64(len(ops)) == limit {
					return true, nil
				}
			}
		}
		return false, nil
	}

	if err := searchTxByAccount(ctx, cursor, accountId, os.Config, opsCallback); err != nil {
		return nil, err
	}

	return ops, nil
}

func (ts *TransactionsService) GetTransactionsByAccount(ctx context.Context, cursor int64, limit uint64, accountId string) ([]common.Transaction, error) {
	txs := []common.Transaction{}

	txsCallback := func(tx archive.LedgerTransaction, ledgerHeader *xdr.LedgerHeader) (bool, error) {
		txs = append(txs, common.Transaction{
			TransactionEnvelope: &tx.Envelope,
			TransactionResult:   &tx.Result.Result,
			LedgerHeader:        ledgerHeader,
			TxIndex:             int32(tx.Index),
		})
		if uint64(len(txs)) == limit {
			return true, nil
		}
		return false, nil
	}

	if err := searchTxByAccount(ctx, cursor, accountId, ts.Config, txsCallback); err != nil {
		return nil, err
	}
	return txs, nil
}

func searchTxByAccount(ctx context.Context, cursor int64, accountId string, config Config, callback searchCallback) error {
	nextLedger, err := getAccountNextLedgerCursor(accountId, cursor, config.IndexStore)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	log.Debugf("Searching index by account %v starting at cursor %v", accountId, nextLedger)

	for {
		ledger, ledgerErr := config.Archive.GetLedger(ctx, uint32(nextLedger))
		if ledgerErr != nil {
			return errors.Wrapf(ledgerErr, "ledger export state is out of sync, missing ledger %v from checkpoint %v", nextLedger, nextLedger/64)
		}

		reader, readerErr := config.Archive.NewLedgerTransactionReaderFromLedgerCloseMeta(config.Passphrase, ledger)
		if readerErr != nil {
			return readerErr
		}

		for {
			tx, readErr := reader.Read()
			if readErr != nil {
				if readErr == io.EOF {
					break
				}
				return readErr
			}

			participants, participantErr := config.Archive.GetTransactionParticipants(tx)
			if participantErr != nil {
				return participantErr
			}

			if _, found := participants[accountId]; found {
				if finished, callBackErr := callback(tx, &ledger.V0.LedgerHeader.Header); callBackErr != nil {
					return callBackErr
				} else if finished {
					return nil
				}
			}

			if ctx.Err() != nil {
				return ctx.Err()
			}
		}
		nextCursor := toid.New(int32(nextLedger), 1, 1).ToInt64()

		nextLedger, err = getAccountNextLedgerCursor(accountId, nextCursor, config.IndexStore)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

// this deals in ledgers but adapts to the index model, which is currently keyed by checkpoint for now
func getAccountNextLedgerCursor(accountId string, cursor int64, store index.Store) (uint64, error) {
	nextLedger := toid.Parse(cursor).LedgerSequence + 1
	cursorCheckpoint := uint32(nextLedger / 64)
	queryCheckpoint := cursorCheckpoint
	if queryCheckpoint > 0 {
		queryCheckpoint = queryCheckpoint - 1
	}
	nextCheckpoint, err := store.NextActive(accountId, "all/all", queryCheckpoint)

	if err != nil {
		return 0, err
	}

	if nextCheckpoint == cursorCheckpoint {
		// querying for a ledger with a derived checkpoint that was active for account in the index
		// but the ledger is in the same checkpoint as requested cursor is already in
		// so just return the requested cursor as-is
		return uint64(nextLedger), nil
	}

	// return the first ledger of the next checkpoint that had account activity after cursor
	return uint64(nextCheckpoint * 64), nil
}
