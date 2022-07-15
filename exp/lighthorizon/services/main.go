package services

import (
	"context"
	"fmt"
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
	GetOperationsByAccount(ctx context.Context, cursor int64, limit int64, accountId string) ([]common.Operation, error)
}

type TransactionRepository interface {
	GetTransactionsByAccount(ctx context.Context, cursor int64, limit int64, accountId string) ([]common.Transaction, error)
}

type searchCallback func(archive.LedgerTransaction, *xdr.LedgerHeader) (finished bool, err error)

func (os *OperationsService) GetOperationsByAccount(ctx context.Context, cursor int64, limit int64, accountId string) ([]common.Operation, error) {
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
				if int64(len(ops)) == limit {
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

func (ts *TransactionsService) GetTransactionsByAccount(ctx context.Context, cursor int64, limit int64, accountId string) ([]common.Transaction, error) {
	txs := []common.Transaction{}

	txsCallback := func(tx archive.LedgerTransaction, ledgerHeader *xdr.LedgerHeader) (bool, error) {
		txs = append(txs, common.Transaction{
			TransactionEnvelope: &tx.Envelope,
			TransactionResult:   &tx.Result.Result,
			LedgerHeader:        ledgerHeader,
			TxIndex:             int32(tx.Index),
		})
		if int64(len(txs)) == limit {
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
	// Skip the cursor ahead to the next active checkpoint for this account
	nextCheckpoint, err := getAccountNextCheckpointCursor(accountId, cursor, config.IndexStore)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	log.Debugf("Searching index by account %v starting at checkpoint cursor %v", accountId, nextCheckpoint)

	for {
		startingCheckPointLedger := cursorLedger(nextCheckpoint)
		ledgerSequence := startingCheckPointLedger

		for (ledgerSequence - startingCheckPointLedger) < 64 {
			ledger, ledgerErr := config.Archive.GetLedger(ctx, ledgerSequence)
			if ledgerErr != nil {
				return errors.Wrapf(ledgerErr, "ledger export state is out of sync, missing ledger %v from checkpoint %v", ledgerSequence, ledgerSequence/64)
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
					} else {
						if finished {
							return nil
						}
					}
				}

				if ctx.Err() != nil {
					return ctx.Err()
				}
			}
			ledgerSequence++
		}

		nextCheckpoint, err = getAccountNextCheckpointCursor(accountId, nextCheckpoint, config.IndexStore)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func getAccountNextCheckpointCursor(accountId string, cursor int64, store index.Store) (int64, error) {
	var checkpoint uint32
	checkpoint, err := store.NextActive(accountId, "all/all", uint32(toid.Parse(cursor).LedgerSequence/64))
	if err != nil {
		return 0, err
	}
	ledger := int32(checkpoint * 64)
	if ledger < 0 {
		// Check we don't overflow going from uint32 -> int32
		return 0, fmt.Errorf("overflowed ledger number")
	}
	cursor = toid.New(ledger, 1, 1).ToInt64()

	return cursor, nil
}

func cursorLedger(cursor int64) uint32 {
	parsedID := toid.Parse(cursor)
	ledgerSequence := uint32(parsedID.LedgerSequence)
	if ledgerSequence < 2 {
		ledgerSequence = 2
	}
	return ledgerSequence
}
