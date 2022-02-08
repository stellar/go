package adapters

import (
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/protocols/horizon/operations"
)

func populateBeginSponsoringFutureReservesOperation(op *common.Operation, baseOp operations.Base) (operations.BeginSponsoringFutureReserves, error) {
	beginSponsoringFutureReserves := op.Get().Body.MustBeginSponsoringFutureReservesOp()

	return operations.BeginSponsoringFutureReserves{
		Base:        baseOp,
		SponsoredID: beginSponsoringFutureReserves.SponsoredId.Address(),
	}, nil
}
