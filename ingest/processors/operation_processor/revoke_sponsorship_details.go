package operation

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

type RevokeSponsorshipDetail struct {
	SignerAccountID  string          `json:"signer_account_id"`
	SignerKey        string          `json:"signer_key"`
	LedgerKeyDetails LedgerKeyDetail `json:"ledger_key_detail"`
}

func (o *LedgerOperation) RevokeSponsorshipDetails() (RevokeSponsorshipDetail, error) {
	op, ok := o.Operation.Body.GetRevokeSponsorshipOp()
	if !ok {
		return RevokeSponsorshipDetail{}, fmt.Errorf("could not access RevokeSponsorship info for this operation (index %d)", o.OperationIndex)
	}

	var revokeSponsorshipDetail RevokeSponsorshipDetail

	switch op.Type {
	case xdr.RevokeSponsorshipTypeRevokeSponsorshipLedgerEntry:
		ledgerKeyDetail, err := addLedgerKey(*op.LedgerKey)
		if err != nil {
			return RevokeSponsorshipDetail{}, err
		}

		revokeSponsorshipDetail.LedgerKeyDetails = ledgerKeyDetail
	case xdr.RevokeSponsorshipTypeRevokeSponsorshipSigner:
		revokeSponsorshipDetail.SignerAccountID = op.Signer.AccountId.Address()
		revokeSponsorshipDetail.SignerKey = op.Signer.SignerKey.Address()
	}

	return revokeSponsorshipDetail, nil
}
