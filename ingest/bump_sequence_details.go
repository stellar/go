package ingest

import "fmt"

type BumpSequenceDetails struct {
	BumpTo string `json:"bump_to"`
}

func (o *LedgerOperation) BumpSequenceDetails() (BumpSequenceDetails, error) {
	op, ok := o.Operation.Body.GetBumpSequenceOp()
	if !ok {
		return BumpSequenceDetails{}, fmt.Errorf("could not access BumpSequence info for this operation (index %d)", o.OperationIndex)
	}

	return BumpSequenceDetails{
		BumpTo: fmt.Sprintf("%d", op.BumpTo),
	}, nil
}
