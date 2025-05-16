package operation

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

type ChangeTrustDetail struct {
	AssetCode       string `json:"asset_code"`
	AssetIssuer     string `json:"asset_issuer"`
	AssetType       string `json:"asset_type"`
	LiquidityPoolID string `json:"liquidity_pool_id"`
	Limit           int64  `json:"limit,string"`
	Trustee         string `json:"trustee"`
	Trustor         string `json:"trustor"`
	TrustorMuxed    string `json:"trustor_muxed"`
	TrustorMuxedID  uint64 `json:"trustor_muxed_id,string"`
}

func (o *LedgerOperation) ChangeTrustDetails() (ChangeTrustDetail, error) {
	op, ok := o.Operation.Body.GetChangeTrustOp()
	if !ok {
		return ChangeTrustDetail{}, fmt.Errorf("could not access GetChangeTrust info for this operation (index %d)", o.OperationIndex)
	}

	var err error
	changeTrustDetail := ChangeTrustDetail{
		Trustor: o.SourceAccount(),
		Limit:   int64(op.Limit),
	}

	if op.Line.Type == xdr.AssetTypeAssetTypePoolShare {
		changeTrustDetail.AssetType, changeTrustDetail.LiquidityPoolID, err = getLiquidityPoolAssetDetails(*op.Line.LiquidityPool)
		if err != nil {
			return ChangeTrustDetail{}, err
		}
	} else {
		var assetCode, assetIssuer, assetType string
		err = op.Line.ToAsset().Extract(&assetType, &assetCode, &assetIssuer)
		if err != nil {
			return ChangeTrustDetail{}, err
		}

		changeTrustDetail.AssetCode = assetCode
		changeTrustDetail.AssetIssuer = assetIssuer
		changeTrustDetail.AssetType = assetType
		changeTrustDetail.Trustee = assetIssuer
	}

	var trustorMuxed string
	var trustorMuxedID uint64
	trustorMuxed, trustorMuxedID, err = getMuxedAccountDetails(o.sourceAccountXDR())
	if err != nil {
		return ChangeTrustDetail{}, err
	}

	changeTrustDetail.TrustorMuxed = trustorMuxed
	changeTrustDetail.TrustorMuxedID = trustorMuxedID

	return changeTrustDetail, nil
}

func getLiquidityPoolAssetDetails(lpp xdr.LiquidityPoolParameters) (string, string, error) {
	if lpp.Type != xdr.LiquidityPoolTypeLiquidityPoolConstantProduct {
		return "", "", fmt.Errorf("unknown liquidity pool type %d", lpp.Type)
	}

	cp := lpp.ConstantProduct

	var err error
	var poolID xdr.PoolId
	var poolIDString string
	poolID, err = xdr.NewPoolId(cp.AssetA, cp.AssetB, cp.Fee)
	if err != nil {
		return "", "", err
	}

	poolIDString = PoolIDToString(poolID)

	return "liquidity_pool_shares", poolIDString, nil
}
