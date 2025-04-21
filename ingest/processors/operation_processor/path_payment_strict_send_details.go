package operation

import (
	"fmt"
)

type PathPaymentStrictSendDetail struct {
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
	DestinationMin    int64  `json:"destination_min,string"`
	Path              []Path `json:"path"`
}

func (o *LedgerOperation) PathPaymentStrictSendDetails() (PathPaymentStrictSendDetail, error) {
	op, ok := o.Operation.Body.GetPathPaymentStrictSendOp()
	if !ok {
		return PathPaymentStrictSendDetail{}, fmt.Errorf("could not access PathPaymentStrictSend info for this operation (index %d)", o.OperationIndex)
	}

	pathPaymentStrictSendDetail := PathPaymentStrictSendDetail{
		From:           o.SourceAccount(),
		To:             op.Destination.Address(),
		SourceAmount:   int64(op.SendAmount),
		DestinationMin: int64(op.DestMin),
	}

	var err error
	var fromMuxed string
	var fromMuxedID uint64
	fromMuxed, fromMuxedID, err = getMuxedAccountDetails(o.sourceAccountXDR())
	if err != nil {
		return PathPaymentStrictSendDetail{}, err
	}

	pathPaymentStrictSendDetail.FromMuxed = fromMuxed
	pathPaymentStrictSendDetail.FromMuxedID = fromMuxedID

	var toMuxed string
	var toMuxedID uint64
	toMuxed, toMuxedID, err = getMuxedAccountDetails(op.Destination)
	if err != nil {
		return PathPaymentStrictSendDetail{}, err
	}

	pathPaymentStrictSendDetail.ToMuxed = toMuxed
	pathPaymentStrictSendDetail.ToMuxedID = toMuxedID

	var assetCode, assetIssuer, assetType string
	err = op.DestAsset.Extract(&assetType, &assetCode, &assetIssuer)
	if err != nil {
		return PathPaymentStrictSendDetail{}, err
	}

	pathPaymentStrictSendDetail.AssetCode = assetCode
	pathPaymentStrictSendDetail.AssetIssuer = assetIssuer
	pathPaymentStrictSendDetail.AssetType = assetType

	var sourceAssetCode, sourceAssetIssuer, sourceAssetType string
	err = op.SendAsset.Extract(&sourceAssetType, &sourceAssetCode, &sourceAssetIssuer)
	if err != nil {
		return PathPaymentStrictSendDetail{}, err
	}

	pathPaymentStrictSendDetail.SourceAssetCode = sourceAssetCode
	pathPaymentStrictSendDetail.SourceAssetIssuer = sourceAssetIssuer
	pathPaymentStrictSendDetail.SourceAssetType = sourceAssetType

	if o.Transaction.Successful() {
		allOperationResults, ok := o.Transaction.Result.OperationResults()
		if !ok {
			return PathPaymentStrictSendDetail{}, fmt.Errorf("could not access any results for this transaction")
		}
		currentOperationResult := allOperationResults[o.OperationIndex]
		resultBody, ok := currentOperationResult.GetTr()
		if !ok {
			return PathPaymentStrictSendDetail{}, fmt.Errorf("could not access result body for this operation (index %d)", o.OperationIndex)
		}
		result, ok := resultBody.GetPathPaymentStrictSendResult()
		if !ok {
			return PathPaymentStrictSendDetail{}, fmt.Errorf("could not access GetPathPaymentStrictSendResult result info for this operation (index %d)", o.OperationIndex)
		}
		pathPaymentStrictSendDetail.Amount = int64(result.DestAmount())
	}

	pathPaymentStrictSendDetail.Path = o.TransformPath(op.Path)

	return pathPaymentStrictSendDetail, nil
}
