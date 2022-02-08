package adapters

import (
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func populateClaimClaimableBalanceOperation(op *common.Operation, baseOp operations.Base) (operations.ClaimClaimableBalance, error) {
	claimClaimableBalance := op.Get().Body.MustClaimClaimableBalanceOp()

	balanceID, err := xdr.MarshalHex(claimClaimableBalance.BalanceId)
	if err != nil {
		return operations.ClaimClaimableBalance{}, errors.New("invalid balanceId")
	}

	return operations.ClaimClaimableBalance{
		Base:      baseOp,
		BalanceID: balanceID,
		Claimant:  op.SourceAccount().Address(),
		// TODO
		ClaimantMuxed:   "",
		ClaimantMuxedID: 0,
	}, nil
}
