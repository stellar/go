package operation

import (
	"fmt"
	"strconv"

	"github.com/stellar/go/xdr"
)

type LiquidityPoolDepositDetail struct {
	LiquidityPoolID string       `json:"liquidity_pool_id"`
	ReserveAssetA   ReserveAsset `json:"reserve_asset_a"`
	ReserveAssetB   ReserveAsset `json:"reserve_asset_b"`
	MinPriceN       int32        `json:"min_price_n"`
	MinPriceD       int32        `json:"min_price_d"`
	MinPrice        float64      `json:"min_price"`
	MaxPriceN       int32        `json:"max_price_n"`
	MaxPriceD       int32        `json:"max_price_d"`
	MaxPrice        float64      `json:"max_price"`
	SharesReceived  int64        `json:"shares_received,string"`
}

type ReserveAsset struct {
	AssetCode      string `json:"asset_code"`
	AssetIssuer    string `json:"asset_issuer"`
	AssetType      string `json:"asset_type"`
	MinAmount      int64  `json:"min_amount,string"`
	MaxAmount      int64  `json:"max_amount,string"`
	DepositAmount  int64  `json:"deposit_amount,string"`
	WithdrawAmount int64  `json:"withdraw_amount,string"`
}

func (o *LedgerOperation) LiquidityPoolDepositDetails() (LiquidityPoolDepositDetail, error) {
	op, ok := o.Operation.Body.GetLiquidityPoolDepositOp()
	if !ok {
		return LiquidityPoolDepositDetail{}, fmt.Errorf("could not access LiquidityPoolDeposit info for this operation (index %d)", o.OperationIndex)
	}

	liquidityPoolDepositDetail := LiquidityPoolDepositDetail{
		ReserveAssetA: ReserveAsset{
			MaxAmount: int64(op.MaxAmountA),
		},
		ReserveAssetB: ReserveAsset{
			MaxAmount: int64(op.MaxAmountB),
		},
		MinPriceN: int32(op.MinPrice.N),
		MinPriceD: int32(op.MinPrice.D),
		MaxPriceN: int32(op.MaxPrice.N),
		MaxPriceD: int32(op.MaxPrice.D),
	}

	var err error
	liquidityPoolID := PoolIDToString(op.LiquidityPoolId)

	liquidityPoolDepositDetail.LiquidityPoolID = liquidityPoolID

	var (
		assetA, assetB         xdr.Asset
		depositedA, depositedB xdr.Int64
		sharesReceived         xdr.Int64
		lp                     *xdr.LiquidityPoolEntry
		delta                  *LiquidityPoolDelta
	)
	if o.Transaction.Successful() {
		// we will use the defaults (omitted asset and 0 amounts) if the transaction failed
		lp, delta, err = o.getLiquidityPoolAndProductDelta(&op.LiquidityPoolId)
		if err != nil {
			return LiquidityPoolDepositDetail{}, err
		}

		params := lp.Body.ConstantProduct.Params
		assetA, assetB = params.AssetA, params.AssetB
		depositedA, depositedB = delta.ReserveA, delta.ReserveB
		sharesReceived = delta.TotalPoolShares
	}

	// Process ReserveA Details
	var assetACode, assetAIssuer, assetAType string
	err = assetA.Extract(&assetAType, &assetACode, &assetAIssuer)
	if err != nil {
		return LiquidityPoolDepositDetail{}, err
	}

	liquidityPoolDepositDetail.ReserveAssetA.AssetCode = assetACode
	liquidityPoolDepositDetail.ReserveAssetA.AssetIssuer = assetAIssuer
	liquidityPoolDepositDetail.ReserveAssetA.AssetType = assetAType
	liquidityPoolDepositDetail.ReserveAssetA.DepositAmount = int64(depositedA)

	//Process ReserveB Details
	var assetBCode, assetBIssuer, assetBType string
	err = assetB.Extract(&assetBType, &assetBCode, &assetBIssuer)
	if err != nil {
		return LiquidityPoolDepositDetail{}, err
	}

	liquidityPoolDepositDetail.ReserveAssetB.AssetCode = assetBCode
	liquidityPoolDepositDetail.ReserveAssetB.AssetIssuer = assetBIssuer
	liquidityPoolDepositDetail.ReserveAssetB.AssetType = assetBType
	liquidityPoolDepositDetail.ReserveAssetB.DepositAmount = int64(depositedB)

	liquidityPoolDepositDetail.MinPrice, err = strconv.ParseFloat(op.MinPrice.String(), 64)
	if err != nil {
		return LiquidityPoolDepositDetail{}, err
	}

	liquidityPoolDepositDetail.MaxPrice, err = strconv.ParseFloat(op.MaxPrice.String(), 64)
	if err != nil {
		return LiquidityPoolDepositDetail{}, err
	}

	liquidityPoolDepositDetail.SharesReceived = int64(sharesReceived)

	return liquidityPoolDepositDetail, nil
}
