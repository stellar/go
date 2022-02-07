package adapters

import (
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/protocols/horizon/operations"
)

func populateAccountMergeOperation(op *common.Operation, baseOp operations.Base) (operations.AccountMerge, error) {
	destination := op.Get().Body.MustDestination()
	baseOp.Type = "account_merge"

	return operations.AccountMerge{
		Base:    baseOp,
		Account: op.SourceAccount().Address(),
		Into:    destination.Address(),
		// TODO:
		AccountMuxed:   "",
		AccountMuxedID: 0,
		IntoMuxed:      "",
		IntoMuxedID:    0,
	}, nil
}
