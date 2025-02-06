package ingest

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

type LiquidityPoolWithdrawDetail struct {
	LiquidityPoolID        string `json:"liquidity_pool_id"`
	ReserveAAssetCode      string `json:"reserve_a_asset_code"`
	ReserveAAssetIssuer    string `json:"reserve_a_asset_issuer"`
	ReserveAAssetType      string `json:"reserve_a_asset_type"`
	ReserveAMinAmount      int64  `json:"reserve_a_min_amount,string"`
	ReserveAWithdrawAmount int64  `json:"reserve_a_withdraw_amount,string"`
	ReserveBAssetCode      string `json:"reserve_b_asset_code"`
	ReserveBAssetIssuer    string `json:"reserve_b_asset_issuer"`
	ReserveBAssetType      string `json:"reserve_b_asset_type"`
	ReserveBMinAmount      int64  `json:"reserve_b_min_amount,string"`
	ReserveBWithdrawAmount int64  `json:"reserve_b_withdraw_amount,string"`
	Shares                 int64  `json:"shares,string"`
}

func (o *LedgerOperation) LiquidityPoolWithdrawDetails() (LiquidityPoolWithdrawDetail, error) {
	op, ok := o.Operation.Body.GetLiquidityPoolWithdrawOp()
	if !ok {
		return LiquidityPoolWithdrawDetail{}, fmt.Errorf("could not access LiquidityPoolWithdraw info for this operation (index %d)", o.OperationIndex)
	}

	liquidityPoolWithdrawDetail := LiquidityPoolWithdrawDetail{
		ReserveAMinAmount: int64(op.MinAmountA),
		ReserveBMinAmount: int64(op.MinAmountB),
		Shares:            int64(op.Amount),
	}

	var err error
	var liquidityPoolID string
	liquidityPoolID, err = PoolIDToString(op.LiquidityPoolId)
	if err != nil {
		return LiquidityPoolWithdrawDetail{}, err
	}

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

	liquidityPoolWithdrawDetail.ReserveAAssetCode = assetACode
	liquidityPoolWithdrawDetail.ReserveAAssetIssuer = assetAIssuer
	liquidityPoolWithdrawDetail.ReserveAAssetType = assetAType
	liquidityPoolWithdrawDetail.ReserveAWithdrawAmount = int64(receivedA)

	// Process AssetB Details
	var assetBCode, assetBIssuer, assetBType string
	err = assetB.Extract(&assetBType, &assetBCode, &assetBIssuer)
	if err != nil {
		return LiquidityPoolWithdrawDetail{}, err
	}

	liquidityPoolWithdrawDetail.ReserveBAssetCode = assetBCode
	liquidityPoolWithdrawDetail.ReserveBAssetIssuer = assetBIssuer
	liquidityPoolWithdrawDetail.ReserveBAssetType = assetBType
	liquidityPoolWithdrawDetail.ReserveBWithdrawAmount = int64(receivedB)

	return liquidityPoolWithdrawDetail, nil
}
