package adapters

import (
	"strconv"

	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/protocols/horizon/operations"
)

func populateBumpSequenceOperation(op *common.Operation, baseOp operations.Base) (operations.BumpSequence, error) {
	bumpSequence := op.Get().Body.MustBumpSequenceOp()

	return operations.BumpSequence{
		Base:   baseOp,
		BumpTo: strconv.FormatInt(int64(bumpSequence.BumpTo), 10),
	}, nil
}
