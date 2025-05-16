package operation

import (
	"encoding/base64"
	"fmt"
)

type ManageDataDetail struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (o *LedgerOperation) ManageDataDetails() (ManageDataDetail, error) {
	op, ok := o.Operation.Body.GetManageDataOp()
	if !ok {
		return ManageDataDetail{}, fmt.Errorf("could not access GetManageData info for this operation (index %d)", o.OperationIndex)
	}

	manageDataDetail := ManageDataDetail{
		Name: string(op.DataName),
	}

	if op.DataValue != nil {
		manageDataDetail.Value = base64.StdEncoding.EncodeToString(*op.DataValue)
	}

	return manageDataDetail, nil
}
