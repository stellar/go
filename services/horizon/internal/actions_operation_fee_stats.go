package horizon

import (
	"net/http"
	"strconv"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/operationfeestats"
	"github.com/stellar/go/support/render/httpjson"
	"github.com/stellar/go/support/render/problem"
)

// This file contains the actions:
//
// FeeStatsAction: stats representing current state of network fees

var _ actions.JSONer = (*FeeStatsAction)(nil)

// this struct is very similar to hProtocol.feeStats but drops the usage of int
// in favor of int64
type feeStats struct {
	LastLedger          uint32  `json:"last_ledger,string"`
	LastLedgerBaseFee   int64   `json:"last_ledger_base_fee,string"`
	LedgerCapacityUsage float64 `json:"ledger_capacity_usage,string"`
	MinAcceptedFee      int64   `json:"min_accepted_fee,string"`
	ModeAcceptedFee     int64   `json:"mode_accepted_fee,string"`
	P10AcceptedFee      int64   `json:"p10_accepted_fee,string"`
	P20AcceptedFee      int64   `json:"p20_accepted_fee,string"`
	P30AcceptedFee      int64   `json:"p30_accepted_fee,string"`
	P40AcceptedFee      int64   `json:"p40_accepted_fee,string"`
	P50AcceptedFee      int64   `json:"p50_accepted_fee,string"`
	P60AcceptedFee      int64   `json:"p60_accepted_fee,string"`
	P70AcceptedFee      int64   `json:"p70_accepted_fee,string"`
	P80AcceptedFee      int64   `json:"p80_accepted_fee,string"`
	P90AcceptedFee      int64   `json:"p90_accepted_fee,string"`
	P95AcceptedFee      int64   `json:"p95_accepted_fee,string"`
	P99AcceptedFee      int64   `json:"p99_accepted_fee,string"`

	FeeCharged hProtocol.FeeDistribution `json:"fee_charged"`
	MaxFee     hProtocol.FeeDistribution `json:"max_fee"`
}

// FeeStatsAction renders a few useful statistics that describe the
// current state of operation fees on the network.
type FeeStatsAction struct {
	Action
	feeStats feeStats
}

// JSON is a method for actions.JSON
func (action *FeeStatsAction) JSON() error {
	if !action.App.config.IngestFailedTransactions {
		// If Horizon is not ingesting failed transaction it does not make sense to display
		// operation fee stats because they will be incorrect.
		p := problem.P{
			Type:   "endpoint_not_available",
			Title:  "Endpoint Not Available",
			Status: http.StatusNotImplemented,
			Detail: "/fee_stats is unavailable when Horizon is not ingesting failed " +
				"transactions. Set `INGEST_FAILED_TRANSACTIONS=true` to start ingesting them.",
		}
		problem.Render(action.R.Context(), action.W, p)
		return nil
	}

	action.Do(
		action.loadRecords,
		func() {
			httpjson.Render(
				action.W,
				action.feeStats,
				httpjson.HALJSON,
			)
		},
	)
	return action.Err
}

func (action *FeeStatsAction) loadRecords() {
	cur := operationfeestats.CurrentState()
	action.feeStats.LastLedgerBaseFee = cur.LastBaseFee
	action.feeStats.LastLedger = cur.LastLedger

	ledgerCapacityUsage, err := strconv.ParseFloat(cur.LedgerCapacityUsage, 64)
	if err != nil {
		action.Err = err
		return
	}

	action.feeStats.LedgerCapacityUsage = ledgerCapacityUsage

	// FeeCharged
	action.feeStats.FeeCharged.Max = cur.FeeChargedMax
	action.feeStats.FeeCharged.Min = cur.FeeChargedMin
	action.feeStats.FeeCharged.Mode = cur.FeeChargedMode
	action.feeStats.FeeCharged.P10 = cur.FeeChargedP10
	action.feeStats.FeeCharged.P20 = cur.FeeChargedP20
	action.feeStats.FeeCharged.P30 = cur.FeeChargedP30
	action.feeStats.FeeCharged.P40 = cur.FeeChargedP40
	action.feeStats.FeeCharged.P50 = cur.FeeChargedP50
	action.feeStats.FeeCharged.P60 = cur.FeeChargedP60
	action.feeStats.FeeCharged.P70 = cur.FeeChargedP70
	action.feeStats.FeeCharged.P80 = cur.FeeChargedP80
	action.feeStats.FeeCharged.P90 = cur.FeeChargedP90
	action.feeStats.FeeCharged.P95 = cur.FeeChargedP95
	action.feeStats.FeeCharged.P99 = cur.FeeChargedP99

	// MaxFee
	action.feeStats.MaxFee.Max = cur.MaxFeeMax
	action.feeStats.MaxFee.Min = cur.MaxFeeMin
	action.feeStats.MaxFee.Mode = cur.MaxFeeMode
	action.feeStats.MaxFee.P10 = cur.MaxFeeP10
	action.feeStats.MaxFee.P20 = cur.MaxFeeP20
	action.feeStats.MaxFee.P30 = cur.MaxFeeP30
	action.feeStats.MaxFee.P40 = cur.MaxFeeP40
	action.feeStats.MaxFee.P50 = cur.MaxFeeP50
	action.feeStats.MaxFee.P60 = cur.MaxFeeP60
	action.feeStats.MaxFee.P70 = cur.MaxFeeP70
	action.feeStats.MaxFee.P80 = cur.MaxFeeP80
	action.feeStats.MaxFee.P90 = cur.MaxFeeP90
	action.feeStats.MaxFee.P95 = cur.MaxFeeP95
	action.feeStats.MaxFee.P99 = cur.MaxFeeP99

	// AcceptedFee is an alias for MaxFee
	// Action needed in release: horizon-v0.25.0
	// Remove AcceptedFee fields
	action.feeStats.MinAcceptedFee = action.feeStats.MaxFee.Min
	action.feeStats.ModeAcceptedFee = action.feeStats.MaxFee.Mode
	action.feeStats.P10AcceptedFee = action.feeStats.MaxFee.P10
	action.feeStats.P20AcceptedFee = action.feeStats.MaxFee.P20
	action.feeStats.P30AcceptedFee = action.feeStats.MaxFee.P30
	action.feeStats.P40AcceptedFee = action.feeStats.MaxFee.P40
	action.feeStats.P50AcceptedFee = action.feeStats.MaxFee.P50
	action.feeStats.P60AcceptedFee = action.feeStats.MaxFee.P60
	action.feeStats.P70AcceptedFee = action.feeStats.MaxFee.P70
	action.feeStats.P80AcceptedFee = action.feeStats.MaxFee.P80
	action.feeStats.P90AcceptedFee = action.feeStats.MaxFee.P90
	action.feeStats.P95AcceptedFee = action.feeStats.MaxFee.P95
	action.feeStats.P99AcceptedFee = action.feeStats.MaxFee.P99
}
