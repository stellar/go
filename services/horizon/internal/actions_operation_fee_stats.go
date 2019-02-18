package horizon

import (
	"fmt"
	"net/http"

	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/operationfeestats"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
)

// This file contains the actions:
//
// OperationFeeStatsAction: stats representing current state of network fees

var _ actions.JSONer = (*OperationFeeStatsAction)(nil)

// OperationFeeStatsAction renders a few useful statistics that describe the
// current state of operation fees on the network.
type OperationFeeStatsAction struct {
	Action
	Min         int64
	Mode        int64
	LastBaseFee int64
	LastLedger  int64
}

// JSON is a method for actions.JSON
func (action *OperationFeeStatsAction) JSON() error {
	if !action.App.config.IngestFailedTransactions {
		// If Horizon is not ingesting failed transaction it does not make sense to display
		// operation fee stats because they will be incorrect.
		p := problem.P{
			Type:   "endpoint_not_available",
			Title:  "Endpoint Not Available",
			Status: http.StatusNotImplemented,
			Detail: "/operation_fee_stats is unavailable when Horizon is not ingesting failed " +
				"transactions. Set `INGEST_FAILED_TRANSACTIONS=true` to start ingesting them.",
		}
		problem.Render(action.R.Context(), action.W, p)
		return nil
	}

	action.Do(
		action.loadRecords,
		func() {
			hal.Render(action.W, map[string]string{
				"min_accepted_fee":     fmt.Sprint(action.Min),
				"mode_accepted_fee":    fmt.Sprint(action.Mode),
				"last_ledger_base_fee": fmt.Sprint(action.LastBaseFee),
				"last_ledger":          fmt.Sprint(action.LastLedger),
			})
		},
	)
	return action.Err
}

func (action *OperationFeeStatsAction) loadRecords() {
	cur := operationfeestats.CurrentState()
	action.Min = cur.Min
	action.Mode = cur.Mode
	action.LastBaseFee = cur.LastBaseFee
	action.LastLedger = cur.LastLedger
}
