package services

import (
	"context"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

type OperationService interface {
	GetOperationsByAccount(ctx context.Context,
		cursor int64, limit uint64,
		accountId string,
	) ([]common.Operation, error)
}

type OperationRepository struct {
	OperationService
	Config Config
}

func (or *OperationRepository) GetOperationsByAccount(ctx context.Context,
	cursor int64, limit uint64,
	accountId string,
) ([]common.Operation, error) {
	ops := []common.Operation{}

	opsCallback := func(tx archive.LedgerTransaction, ledgerHeader *xdr.LedgerHeader) (bool, error) {
		for operationOrder, op := range tx.Envelope.Operations() {
			opParticipants, err := or.Config.Archive.GetOperationParticipants(tx, op, operationOrder)
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

	err := searchAccountTransactions(ctx, cursor, accountId, or.Config, opsCallback)
	if age := operationsResponseAgeSeconds(ops); age >= 0 {
		or.Config.Metrics.ResponseAgeHistogram.With(prometheus.Labels{
			"request":    "GetOperationsByAccount",
			"successful": strconv.FormatBool(err == nil),
		}).Observe(age)
	}

	return ops, err
}

func operationsResponseAgeSeconds(ops []common.Operation) float64 {
	if len(ops) == 0 {
		return -1
	}

	oldest := ops[0].LedgerHeader.ScpValue.CloseTime
	for i := 1; i < len(ops); i++ {
		if closeTime := ops[i].LedgerHeader.ScpValue.CloseTime; closeTime < oldest {
			oldest = closeTime
		}
	}

	lastCloseTime := time.Unix(int64(oldest), 0).UTC()
	now := time.Now().UTC()
	if now.Before(lastCloseTime) {
		log.Errorf("current time %v is before oldest operation close time %v", now, lastCloseTime)
		return -1
	}
	return now.Sub(lastCloseTime).Seconds()
}

var _ OperationService = (*OperationRepository)(nil) // ensure conformity to the interface
