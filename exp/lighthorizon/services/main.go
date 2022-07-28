package services

import (
	"context"
	"io"

	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/xdr"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

const (
	allTransactionsIndex = "all/all"
	allPaymentsIndex     = "all/payments"
)

var (
	checkpointManager = historyarchive.NewCheckpointManager(0)
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

// searchCallback is a generic way for any endpoint to process a transaction and
// its corresponding ledger. It should return whether or not we should stop
// processing (e.g. when a limit is reached) and any error that occurred.
type searchCallback func(archive.LedgerTransaction, *xdr.LedgerHeader) (finished bool, err error)

func (os *OperationsService) GetOperationsByAccount(ctx context.Context,
	cursor int64, limit uint64,
	accountId string,
) ([]common.Operation, error) {
	ops := []common.Operation{}

	opsCallback := func(tx archive.LedgerTransaction, ledgerHeader *xdr.LedgerHeader) (bool, error) {
		for operationOrder, op := range tx.Envelope.Operations() {
			opParticipants, err := os.Config.Archive.GetOperationParticipants(tx, op, operationOrder)
			if err != nil {
				return false, err
			}

			if _, foundInOp := opParticipants[accountId]; foundInOp {
				ops = append(ops, common.Operation{
					TransactionEnvelope: &tx.Envelope,
					TransactionResult:   &tx.Result.Result,
					LedgerHeader:        ledgerHeader,
					TxIndex:             int32(tx.Index),
					OpIndex:             int32(operationOrder),
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

func (ts *TransactionsService) GetTransactionsByAccount(ctx context.Context,
	cursor int64, limit uint64,
	accountId string,
) ([]common.Transaction, error) {
	txs := []common.Transaction{}

	txsCallback := func(tx archive.LedgerTransaction, ledgerHeader *xdr.LedgerHeader) (bool, error) {
		txs = append(txs, common.Transaction{
			TransactionEnvelope: &tx.Envelope,
			TransactionResult:   &tx.Result.Result,
			LedgerHeader:        ledgerHeader,
			TxIndex:             int32(tx.Index),
			NetworkPassphrase:   ts.Config.Passphrase,
		})

		return uint64(len(txs)) == limit, nil
	}

	if err := searchTxByAccount(ctx, cursor, accountId, ts.Config, txsCallback); err != nil {
		return nil, err
	}
	return txs, nil
}

func searchTxByAccount(ctx context.Context, cursor int64, accountId string, config Config, callback searchCallback) error {
	cursorMgr := NewCursorManagerForAccountActivity(config.IndexStore, accountId)
	cursor, err := cursorMgr.Begin(cursor)
	if err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}

	nextLedger := getLedgerFromCursor(cursor)
	log.Debugf("Searching %s for account %s starting at ledger %d",
		allTransactionsIndex, accountId, nextLedger)

	for {
		ledger, ledgerErr := config.Archive.GetLedger(ctx, nextLedger)
		if ledgerErr != nil {
			return errors.Wrapf(ledgerErr,
				"ledger export state is out of sync at ledger %d", nextLedger)
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
				finished, callBackErr := callback(tx, &ledger.V0.LedgerHeader.Header)
				if finished || callBackErr != nil {
					return callBackErr
				}
			}

			if ctx.Err() != nil {
				return ctx.Err()
			}
		}

		cursor, err = cursorMgr.Advance()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		nextLedger = getLedgerFromCursor(cursor)
	}
}
