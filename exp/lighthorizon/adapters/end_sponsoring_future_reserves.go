package adapters

import (
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/protocols/horizon/operations"
)

func populateEndSponsoringFutureReservesOperation(op *common.Operation, baseOp operations.Base) (operations.EndSponsoringFutureReserves, error) {
	return operations.EndSponsoringFutureReserves{
		Base:         baseOp,
		BeginSponsor: findInitatingSandwichSponsor(op),
		// TODO
		BeginSponsorMuxed:   "",
		BeginSponsorMuxedID: 0,
	}, nil
}

func findInitatingSandwichSponsor(op *common.Operation) string {
	if !op.TransactionResult.Successful() {
		// Failed transactions may not have a compliant sandwich structure
		// we can rely on (e.g. invalid nesting or a being operation with the wrong sponsoree ID)
		// and thus we bail out since we could return incorrect information.
		return ""
	}
	sponsoree := op.SourceAccount()
	operations := op.TransactionEnvelope.Operations()
	for i := int(op.OpIndex) - 1; i >= 0; i-- {
		if beginOp, ok := operations[i].Body.GetBeginSponsoringFutureReservesOp(); ok &&
			beginOp.SponsoredId.Address() == sponsoree.Address() {
			if operations[i].SourceAccount != nil {
				return operations[i].SourceAccount.Address()
			} else {
				return op.TransactionEnvelope.SourceAccount().ToAccountId().Address()
			}
		}
	}
	return ""
}
