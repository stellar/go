package services

import (
	"context"
	"fmt"
	"io"

	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/toid"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

type LightHorizon struct {
	Operations   OperationsService
	Transactions TransactionsService
}

type TransactionsService struct {
	TransactionRepository,
	Archive archive.Archive
	IndexStore index.Store
	Passphrase string
}

type OperationsService struct {
	OperationsRepository,
	Archive archive.Archive
	IndexStore index.Store
	Passphrase string
}

type OperationsRepository interface {
	GetOperationsByAccount(ctx context.Context, cursor int64, limit int64, accountId string) ([]common.Operation, error)
}

type TransactionRepository interface {
	GetTransactionsByAccount(ctx context.Context, cursor int64, limit int64, accountId string) ([]common.Transaction, error)
}

func (os *OperationsService) GetOperationsByAccount(ctx context.Context, cursor int64, limit int64, accountId string) ([]common.Operation, error) {
	ops := []common.Operation{}
	// Skip the cursor ahead to the next active checkpoint for this account
	nextCheckpoint, err := getAccountNextCheckpointCursor(accountId, cursor, os.IndexStore)
	if err != nil {
		if err == io.EOF {
			return ops, nil
		}
		return ops, err
	}
	log.Debugf("Searching ops by account %v starting at checkpoint cursor %v", accountId, nextCheckpoint)

	for {
		startingCheckPointLedger := cursorLedger(nextCheckpoint)
		ledgerSequence := startingCheckPointLedger

		for (ledgerSequence - startingCheckPointLedger) < 64 {
			ledger, ledgerErr := os.Archive.GetLedger(ctx, ledgerSequence)
			if ledgerErr != nil {
				return nil, errors.Wrapf(ledgerErr, "ledger export state is out of sync, missing ledger %v from checkpoint %v", ledgerSequence, ledgerSequence/64)
			}

			reader, readerErr := os.Archive.NewLedgerTransactionReaderFromLedgerCloseMeta(os.Passphrase, ledger)
			if readerErr != nil {
				return nil, readerErr
			}

			transactionOrder := int32(0)
			for {
				tx, readErr := reader.Read()
				if readErr != nil {
					if readErr == io.EOF {
						break
					}
					return nil, readErr
				}

				transactionOrder++
				participants, participantErr := os.Archive.GetTransactionParticipants(tx)
				if participantErr != nil {
					return nil, participantErr
				}

				if _, found := participants[accountId]; found {
					for operationOrder, op := range tx.Envelope.Operations() {
						opParticipants, opParticipantErr := os.Archive.GetOperationParticipants(tx, op, operationOrder+1)
						if opParticipantErr != nil {
							return nil, opParticipantErr
						}
						if _, foundInOp := opParticipants[accountId]; foundInOp {
							ops = append(ops, common.Operation{
								TransactionEnvelope: &tx.Envelope,
								TransactionResult:   &tx.Result.Result,
								LedgerHeader:        &ledger.V0.LedgerHeader.Header,
								TxIndex:             int32(transactionOrder),
								OpIndex:             int32(operationOrder + 1),
							})
							if int64(len(ops)) == limit {
								return ops, nil
							}
						}
					}
				}

				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
			}
			ledgerSequence++
		}

		nextCheckpoint, err = getAccountNextCheckpointCursor(accountId, nextCheckpoint, os.IndexStore)
		if err != nil {
			if err == io.EOF {
				return ops, nil
			}
			return ops, err
		}
	}
}

func (ts *TransactionsService) GetTransactionsByAccount(ctx context.Context, cursor int64, limit int64, accountId string) ([]common.Transaction, error) {
	txs := []common.Transaction{}
	// Skip the cursor ahead to the next active checkpoint for this account
	nextCheckpoint, err := getAccountNextCheckpointCursor(accountId, cursor, ts.IndexStore)
	if err != nil {
		if err == io.EOF {
			return txs, nil
		}
		return txs, err
	}
	log.Debugf("Searching tx by account %v starting at checkpoint cursor %v", accountId, nextCheckpoint)

	for {
		startingCheckPointLedger := cursorLedger(nextCheckpoint)
		ledgerSequence := startingCheckPointLedger

		for (ledgerSequence - startingCheckPointLedger) < 64 {
			ledger, ledgerErr := ts.Archive.GetLedger(ctx, ledgerSequence)
			if ledgerErr != nil {
				return nil, errors.Wrapf(ledgerErr, "ledger export state is out of sync, missing ledger %v from checkpoint %v", ledgerSequence, ledgerSequence/64)
			}

			reader, readerErr := ts.Archive.NewLedgerTransactionReaderFromLedgerCloseMeta(ts.Passphrase, ledger)
			if readerErr != nil {
				return nil, readerErr
			}

			transactionOrder := int32(0)
			for {
				tx, readErr := reader.Read()
				if readErr != nil {
					if readErr == io.EOF {
						break
					}
					return nil, readErr
				}

				transactionOrder++
				participants, participantErr := ts.Archive.GetTransactionParticipants(tx)
				if participantErr != nil {
					return nil, participantErr
				}

				if _, found := participants[accountId]; found {
					txs = append(txs, common.Transaction{
						TransactionEnvelope: &tx.Envelope,
						TransactionResult:   &tx.Result.Result,
						LedgerHeader:        &ledger.V0.LedgerHeader.Header,
						TxIndex:             int32(transactionOrder),
					})
					if int64(len(txs)) == limit {
						return txs, nil
					}
				}

				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
			}
			ledgerSequence++
		}

		nextCheckpoint, err = getAccountNextCheckpointCursor(accountId, nextCheckpoint, ts.IndexStore)
		if err != nil {
			if err == io.EOF {
				return txs, nil
			}
			return txs, err
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
