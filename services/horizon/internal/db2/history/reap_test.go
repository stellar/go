package history_test

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
)

func TestReapLookupTables(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	tt.Scenario("kahuna")
	//
	//db := tt.HorizonSession()
	//reaper := ingest.NewReaper(
	//	ingest.ReapConfig{
	//		RetentionCount: 1,
	//		BatchSize:      50,
	//	},
	//	db,
	//)
	//
	//var (
	//	prevLedgers, curLedgers                     int
	//	prevAccounts, curAccounts                   int
	//	prevAssets, curAssets                       int
	//	prevClaimableBalances, curClaimableBalances int
	//	prevLiquidityPools, curLiquidityPools       int
	//)
	//
	//err := db.GetRaw(tt.Ctx, &prevLedgers, `SELECT COUNT(*) FROM history_ledgers`)
	//tt.Require.NoError(err)
	//err = db.GetRaw(tt.Ctx, &prevAccounts, `SELECT COUNT(*) FROM history_accounts`)
	//tt.Require.NoError(err)
	//err = db.GetRaw(tt.Ctx, &prevAssets, `SELECT COUNT(*) FROM history_assets`)
	//tt.Require.NoError(err)
	//err = db.GetRaw(tt.Ctx, &prevClaimableBalances, `SELECT COUNT(*) FROM history_claimable_balances`)
	//tt.Require.NoError(err)
	//err = db.GetRaw(tt.Ctx, &prevLiquidityPools, `SELECT COUNT(*) FROM history_liquidity_pools`)
	//tt.Require.NoError(err)
	//
	//err = reaper.DeleteUnretainedHistory(tt.Ctx)
	//tt.Require.NoError(err)
	//
	//q := &history.Q{tt.HorizonSession()}
	//
	//err = q.Begin(tt.Ctx)
	//tt.Require.NoError(err)
	//
	//results, err := q.ReapLookupTables(tt.Ctx, 5)
	//tt.Require.NoError(err)
	//
	//err = q.Commit()
	//tt.Require.NoError(err)
	//
	//err = db.GetRaw(tt.Ctx, &curLedgers, `SELECT COUNT(*) FROM history_ledgers`)
	//tt.Require.NoError(err)
	//err = db.GetRaw(tt.Ctx, &curAccounts, `SELECT COUNT(*) FROM history_accounts`)
	//tt.Require.NoError(err)
	//err = db.GetRaw(tt.Ctx, &curAssets, `SELECT COUNT(*) FROM history_assets`)
	//tt.Require.NoError(err)
	//err = db.GetRaw(tt.Ctx, &curClaimableBalances, `SELECT COUNT(*) FROM history_claimable_balances`)
	//tt.Require.NoError(err)
	//err = db.GetRaw(tt.Ctx, &curLiquidityPools, `SELECT COUNT(*) FROM history_liquidity_pools`)
	//tt.Require.NoError(err)
	//
	//tt.Assert.Equal(61, prevLedgers, "prevLedgers")
	//tt.Assert.Equal(1, curLedgers, "curLedgers")
	//
	//tt.Assert.Equal(25, prevAccounts, "prevAccounts")
	//tt.Assert.Equal(21, curAccounts, "curAccounts")
	//tt.Assert.Equal(int64(4), results["history_accounts"].RowsDeleted, `deletedCount["history_accounts"]`)
	//
	//tt.Assert.Equal(7, prevAssets, "prevAssets")
	//tt.Assert.Equal(2, curAssets, "curAssets")
	//tt.Assert.Equal(int64(5), results["history_assets"].RowsDeleted, `deletedCount["history_assets"]`)
	//
	//tt.Assert.Equal(1, prevClaimableBalances, "prevClaimableBalances")
	//tt.Assert.Equal(0, curClaimableBalances, "curClaimableBalances")
	//tt.Assert.Equal(int64(1), results["history_claimable_balances"].RowsDeleted, `deletedCount["history_claimable_balances"]`)
	//
	//tt.Assert.Equal(1, prevLiquidityPools, "prevLiquidityPools")
	//tt.Assert.Equal(0, curLiquidityPools, "curLiquidityPools")
	//tt.Assert.Equal(int64(1), results["history_liquidity_pools"].RowsDeleted, `deletedCount["history_liquidity_pools"]`)
	//
	//tt.Assert.Len(results, 4)
	//tt.Assert.Equal(int64(6), results["history_accounts"].Offset)
	//tt.Assert.Equal(int64(6), results["history_assets"].Offset)
	//tt.Assert.Equal(int64(0), results["history_claimable_balances"].Offset)
	//tt.Assert.Equal(int64(0), results["history_liquidity_pools"].Offset)
	//
	//err = q.Begin(tt.Ctx)
	//tt.Require.NoError(err)
	//
	//results, err = q.ReapLookupTables(tt.Ctx, 5)
	//tt.Require.NoError(err)
	//
	//err = q.Commit()
	//tt.Require.NoError(err)
	//
	//err = db.GetRaw(tt.Ctx, &curAccounts, `SELECT COUNT(*) FROM history_accounts`)
	//tt.Require.NoError(err)
	//err = db.GetRaw(tt.Ctx, &curAssets, `SELECT COUNT(*) FROM history_assets`)
	//tt.Require.NoError(err)
	//
	//tt.Assert.Equal(16, curAccounts, "curAccounts")
	//tt.Assert.Equal(int64(5), results["history_accounts"].RowsDeleted, `deletedCount["history_accounts"]`)
	//
	//tt.Assert.Equal(0, curAssets, "curAssets")
	//tt.Assert.Equal(int64(2), results["history_assets"].RowsDeleted, `deletedCount["history_assets"]`)
	//
	//tt.Assert.Equal(int64(0), results["history_claimable_balances"].RowsDeleted, `deletedCount["history_claimable_balances"]`)
	//
	//tt.Assert.Equal(int64(0), results["history_liquidity_pools"].RowsDeleted, `deletedCount["history_liquidity_pools"]`)
	//
	//tt.Assert.Len(results, 4)
	//tt.Assert.Equal(int64(11), results["history_accounts"].Offset)
	//tt.Assert.Equal(int64(0), results["history_assets"].Offset)
	//tt.Assert.Equal(int64(0), results["history_claimable_balances"].Offset)
	//tt.Assert.Equal(int64(0), results["history_liquidity_pools"].Offset)
	//
	//err = q.Begin(tt.Ctx)
	//tt.Require.NoError(err)
	//
	//results, err = q.ReapLookupTables(tt.Ctx, 1000)
	//tt.Require.NoError(err)
	//
	//err = q.Commit()
	//tt.Require.NoError(err)
	//
	//err = db.GetRaw(tt.Ctx, &curAccounts, `SELECT COUNT(*) FROM history_accounts`)
	//tt.Require.NoError(err)
	//err = db.GetRaw(tt.Ctx, &curAssets, `SELECT COUNT(*) FROM history_assets`)
	//tt.Require.NoError(err)
	//
	//tt.Assert.Equal(1, curAccounts, "curAccounts")
	//tt.Assert.Equal(int64(15), results["history_accounts"].RowsDeleted, `deletedCount["history_accounts"]`)
	//
	//tt.Assert.Equal(0, curAssets, "curAssets")
	//tt.Assert.Equal(int64(0), results["history_assets"].RowsDeleted, `deletedCount["history_assets"]`)
	//
	//tt.Assert.Equal(int64(0), results["history_claimable_balances"].RowsDeleted, `deletedCount["history_claimable_balances"]`)
	//
	//tt.Assert.Equal(int64(0), results["history_liquidity_pools"].RowsDeleted, `deletedCount["history_liquidity_pools"]`)
	//
	//tt.Assert.Len(results, 4)
	//tt.Assert.Equal(int64(0), results["history_accounts"].Offset)
	//tt.Assert.Equal(int64(0), results["history_assets"].Offset)
	//tt.Assert.Equal(int64(0), results["history_claimable_balances"].Offset)
	//tt.Assert.Equal(int64(0), results["history_liquidity_pools"].Offset)
}
