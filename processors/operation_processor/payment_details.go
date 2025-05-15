package operation

import (
	"fmt"
)

type PaymentDetail struct {
	From        string `json:"from"`
	FromMuxed   string `json:"from_muxed"`
	FromMuxedID uint64 `json:"from_muxed_id,string"`
	To          string `json:"to"`
	ToMuxed     string `json:"to_muxed"`
	ToMuxedID   uint64 `json:"to_muxed_id,string"`
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
	AssetType   string `json:"asset_type"`
	Amount      int64  `json:"amount,string"`
}

func (o *LedgerOperation) PaymentDetails() (PaymentDetail, error) {
	op, ok := o.Operation.Body.GetPaymentOp()
	if !ok {
		return PaymentDetail{}, fmt.Errorf("could not access Payment info for this operation (index %d)", o.OperationIndex)
	}

	paymentDetail := PaymentDetail{
		From:   o.SourceAccount(),
		To:     op.Destination.Address(),
		Amount: int64(op.Amount),
	}

	var err error
	var fromMuxed string
	var fromMuxedID uint64
	fromMuxed, fromMuxedID, err = getMuxedAccountDetails(o.sourceAccountXDR())
	if err != nil {
		return PaymentDetail{}, err
	}

	paymentDetail.FromMuxed = fromMuxed
	paymentDetail.FromMuxedID = fromMuxedID

	var toMuxed string
	var toMuxedID uint64
	toMuxed, toMuxedID, err = getMuxedAccountDetails(op.Destination)
	if err != nil {
		return PaymentDetail{}, err
	}

	paymentDetail.ToMuxed = toMuxed
	paymentDetail.ToMuxedID = toMuxedID

	var assetCode, assetIssuer, assetType string
	err = op.Asset.Extract(&assetType, &assetCode, &assetIssuer)
	if err != nil {
		return PaymentDetail{}, err
	}

	paymentDetail.AssetCode = assetCode
	paymentDetail.AssetIssuer = assetIssuer
	paymentDetail.AssetType = assetType

	return paymentDetail, nil
}
