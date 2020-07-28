package actions

import (
	"net/http"

	"github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
)

// EffectsQuery query struct for effects end-points
type EffectsQuery struct {
	AccountID   string `schema:"account_id" valid:"accountID,optional"`
	OperationID uint64 `schema:"op_id" valid:"-"`
	TxHash      string `schema:"tx_id" valid:"transactionHash,optional"`
	LedgerID    uint32 `schema:"ledger_id" valid:"-"`
}

// Validate runs extra validations on query parameters
func (qp EffectsQuery) Validate() error {
	count, err := countNonEmpty(
		qp.AccountID,
		qp.OperationID,
		qp.TxHash,
		qp.LedgerID,
	)

	if err != nil {
		return problem.BadRequest
	}

	if count > 1 {
		return problem.MakeInvalidFieldProblem(
			"filters",
			errors.New("Use a single filter for effects, you can only use one of account_id, op_id, tx_id or ledger_id"),
		)
	}
	return nil
}

type GetEffectsHandler struct{}

func (handler GetEffectsHandler) GetResourcePage(w HeaderWriter, r *http.Request) ([]hal.Pageable, error) {
	pq, err := GetPageQuery(r)
	if err != nil {
		return nil, err
	}

	err = validateCursorWithinHistory(pq)
	if err != nil {
		return nil, err
	}

	qp := EffectsQuery{}
	err = getParams(&qp, r)
	if err != nil {
		return nil, err
	}

	historyQ, err := context.HistoryQFromRequest(r)
	if err != nil {
		return nil, err
	}

	records, err := loadEffectRecords(historyQ, qp.AccountID, int64(qp.OperationID), qp.TxHash, qp.LedgerID, pq)
	if err != nil {
		return nil, errors.Wrap(err, "loading transaction records")
	}

	ledgers, err := loadEffectLedgers(historyQ, records)
	if err != nil {
		return nil, errors.Wrap(err, "loading ledgers")
	}

	var result []hal.Pageable
	for _, record := range records {
		effect, err := resourceadapter.NewEffect(r.Context(), record, ledgers[record.LedgerSequence()])
		if err != nil {
			return nil, errors.Wrap(err, "could not create effect")
		}
		result = append(result, effect)
	}

	return result, nil
}

func loadEffectRecords(hq *history.Q, accountID string, operationID int64, transactionHash string, ledgerID uint32,
	pq db2.PageQuery) ([]history.Effect, error) {
	effects := hq.Effects()

	switch {
	case accountID != "":
		effects.ForAccount(accountID)
	case ledgerID > 0:
		effects.ForLedger(int32(ledgerID))
	case operationID > 0:
		effects.ForOperation(operationID)
	case transactionHash != "":
		effects.ForTransaction(transactionHash)
	}

	var result []history.Effect
	err := effects.Page(pq).Select(&result)

	return result, err
}

func loadEffectLedgers(hq *history.Q, effects []history.Effect) (map[int32]history.Ledger, error) {
	ledgers := &history.LedgerCache{}

	for _, e := range effects {
		ledgers.Queue(e.LedgerSequence())
	}

	if err := ledgers.Load(hq); err != nil {
		return nil, err
	}
	return ledgers.Records, nil
}
