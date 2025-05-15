package operation

type EndSponsoringFutureReserveDetail struct {
	BeginSponsor        string `json:"begin_sponsor"`
	BeginSponsorMuxed   string `json:"begin_sponsor_muxed"`
	BeginSponsorMuxedID uint64 `json:"begin_sponsor_muxed_id,string"`
}

func (o *LedgerOperation) EndSponsoringFutureReserveDetails() (EndSponsoringFutureReserveDetail, error) {
	var endSponsoringFutureReserveDetail EndSponsoringFutureReserveDetail

	beginSponsorOp := o.findInitatingBeginSponsoringOp()
	if beginSponsorOp != nil {
		endSponsoringFutureReserveDetail.BeginSponsor = o.SourceAccount()

		var err error
		var beginSponsorMuxed string
		var beginSponsorMuxedID uint64
		beginSponsorMuxed, beginSponsorMuxedID, err = getMuxedAccountDetails(o.sourceAccountXDR())
		if err != nil {
			return EndSponsoringFutureReserveDetail{}, err
		}

		endSponsoringFutureReserveDetail.BeginSponsorMuxed = beginSponsorMuxed
		endSponsoringFutureReserveDetail.BeginSponsorMuxedID = beginSponsorMuxedID
	}

	return endSponsoringFutureReserveDetail, nil
}

func (o *LedgerOperation) findInitatingBeginSponsoringOp() *SponsorshipOutput {
	if !o.Transaction.Successful() {
		// Failed transactions may not have a compliant sandwich structure
		// we can rely on (e.g. invalid nesting or a being operation with the wrong sponsoree ID)
		// and thus we bail out since we could return incorrect information.
		return nil
	}
	sponsoree := o.sourceAccountXDR().ToAccountId()
	operations := o.Transaction.Envelope.Operations()
	for i := int(o.OperationIndex) - 1; i >= 0; i-- {
		if beginOp, ok := operations[i].Body.GetBeginSponsoringFutureReservesOp(); ok &&
			beginOp.SponsoredId.Address() == sponsoree.Address() {
			result := SponsorshipOutput{
				Operation:      operations[i],
				OperationIndex: uint32(i),
			}
			return &result
		}
	}
	return nil
}
