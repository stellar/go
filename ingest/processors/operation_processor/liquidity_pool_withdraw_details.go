package operation

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

type LiquidityPoolWithdrawDetail struct {
	LiquidityPoolID string       `json:"liquidity_pool_id"`
	ReserveAssetA   ReserveAsset `json:"reserve_asset_a"`
	ReserveAssetB   ReserveAsset `json:"reserve_asset_b"`
	Shares          int64        `json:"shares,string"`
}

func (o *LedgerOperation) LiquidityPoolWithdrawDetails() (LiquidityPoolWithdrawDetail, error) {
	op, ok := o.Operation.Body.GetLiquidityPoolWithdrawOp()
	if !ok {
		return LiquidityPoolWithdrawDetail{}, fmt.Errorf("could not access LiquidityPoolWithdraw info for this operation (index %d)", o.OperationIndex)
	}

	liquidityPoolWithdrawDetail := LiquidityPoolWithdrawDetail{
		ReserveAssetA: ReserveAsset{
			MinAmount: int64(op.MinAmountA),
		},
		ReserveAssetB: ReserveAsset{
			MinAmount: int64(op.MinAmountB),
		},
		Shares: int64(op.Amount),
	}

	var err error
	liquidityPoolID := PoolIDToString(op.LiquidityPoolId)

	liquidityPoolWithdrawDetail.LiquidityPoolID = liquidityPoolID

	var (
		assetA, assetB       xdr.Asset
		receivedA, receivedB xdr.Int64
		lp                   *xdr.LiquidityPoolEntry
		delta                *LiquidityPoolDelta
	)
	if o.Transaction.Successful() {
		// we will use the defaults (omitted asset and 0 amounts) if the transaction failed
		lp, delta, err = o.getLiquidityPoolAndProductDelta(&op.LiquidityPoolId)
		if err != nil {
			return LiquidityPoolWithdrawDetail{}, err
		}
		params := lp.Body.ConstantProduct.Params
		assetA, assetB = params.AssetA, params.AssetB
		receivedA, receivedB = -delta.ReserveA, -delta.ReserveB
	}

	// Process AssetA Details
	var assetACode, assetAIssuer, assetAType string
	err = assetA.Extract(&assetAType, &assetACode, &assetAIssuer)
	if err != nil {
		return LiquidityPoolWithdrawDetail{}, err
	}

	liquidityPoolWithdrawDetail.ReserveAssetA.AssetCode = assetACode
	liquidityPoolWithdrawDetail.ReserveAssetA.AssetIssuer = assetAIssuer
	liquidityPoolWithdrawDetail.ReserveAssetA.AssetType = assetAType
	liquidityPoolWithdrawDetail.ReserveAssetA.WithdrawAmount = int64(receivedA)

	// Process AssetB Details
	var assetBCode, assetBIssuer, assetBType string
	err = assetB.Extract(&assetBType, &assetBCode, &assetBIssuer)
	if err != nil {
		return LiquidityPoolWithdrawDetail{}, err
	}

	liquidityPoolWithdrawDetail.ReserveAssetB.AssetCode = assetBCode
	liquidityPoolWithdrawDetail.ReserveAssetB.AssetIssuer = assetBIssuer
	liquidityPoolWithdrawDetail.ReserveAssetB.AssetType = assetBType
	liquidityPoolWithdrawDetail.ReserveAssetB.WithdrawAmount = int64(receivedB)

	return liquidityPoolWithdrawDetail, nil
}
