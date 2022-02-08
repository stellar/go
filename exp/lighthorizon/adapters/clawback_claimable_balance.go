package adapters

import (
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func populateClawbackClaimableBalanceOperation(op *common.Operation, baseOp operations.Base) (operations.ClawbackClaimableBalance, error) {
	clawbackClaimableBalance := op.Get().Body.MustClawbackClaimableBalanceOp()

	balanceID, err := xdr.MarshalHex(clawbackClaimableBalance.BalanceId)
	if err != nil {
		return operations.ClawbackClaimableBalance{}, errors.Wrap(err, "invalid balanceId")
	}

	return operations.ClawbackClaimableBalance{
		Base:      baseOp,
		BalanceID: balanceID,
	}, nil
}
