package operation

import "fmt"

type BumpSequenceDetails struct {
	BumpTo int64 `json:"bump_to,string"`
}

func (o *LedgerOperation) BumpSequenceDetails() (BumpSequenceDetails, error) {
	op, ok := o.Operation.Body.GetBumpSequenceOp()
	if !ok {
		return BumpSequenceDetails{}, fmt.Errorf("could not access BumpSequence info for this operation (index %d)", o.OperationIndex)
	}

	return BumpSequenceDetails{
		BumpTo: int64(op.BumpTo),
	}, nil
}
