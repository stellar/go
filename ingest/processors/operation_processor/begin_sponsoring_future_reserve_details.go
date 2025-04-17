package operation

import "fmt"

type BeginSponsoringFutureReservesDetail struct {
	SponsoredID string `json:"sponsored_id"`
}

func (o *LedgerOperation) BeginSponsoringFutureReservesDetails() (BeginSponsoringFutureReservesDetail, error) {
	op, ok := o.Operation.Body.GetBeginSponsoringFutureReservesOp()
	if !ok {
		return BeginSponsoringFutureReservesDetail{}, fmt.Errorf("could not access BeginSponsoringFutureReserves info for this operation (index %d)", o.OperationIndex)
	}

	return BeginSponsoringFutureReservesDetail{
		SponsoredID: op.SponsoredId.Address(),
	}, nil
}
