package adapters

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/operations"
)

func populateCreateClaimableBalanceOperation(op *common.Operation, baseOp operations.Base) (operations.CreateClaimableBalance, error) {
	createClaimableBalance := op.Get().Body.MustCreateClaimableBalanceOp()

	claimants := make([]horizon.Claimant, len(createClaimableBalance.Claimants))
	for i, claimant := range createClaimableBalance.Claimants {
		claimants[i] = horizon.Claimant{
			Destination: claimant.MustV0().Destination.Address(),
			Predicate:   claimant.MustV0().Predicate,
		}
	}

	return operations.CreateClaimableBalance{
		Base:      baseOp,
		Asset:     createClaimableBalance.Asset.StringCanonical(),
		Amount:    amount.String(createClaimableBalance.Amount),
		Claimants: claimants,
	}, nil
}
