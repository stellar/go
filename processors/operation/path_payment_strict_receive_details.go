package operation

import (
	"fmt"
)

type PathPaymentStrictReceiveDetail struct {
	From              string `json:"from"`
	FromMuxed         string `json:"from_muxed"`
	FromMuxedID       uint64 `json:"from_muxed_id,string"`
	To                string `json:"to"`
	ToMuxed           string `json:"to_muxed"`
	ToMuxedID         uint64 `json:"to_muxed_id,string"`
	AssetCode         string `json:"asset_code"`
	AssetIssuer       string `json:"asset_issuer"`
	AssetType         string `json:"asset_type"`
	Amount            int64  `json:"amount,string"`
	SourceAssetCode   string `json:"source_asset_code"`
	SourceAssetIssuer string `json:"source_asset_issuer"`
	SourceAssetType   string `json:"source_asset_type"`
	SourceAmount      int64  `json:"source_amount,string"`
	SourceMax         int64  `json:"source_max,string"`
	Path              []Path `json:"path"`
}

func (o *LedgerOperation) PathPaymentStrictReceiveDetails() (PathPaymentStrictReceiveDetail, error) {
	op, ok := o.Operation.Body.GetPathPaymentStrictReceiveOp()
	if !ok {
		return PathPaymentStrictReceiveDetail{}, fmt.Errorf("could not access PathPaymentStrictReceive info for this operation (index %d)", o.OperationIndex)
	}

	pathPaymentStrictReceiveDetail := PathPaymentStrictReceiveDetail{
		From:      o.SourceAccount(),
		To:        op.Destination.Address(),
		Amount:    int64(op.DestAmount),
		SourceMax: int64(op.SendMax),
	}

	var err error
	var fromMuxed string
	var fromMuxedID uint64
	fromMuxed, fromMuxedID, err = getMuxedAccountDetails(o.sourceAccountXDR())
	if err != nil {
		return PathPaymentStrictReceiveDetail{}, err
	}

	pathPaymentStrictReceiveDetail.FromMuxed = fromMuxed
	pathPaymentStrictReceiveDetail.FromMuxedID = fromMuxedID

	var toMuxed string
	var toMuxedID uint64
	toMuxed, toMuxedID, err = getMuxedAccountDetails(op.Destination)
	if err != nil {
		return PathPaymentStrictReceiveDetail{}, err
	}

	pathPaymentStrictReceiveDetail.ToMuxed = toMuxed
	pathPaymentStrictReceiveDetail.ToMuxedID = toMuxedID

	var assetCode, assetIssuer, assetType string
	err = op.DestAsset.Extract(&assetType, &assetCode, &assetIssuer)
	if err != nil {
		return PathPaymentStrictReceiveDetail{}, err
	}

	pathPaymentStrictReceiveDetail.AssetCode = assetCode
	pathPaymentStrictReceiveDetail.AssetIssuer = assetIssuer
	pathPaymentStrictReceiveDetail.AssetType = assetType

	var sourceAssetCode, sourceAssetIssuer, sourceAssetType string
	err = op.SendAsset.Extract(&sourceAssetType, &sourceAssetCode, &sourceAssetIssuer)
	if err != nil {
		return PathPaymentStrictReceiveDetail{}, err
	}

	pathPaymentStrictReceiveDetail.SourceAssetCode = sourceAssetCode
	pathPaymentStrictReceiveDetail.SourceAssetIssuer = sourceAssetIssuer
	pathPaymentStrictReceiveDetail.SourceAssetType = sourceAssetType

	if o.Transaction.Successful() {
		allOperationResults, ok := o.Transaction.Result.OperationResults()
		if !ok {
			return PathPaymentStrictReceiveDetail{}, fmt.Errorf("could not access any results for this transaction")
		}
		currentOperationResult := allOperationResults[o.OperationIndex]
		resultBody, ok := currentOperationResult.GetTr()
		if !ok {
			return PathPaymentStrictReceiveDetail{}, fmt.Errorf("could not access result body for this operation (index %d)", o.OperationIndex)
		}
		result, ok := resultBody.GetPathPaymentStrictReceiveResult()
		if !ok {
			return PathPaymentStrictReceiveDetail{}, fmt.Errorf("could not access PathPaymentStrictReceive result info for this operation (index %d)", o.OperationIndex)
		}
		pathPaymentStrictReceiveDetail.SourceAmount = int64(result.SendAmount())
	}

	pathPaymentStrictReceiveDetail.Path = o.TransformPath(op.Path)

	return pathPaymentStrictReceiveDetail, nil
}
