package adapters

import (
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/protocols/horizon/operations"
)

func populateInflationOperation(op *common.Operation, baseOp operations.Base) (operations.Inflation, error) {
	baseOp.Type = "inflation"

	return operations.Inflation{
		Base: baseOp,
	}, nil
}
