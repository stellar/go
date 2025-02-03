package ingest

import (
	"fmt"
	"strconv"

	"github.com/stellar/go/xdr"
)

type LiquidityPoolDepositDetail struct {
	LiquidityPoolID       string  `json:"liquidity_pool_id"`
	ReserveAAssetCode     string  `json:"reserve_a_asset_code"`
	ReserveAAssetIssuer   string  `json:"reserve_a_asset_issuer"`
	ReserveAAssetType     string  `json:"reserve_a_asset_type"`
	ReserveAMaxAmount     int64   `json:"reserve_a_max_amount"`
	ReserveADepositAmount int64   `json:"reserve_a_deposit_amount"`
	ReserveBAssetCode     string  `json:"reserve_b_asset_code"`
	ReserveBAssetIssuer   string  `json:"reserve_b_asset_issuer"`
	ReserveBAssetType     string  `json:"reserve_b_asset_type"`
	ReserveBMaxAmount     int64   `json:"reserve_b_max_amount"`
	ReserveBDepositAmount int64   `json:"reserve_b_deposit_amount"`
	MinPriceN             int32   `json:"min_price_n"`
	MinPriceD             int32   `json:"min_price_d"`
	MinPrice              float64 `json:"min_price"`
	MaxPriceN             int32   `json:"max_price_n"`
	MaxPriceD             int32   `json:"max_price_d"`
	MaxPrice              float64 `json:"max_price"`
	SharesReceived        int64   `json:"shares_received"`
}

func (o *LedgerOperation) LiquidityPoolDepositDetails() (LiquidityPoolDepositDetail, error) {
	op, ok := o.Operation.Body.GetLiquidityPoolDepositOp()
	if !ok {
		return LiquidityPoolDepositDetail{}, fmt.Errorf("could not access LiquidityPoolDeposit info for this operation (index %d)", o.OperationIndex)
	}

	liquidityPoolDepositDetail := LiquidityPoolDepositDetail{
		ReserveAMaxAmount: int64(op.MaxAmountA),
		ReserveBMaxAmount: int64(op.MaxAmountB),
		MinPriceN:         int32(op.MinPrice.N),
		MinPriceD:         int32(op.MinPrice.D),
		MaxPriceN:         int32(op.MaxPrice.N),
		MaxPriceD:         int32(op.MaxPrice.D),
	}

	var err error
	var liquidityPoolID string
	liquidityPoolID, err = PoolIDToString(op.LiquidityPoolId)
	if err != nil {
		return LiquidityPoolDepositDetail{}, err
	}

	liquidityPoolDepositDetail.LiquidityPoolID = liquidityPoolID

	var (
		assetA, assetB         xdr.Asset
		depositedA, depositedB xdr.Int64
		sharesReceived         xdr.Int64
	)
	if o.Transaction.Successful() {
		// we will use the defaults (omitted asset and 0 amounts) if the transaction failed
		lp, delta, err := o.getLiquidityPoolAndProductDelta(&op.LiquidityPoolId)
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

	liquidityPoolDepositDetail.ReserveAAssetCode = assetACode
	liquidityPoolDepositDetail.ReserveAAssetIssuer = assetAIssuer
	liquidityPoolDepositDetail.ReserveAAssetType = assetAType
	liquidityPoolDepositDetail.ReserveADepositAmount = int64(depositedA)

	//Process ReserveB Details
	var assetBCode, assetBIssuer, assetBType string
	err = assetB.Extract(&assetBType, &assetBCode, &assetBIssuer)
	if err != nil {
		return LiquidityPoolDepositDetail{}, err
	}

	liquidityPoolDepositDetail.ReserveBAssetCode = assetBCode
	liquidityPoolDepositDetail.ReserveBAssetIssuer = assetBIssuer
	liquidityPoolDepositDetail.ReserveBAssetType = assetBType
	liquidityPoolDepositDetail.ReserveBDepositAmount = int64(depositedB)

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
