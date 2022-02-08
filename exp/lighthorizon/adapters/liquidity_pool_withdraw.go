package adapters

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/xdr"
)

func populateLiquidityPoolWithdrawOperation(op *common.Operation, baseOp operations.Base) (operations.LiquidityPoolWithdraw, error) {
	liquidityPoolWithdraw := op.Get().Body.MustLiquidityPoolWithdrawOp()

	return operations.LiquidityPoolWithdraw{
		Base: baseOp,
		// TODO: some fields missing because derived from meta
		LiquidityPoolID: xdr.Hash(liquidityPoolWithdraw.LiquidityPoolId).HexString(),
		ReservesMin: []base.AssetAmount{
			{Amount: amount.String(liquidityPoolWithdraw.MinAmountA)},
			{Amount: amount.String(liquidityPoolWithdraw.MinAmountB)},
		},
		Shares: amount.String(liquidityPoolWithdraw.Amount),
	}, nil
}
