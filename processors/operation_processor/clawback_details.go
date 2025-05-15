package operation

import "fmt"

type ClawbackDetail struct {
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
	AssetType   string `json:"asset_type"`
	From        string `json:"from"`
	FromMuxed   string `json:"from_muxed"`
	FromMuxedID uint64 `json:"from_muxed_id,string"`
	Amount      int64  `json:"amount,string"`
}

func (o *LedgerOperation) ClawbackDetails() (ClawbackDetail, error) {
	op, ok := o.Operation.Body.GetClawbackOp()
	if !ok {
		return ClawbackDetail{}, fmt.Errorf("could not access Clawback info for this operation (index %d)", o.OperationIndex)
	}

	clawbackDetail := ClawbackDetail{
		Amount: int64(op.Amount),
		From:   op.From.Address(),
	}

	var err error
	var assetCode, assetIssuer, assetType string
	err = op.Asset.Extract(&assetType, &assetCode, &assetIssuer)
	if err != nil {
		return ClawbackDetail{}, err
	}

	clawbackDetail.AssetCode = assetCode
	clawbackDetail.AssetIssuer = assetIssuer
	clawbackDetail.AssetType = assetType

	var fromMuxed string
	var fromMuxedID uint64
	fromMuxed, fromMuxedID, err = getMuxedAccountDetails(op.From)
	if err != nil {
		return ClawbackDetail{}, err
	}

	clawbackDetail.FromMuxed = fromMuxed
	clawbackDetail.FromMuxedID = fromMuxedID

	return clawbackDetail, nil
}
