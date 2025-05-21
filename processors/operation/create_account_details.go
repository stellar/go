package operation

import (
	"fmt"
)

type CreateAccountDetail struct {
	Account         string `json:"account"`
	StartingBalance int64  `json:"starting_balance,string"`
	Funder          string `json:"funder"`
	FunderMuxed     string `json:"funder_muxed"`
	FunderMuxedID   uint64 `json:"funder_muxed_id,string"`
}

func (o *LedgerOperation) CreateAccountDetails() (CreateAccountDetail, error) {
	op, ok := o.Operation.Body.GetCreateAccountOp()
	if !ok {
		return CreateAccountDetail{}, fmt.Errorf("could not access CreateAccount info for this operation (index %d)", o.OperationIndex)
	}

	createAccountDetail := CreateAccountDetail{
		Account:         op.Destination.Address(),
		StartingBalance: int64(op.StartingBalance),
		Funder:          o.SourceAccount(),
	}

	funderMuxed, funderMuxedID, err := getMuxedAccountDetails(o.sourceAccountXDR())
	if err != nil {
		return CreateAccountDetail{}, err
	}

	createAccountDetail.FunderMuxed = funderMuxed
	createAccountDetail.FunderMuxedID = funderMuxedID

	return createAccountDetail, nil
}
