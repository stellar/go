package operation

import (
	"fmt"
)

type CreateClaimableBalanceDetail struct {
	AssetCode   string     `json:"asset_code"`
	AssetIssuer string     `json:"asset_issuer"`
	AssetType   string     `json:"asset_type"`
	Amount      int64      `json:"amount,string"`
	Claimants   []Claimant `json:"claimants"`
}

func (o *LedgerOperation) CreateClaimableBalanceDetails() (CreateClaimableBalanceDetail, error) {
	op, ok := o.Operation.Body.GetCreateClaimableBalanceOp()
	if !ok {
		return CreateClaimableBalanceDetail{}, fmt.Errorf("could not access CreateClaimableBalance info for this operation (index %d)", o.OperationIndex)
	}

	createClaimableBalanceDetail := CreateClaimableBalanceDetail{
		Claimants: transformClaimants(op.Claimants),
		Amount:    int64(op.Amount),
	}

	var err error
	var assetCode, assetIssuer, assetType string
	err = op.Asset.Extract(&assetType, &assetCode, &assetIssuer)
	if err != nil {
		return CreateClaimableBalanceDetail{}, err
	}

	createClaimableBalanceDetail.AssetCode = assetCode
	createClaimableBalanceDetail.AssetIssuer = assetIssuer
	createClaimableBalanceDetail.AssetType = assetType

	return createClaimableBalanceDetail, nil
}
