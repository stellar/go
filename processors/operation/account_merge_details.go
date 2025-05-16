package operation

import "fmt"

type AccountMergeDetail struct {
	Account        string `json:"account"`
	AccountMuxed   string `json:"account_muxed"`
	AccountMuxedID uint64 `json:"account_muxed_id,string"`
	Into           string `json:"into"`
	IntoMuxed      string `json:"into_muxed"`
	IntoMuxedID    uint64 `json:"into_muxed_id,string"`
}

func (o *LedgerOperation) AccountMergeDetails() (AccountMergeDetail, error) {
	destinationAccount, ok := o.Operation.Body.GetDestination()
	if !ok {
		return AccountMergeDetail{}, fmt.Errorf("could not access Destination info for this operation (index %d)", o.OperationIndex)
	}

	accountMergeDetail := AccountMergeDetail{
		Account: o.SourceAccount(),
		Into:    destinationAccount.Address(),
	}

	var err error
	var accountMuxed string
	var accountMuxedID uint64
	accountMuxed, accountMuxedID, err = getMuxedAccountDetails(o.sourceAccountXDR())
	if err != nil {
		return AccountMergeDetail{}, err
	}

	accountMergeDetail.AccountMuxed = accountMuxed
	accountMergeDetail.AccountMuxedID = accountMuxedID

	var intoMuxed string
	var intoMuxedID uint64
	intoMuxed, intoMuxedID, err = getMuxedAccountDetails(destinationAccount)
	if err != nil {
		return AccountMergeDetail{}, err
	}

	accountMergeDetail.IntoMuxed = intoMuxed
	accountMergeDetail.IntoMuxedID = intoMuxedID

	return accountMergeDetail, nil
}
