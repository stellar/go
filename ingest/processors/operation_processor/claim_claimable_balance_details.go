package operation

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

type ClaimClaimableBalanceDetail struct {
	BalanceID       string `json:"balance_id"`
	Claimant        string `json:"claimant"`
	ClaimantMuxed   string `json:"claimant_muxed"`
	ClaimantMuxedID uint64 `json:"claimant_muxed_id,string"`
}

func (o *LedgerOperation) ClaimClaimableBalanceDetails() (ClaimClaimableBalanceDetail, error) {
	op, ok := o.Operation.Body.GetClaimClaimableBalanceOp()
	if !ok {
		return ClaimClaimableBalanceDetail{}, fmt.Errorf("could not access ClaimClaimableBalance info for this operation (index %d)", o.OperationIndex)
	}

	claimClaimableBalanceDetail := ClaimClaimableBalanceDetail{
		Claimant: o.SourceAccount(),
	}

	var err error
	var balanceID string
	balanceID, err = xdr.MarshalBase64(op.BalanceId)
	if err != nil {
		return ClaimClaimableBalanceDetail{}, err
	}

	claimClaimableBalanceDetail.BalanceID = balanceID

	var claimantMuxed string
	var claimantMuxedID uint64
	claimantMuxed, claimantMuxedID, err = getMuxedAccountDetails(o.sourceAccountXDR())
	if err != nil {
		return ClaimClaimableBalanceDetail{}, err
	}

	claimClaimableBalanceDetail.ClaimantMuxed = claimantMuxed
	claimClaimableBalanceDetail.ClaimantMuxedID = claimantMuxedID

	return claimClaimableBalanceDetail, nil
}
