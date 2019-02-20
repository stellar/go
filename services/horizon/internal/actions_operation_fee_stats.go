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
	P10         int64
	P20         int64
	P30         int64
	P40         int64
	P50         int64
	P60         int64
	P70         int64
	P80         int64
	P90         int64
	P95         int64
	P99         int64
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
				"p10_accepted_fee":     fmt.Sprint(action.P10),
				"p20_accepted_fee":     fmt.Sprint(action.P20),
				"p30_accepted_fee":     fmt.Sprint(action.P30),
				"p40_accepted_fee":     fmt.Sprint(action.P40),
				"p50_accepted_fee":     fmt.Sprint(action.P50),
				"p60_accepted_fee":     fmt.Sprint(action.P60),
				"p70_accepted_fee":     fmt.Sprint(action.P70),
				"p80_accepted_fee":     fmt.Sprint(action.P80),
				"p90_accepted_fee":     fmt.Sprint(action.P90),
				"p95_accepted_fee":     fmt.Sprint(action.P95),
				"p99_accepted_fee":     fmt.Sprint(action.P99),
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
	action.P10 = cur.P10
	action.P20 = cur.P20
	action.P30 = cur.P30
	action.P40 = cur.P40
	action.P50 = cur.P50
	action.P60 = cur.P60
	action.P70 = cur.P70
	action.P80 = cur.P80
	action.P90 = cur.P90
	action.P95 = cur.P95
	action.P99 = cur.P99
}
