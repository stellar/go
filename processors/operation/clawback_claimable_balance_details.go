package operation

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

type ClawbackClaimableBalanceDetail struct {
	BalanceID string `json:"balance_id"`
}

func (o *LedgerOperation) ClawbackClaimableBalanceDetails() (ClawbackClaimableBalanceDetail, error) {
	op, ok := o.Operation.Body.GetClawbackClaimableBalanceOp()
	if !ok {
		return ClawbackClaimableBalanceDetail{}, fmt.Errorf("could not access ClawbackClaimableBalance info for this operation (index %d)", o.OperationIndex)
	}

	balanceID, err := xdr.MarshalBase64(op.BalanceId)
	if err != nil {
		return ClawbackClaimableBalanceDetail{}, err
	}

	return ClawbackClaimableBalanceDetail{BalanceID: balanceID}, nil
}
