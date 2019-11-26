package horizon

import (
	"encoding/json"
	hProtocol "github.com/stellar/go/protocols/horizon"
	"testing"
)

func TestOperationFeeTestsActions_Show(t *testing.T) {
	testCases := []struct {
		scenario            string
		lastbasefee         int
		max                 int
		min                 int
		mode                int
		p10                 int
		p20                 int
		p30                 int
		p40                 int
		p50                 int
		p60                 int
		p70                 int
		p80                 int
		p90                 int
		p95                 int
		p99                 int
		feeChargedMax       int64
		feeChargedMin       int64
		feeChargedMode      int64
		feeChargedP10       int64
		feeChargedP20       int64
		feeChargedP30       int64
		feeChargedP40       int64
		feeChargedP50       int64
		feeChargedP60       int64
		feeChargedP70       int64
		feeChargedP80       int64
		feeChargedP90       int64
		feeChargedP95       int64
		feeChargedP99       int64
		ledgerCapacityUsage float64
	}{
		// happy path
		{
			scenario:            "operation_fee_stats_1",
			lastbasefee:         100,
			max:                 100,
			min:                 100,
			mode:                100,
			p10:                 100,
			p20:                 100,
			p30:                 100,
			p40:                 100,
			p50:                 100,
			p60:                 100,
			p70:                 100,
			p80:                 100,
			p90:                 100,
			p95:                 100,
			p99:                 100,
			feeChargedMax:       100,
			feeChargedMin:       100,
			feeChargedMode:      100,
			feeChargedP10:       100,
			feeChargedP20:       100,
			feeChargedP30:       100,
			feeChargedP40:       100,
			feeChargedP50:       100,
			feeChargedP60:       100,
			feeChargedP70:       100,
			feeChargedP80:       100,
			feeChargedP90:       100,
			feeChargedP95:       100,
			feeChargedP99:       100,
			ledgerCapacityUsage: 0.04,
		},
		// no transactions in last 5 ledgers
		{
			scenario:            "operation_fee_stats_2",
			ledgerCapacityUsage: 0.00,
			lastbasefee:         100,
			max:                 100,
			min:                 100,
			mode:                100,
			p10:                 100,
			p20:                 100,
			p30:                 100,
			p40:                 100,
			p50:                 100,
			p60:                 100,
			p70:                 100,
			p80:                 100,
			p90:                 100,
			p95:                 100,
			p99:                 100,
			feeChargedMax:       100,
			feeChargedMin:       100,
			feeChargedMode:      100,
			feeChargedP10:       100,
			feeChargedP20:       100,
			feeChargedP30:       100,
			feeChargedP40:       100,
			feeChargedP50:       100,
			feeChargedP60:       100,
			feeChargedP70:       100,
			feeChargedP80:       100,
			feeChargedP90:       100,
			feeChargedP95:       100,
			feeChargedP99:       100,
		},
		// transactions with varying fees
		{
			scenario:            "operation_fee_stats_3",
			ledgerCapacityUsage: 0.03,
			lastbasefee:         100,
			max:                 400,
			min:                 200,
			mode:                400,
			p10:                 200,
			p20:                 300,
			p30:                 400,
			p40:                 400,
			p50:                 400,
			p60:                 400,
			p70:                 400,
			p80:                 400,
			p90:                 400,
			p95:                 400,
			p99:                 400,
			feeChargedMax:       100,
			feeChargedMin:       100,
			feeChargedMode:      100,
			feeChargedP10:       100,
			feeChargedP20:       100,
			feeChargedP30:       100,
			feeChargedP40:       100,
			feeChargedP50:       100,
			feeChargedP60:       100,
			feeChargedP70:       100,
			feeChargedP80:       100,
			feeChargedP90:       100,
			feeChargedP95:       100,
			feeChargedP99:       100,
		},
	}

	for _, kase := range testCases {
		t.Run("/fee_stats", func(t *testing.T) {
			ht := StartHTTPTest(t, kase.scenario)
			defer ht.Finish()

			// Update max_tx_set_size on ledgers
			_, err := ht.HorizonSession().ExecRaw("UPDATE history_ledgers SET max_tx_set_size = 50")
			ht.Require.NoError(err)

			ht.App.UpdateFeeStatsState()

			w := ht.Get("/fee_stats")

			if ht.Assert.Equal(200, w.Code) {
				var result hProtocol.FeeStats
				err := json.Unmarshal(w.Body.Bytes(), &result)
				ht.Require.NoError(err)
				ht.Assert.Equal(kase.lastbasefee, result.LastLedgerBaseFee, "base_fee")
				ht.Assert.Equal(kase.ledgerCapacityUsage, result.LedgerCapacityUsage, "ledger_capacity_usage")

				ht.Assert.Equal(kase.min, result.MinAcceptedFee, "min")
				ht.Assert.Equal(kase.mode, result.ModeAcceptedFee, "mode")
				ht.Assert.Equal(kase.p10, result.P10AcceptedFee, "p10")
				ht.Assert.Equal(kase.p20, result.P20AcceptedFee, "p20")
				ht.Assert.Equal(kase.p30, result.P30AcceptedFee, "p30")
				ht.Assert.Equal(kase.p40, result.P40AcceptedFee, "p40")
				ht.Assert.Equal(kase.p50, result.P50AcceptedFee, "p50")
				ht.Assert.Equal(kase.p60, result.P60AcceptedFee, "p60")
				ht.Assert.Equal(kase.p70, result.P70AcceptedFee, "p70")
				ht.Assert.Equal(kase.p80, result.P80AcceptedFee, "p80")
				ht.Assert.Equal(kase.p90, result.P90AcceptedFee, "p90")
				ht.Assert.Equal(kase.p95, result.P95AcceptedFee, "p95")
				ht.Assert.Equal(kase.p99, result.P99AcceptedFee, "p99")

				// AcceptedFee is an alias for MaxFee data
				ht.Assert.Equal(int64(kase.min), result.MaxFee.Min, "min")
				ht.Assert.Equal(int64(kase.mode), result.MaxFee.Mode, "mode")
				ht.Assert.Equal(int64(kase.p10), result.MaxFee.P10, "p10")
				ht.Assert.Equal(int64(kase.p20), result.MaxFee.P20, "p20")
				ht.Assert.Equal(int64(kase.p30), result.MaxFee.P30, "p30")
				ht.Assert.Equal(int64(kase.p40), result.MaxFee.P40, "p40")
				ht.Assert.Equal(int64(kase.p50), result.MaxFee.P50, "p50")
				ht.Assert.Equal(int64(kase.p60), result.MaxFee.P60, "p60")
				ht.Assert.Equal(int64(kase.p70), result.MaxFee.P70, "p70")
				ht.Assert.Equal(int64(kase.p80), result.MaxFee.P80, "p80")
				ht.Assert.Equal(int64(kase.p90), result.MaxFee.P90, "p90")
				ht.Assert.Equal(int64(kase.p95), result.MaxFee.P95, "p95")
				ht.Assert.Equal(int64(kase.p99), result.MaxFee.P99, "p99")

				ht.Assert.Equal(kase.feeChargedMax, result.FeeCharged.Max, "fee_charged_max")
				ht.Assert.Equal(kase.feeChargedMin, result.FeeCharged.Min, "fee_charged_min")
				ht.Assert.Equal(kase.feeChargedMode, result.FeeCharged.Mode, "fee_charged_mode")
				ht.Assert.Equal(kase.feeChargedP10, result.FeeCharged.P10, "fee_charged_p10")
				ht.Assert.Equal(kase.feeChargedP20, result.FeeCharged.P20, "fee_charged_p20")
				ht.Assert.Equal(kase.feeChargedP30, result.FeeCharged.P30, "fee_charged_p30")
				ht.Assert.Equal(kase.feeChargedP40, result.FeeCharged.P40, "fee_charged_p40")
				ht.Assert.Equal(kase.feeChargedP50, result.FeeCharged.P50, "fee_charged_p50")
				ht.Assert.Equal(kase.feeChargedP60, result.FeeCharged.P60, "fee_charged_p60")
				ht.Assert.Equal(kase.feeChargedP70, result.FeeCharged.P70, "fee_charged_p70")
				ht.Assert.Equal(kase.feeChargedP80, result.FeeCharged.P80, "fee_charged_p80")
				ht.Assert.Equal(kase.feeChargedP90, result.FeeCharged.P90, "fee_charged_p90")
				ht.Assert.Equal(kase.feeChargedP95, result.FeeCharged.P95, "fee_charged_p95")
				ht.Assert.Equal(kase.feeChargedP99, result.FeeCharged.P99, "fee_charged_p99")
			}
		})
	}
}

// TestOperationFeeTestsActions_ShowMultiOp tests fee stats in case transactions contain multiple operations.
// In such case, since protocol v11, we should use number of operations as the indicator of ledger capacity usage.
func TestOperationFeeTestsActions_ShowMultiOp(t *testing.T) {
	ht := StartHTTPTest(t, "operation_fee_stats_3")
	defer ht.Finish()

	// Update max_tx_set_size on ledgers
	_, err := ht.HorizonSession().ExecRaw("UPDATE history_ledgers SET max_tx_set_size = 50")
	ht.Require.NoError(err)

	// Update number of ops on each transaction
	_, err = ht.HorizonSession().ExecRaw("UPDATE history_transactions SET operation_count = operation_count * 2")
	ht.Require.NoError(err)

	ht.App.UpdateFeeStatsState()

	w := ht.Get("/fee_stats")

	if ht.Assert.Equal(200, w.Code) {
		var result hProtocol.FeeStats
		err := json.Unmarshal(w.Body.Bytes(), &result)
		ht.Require.NoError(err)
		ht.Assert.Equal(100, result.LastLedgerBaseFee, "base_fee")
		ht.Assert.Equal(0.06, result.LedgerCapacityUsage, "ledger_capacity_usage")

		ht.Assert.Equal(100, result.MinAcceptedFee, "min")
		ht.Assert.Equal(200, result.ModeAcceptedFee, "mode")
		ht.Assert.Equal(100, result.P10AcceptedFee, "p10")
		ht.Assert.Equal(150, result.P20AcceptedFee, "p20")
		ht.Assert.Equal(200, result.P30AcceptedFee, "p30")
		ht.Assert.Equal(200, result.P40AcceptedFee, "p40")
		ht.Assert.Equal(200, result.P50AcceptedFee, "p50")
		ht.Assert.Equal(200, result.P60AcceptedFee, "p60")
		ht.Assert.Equal(200, result.P70AcceptedFee, "p70")
		ht.Assert.Equal(200, result.P80AcceptedFee, "p80")
		ht.Assert.Equal(200, result.P90AcceptedFee, "p90")
		ht.Assert.Equal(200, result.P95AcceptedFee, "p95")
		ht.Assert.Equal(200, result.P99AcceptedFee, "p99")

		// AcceptedFee is an alias for MaxFee data
		ht.Assert.Equal(int64(200), result.MaxFee.Max, "max_fee_max")
		ht.Assert.Equal(int64(100), result.MaxFee.Min, "max_fee_min")
		ht.Assert.Equal(int64(200), result.MaxFee.Mode, "max_fee_mode")
		ht.Assert.Equal(int64(100), result.MaxFee.P10, "max_fee_p10")
		ht.Assert.Equal(int64(150), result.MaxFee.P20, "max_fee_p20")
		ht.Assert.Equal(int64(200), result.MaxFee.P30, "max_fee_p30")
		ht.Assert.Equal(int64(200), result.MaxFee.P40, "max_fee_p40")
		ht.Assert.Equal(int64(200), result.MaxFee.P50, "max_fee_p50")
		ht.Assert.Equal(int64(200), result.MaxFee.P60, "max_fee_p60")
		ht.Assert.Equal(int64(200), result.MaxFee.P70, "max_fee_p70")
		ht.Assert.Equal(int64(200), result.MaxFee.P80, "max_fee_p80")
		ht.Assert.Equal(int64(200), result.MaxFee.P90, "max_fee_p90")
		ht.Assert.Equal(int64(200), result.MaxFee.P95, "max_fee_p95")
		ht.Assert.Equal(int64(200), result.MaxFee.P99, "max_fee_p99")

		ht.Assert.Equal(int64(50), result.FeeCharged.Max, "fee_charged_max")
		ht.Assert.Equal(int64(50), result.FeeCharged.Min, "fee_charged_min")
		ht.Assert.Equal(int64(50), result.FeeCharged.Mode, "fee_charged_mode")
		ht.Assert.Equal(int64(50), result.FeeCharged.P10, "fee_charged_p10")
		ht.Assert.Equal(int64(50), result.FeeCharged.P20, "fee_charged_p20")
		ht.Assert.Equal(int64(50), result.FeeCharged.P30, "fee_charged_p30")
		ht.Assert.Equal(int64(50), result.FeeCharged.P40, "fee_charged_p40")
		ht.Assert.Equal(int64(50), result.FeeCharged.P50, "fee_charged_p50")
		ht.Assert.Equal(int64(50), result.FeeCharged.P60, "fee_charged_p60")
		ht.Assert.Equal(int64(50), result.FeeCharged.P70, "fee_charged_p70")
		ht.Assert.Equal(int64(50), result.FeeCharged.P80, "fee_charged_p80")
		ht.Assert.Equal(int64(50), result.FeeCharged.P90, "fee_charged_p90")
		ht.Assert.Equal(int64(50), result.FeeCharged.P95, "fee_charged_p95")
		ht.Assert.Equal(int64(50), result.FeeCharged.P99, "fee_charged_p99")
	}
}

func TestOperationFeeTestsActions_NotInterpolating(t *testing.T) {
	ht := StartHTTPTest(t, "operation_fee_stats_3")
	defer ht.Finish()

	// Update max_tx_set_size on ledgers
	_, err := ht.HorizonSession().ExecRaw("UPDATE history_ledgers SET max_tx_set_size = 50")
	ht.Require.NoError(err)

	// Update one tx to a huge fee
	_, err = ht.HorizonSession().ExecRaw("UPDATE history_transactions SET max_fee = 256000, operation_count = 16 WHERE transaction_hash = '6a349e7331e93a251367287e274fb1699abaf723bde37aebe96248c76fd3071a'")
	ht.Require.NoError(err)

	ht.App.UpdateFeeStatsState()

	w := ht.Get("/fee_stats")

	if ht.Assert.Equal(200, w.Code) {
		var result hProtocol.FeeStats
		err := json.Unmarshal(w.Body.Bytes(), &result)
		ht.Require.NoError(err)
		ht.Assert.Equal(100, result.LastLedgerBaseFee, "base_fee")
		ht.Assert.Equal(0.09, result.LedgerCapacityUsage, "ledger_capacity_usage")
		ht.Assert.Equal(200, result.MinAcceptedFee, "min")
		ht.Assert.Equal(400, result.ModeAcceptedFee, "mode")
		ht.Assert.Equal(200, result.P10AcceptedFee, "p10")
		ht.Assert.Equal(300, result.P20AcceptedFee, "p20")
		ht.Assert.Equal(400, result.P30AcceptedFee, "p30")
		ht.Assert.Equal(400, result.P40AcceptedFee, "p40")
		ht.Assert.Equal(400, result.P50AcceptedFee, "p50")
		ht.Assert.Equal(400, result.P60AcceptedFee, "p60")
		ht.Assert.Equal(400, result.P70AcceptedFee, "p70")
		ht.Assert.Equal(400, result.P80AcceptedFee, "p80")
		ht.Assert.Equal(16000, result.P90AcceptedFee, "p90")
		ht.Assert.Equal(16000, result.P95AcceptedFee, "p95")
		ht.Assert.Equal(16000, result.P99AcceptedFee, "p99")

		// AcceptedFee is an alias for MaxFee data
		ht.Assert.Equal(int64(16000), result.MaxFee.Max, "max_fee_max")
		ht.Assert.Equal(int64(200), result.MaxFee.Min, "max_fee_min")
		ht.Assert.Equal(int64(400), result.MaxFee.Mode, "max_fee_mode")
		ht.Assert.Equal(int64(200), result.MaxFee.P10, "max_fee_p10")
		ht.Assert.Equal(int64(300), result.MaxFee.P20, "max_fee_p20")
		ht.Assert.Equal(int64(400), result.MaxFee.P30, "max_fee_p30")
		ht.Assert.Equal(int64(400), result.MaxFee.P40, "max_fee_p40")
		ht.Assert.Equal(int64(400), result.MaxFee.P50, "max_fee_p50")
		ht.Assert.Equal(int64(400), result.MaxFee.P60, "max_fee_p60")
		ht.Assert.Equal(int64(400), result.MaxFee.P70, "max_fee_p70")
		ht.Assert.Equal(int64(400), result.MaxFee.P80, "max_fee_p80")
		ht.Assert.Equal(int64(16000), result.MaxFee.P90, "max_fee_p90")
		ht.Assert.Equal(int64(16000), result.MaxFee.P95, "max_fee_p95")
		ht.Assert.Equal(int64(16000), result.MaxFee.P99, "max_fee_p99")

		ht.Assert.Equal(int64(100), result.FeeCharged.Max, "fee_charged_max")
		ht.Assert.Equal(int64(6), result.FeeCharged.Min, "fee_charged_min")
		ht.Assert.Equal(int64(100), result.FeeCharged.Mode, "fee_charged_mode")
		ht.Assert.Equal(int64(6), result.FeeCharged.P10, "fee_charged_p10")
		ht.Assert.Equal(int64(100), result.FeeCharged.P20, "fee_charged_p20")
		ht.Assert.Equal(int64(100), result.FeeCharged.P30, "fee_charged_p30")
		ht.Assert.Equal(int64(100), result.FeeCharged.P40, "fee_charged_p40")
		ht.Assert.Equal(int64(100), result.FeeCharged.P50, "fee_charged_p50")
		ht.Assert.Equal(int64(100), result.FeeCharged.P60, "fee_charged_p60")
		ht.Assert.Equal(int64(100), result.FeeCharged.P70, "fee_charged_p70")
		ht.Assert.Equal(int64(100), result.FeeCharged.P80, "fee_charged_p80")
		ht.Assert.Equal(int64(100), result.FeeCharged.P90, "fee_charged_p90")
		ht.Assert.Equal(int64(100), result.FeeCharged.P95, "fee_charged_p95")
		ht.Assert.Equal(int64(100), result.FeeCharged.P99, "fee_charged_p99")
	}
}
