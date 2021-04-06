package txnbuild

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

// Operation represents the operation types of the Stellar network.
type Operation interface {
	BuildXDR() (xdr.Operation, error)
	FromXDR(xdrOp xdr.Operation) error
	Validate() error
	GetSourceAccount() string
	BuildXDRWithSEP23() (xdr.Operation, error)
	FromXDRWithSEP23(xdrOp xdr.Operation) error
	ValidateWithSEP23() error
}

// SetOpSourceAccount sets the source account ID on an Operation.
func SetOpSourceAccount(op *xdr.Operation, sourceAccount string) {
	if sourceAccount == "" {
		return
	}
	var opSourceAccountID xdr.MuxedAccount
	opSourceAccountID.SetAddress(sourceAccount)
	op.SourceAccount = &opSourceAccountID
}

// SetOpSourceAccount sets the source account ID on an Operation, allowing M-strkeys (as defined in SEP23).
func SetOpSourceAccountWithSEP23(op *xdr.Operation, sourceAccount string) {
	if sourceAccount == "" {
		return
	}
	var opSourceAccountID xdr.MuxedAccount
	opSourceAccountID.SetAddressWithSEP23(sourceAccount)
	op.SourceAccount = &opSourceAccountID
}

// operationFromXDR returns a txnbuild Operation from its corresponding XDR operation
func operationFromXDR(xdrOp xdr.Operation, withSEP23 bool) (Operation, error) {
	var newOp Operation
	switch xdrOp.Body.Type {
	case xdr.OperationTypeCreateAccount:
		newOp = &CreateAccount{}
	case xdr.OperationTypePayment:
		newOp = &Payment{}
	case xdr.OperationTypePathPaymentStrictReceive:
		newOp = &PathPayment{}
	case xdr.OperationTypeManageSellOffer:
		newOp = &ManageSellOffer{}
	case xdr.OperationTypeCreatePassiveSellOffer:
		newOp = &CreatePassiveSellOffer{}
	case xdr.OperationTypeSetOptions:
		newOp = &SetOptions{}
	case xdr.OperationTypeChangeTrust:
		newOp = &ChangeTrust{}
	case xdr.OperationTypeAllowTrust:
		newOp = &AllowTrust{}
	case xdr.OperationTypeAccountMerge:
		newOp = &AccountMerge{}
	case xdr.OperationTypeInflation:
		newOp = &Inflation{}
	case xdr.OperationTypeManageData:
		newOp = &ManageData{}
	case xdr.OperationTypeBumpSequence:
		newOp = &BumpSequence{}
	case xdr.OperationTypeManageBuyOffer:
		newOp = &ManageBuyOffer{}
	case xdr.OperationTypePathPaymentStrictSend:
		newOp = &PathPaymentStrictSend{}
	case xdr.OperationTypeBeginSponsoringFutureReserves:
		newOp = &BeginSponsoringFutureReserves{}
	case xdr.OperationTypeEndSponsoringFutureReserves:
		newOp = &EndSponsoringFutureReserves{}
	case xdr.OperationTypeCreateClaimableBalance:
		newOp = &CreateClaimableBalance{}
	case xdr.OperationTypeClaimClaimableBalance:
		newOp = &ClaimClaimableBalance{}
	case xdr.OperationTypeRevokeSponsorship:
		newOp = &RevokeSponsorship{}
	case xdr.OperationTypeClawback:
		newOp = &Clawback{}
	case xdr.OperationTypeClawbackClaimableBalance:
		newOp = &ClawbackClaimableBalance{}
	case xdr.OperationTypeSetTrustLineFlags:
		newOp = &SetTrustLineFlags{}
	default:
		return nil, fmt.Errorf("unknown operation type: %d", xdrOp.Body.Type)
	}
	var err error
	if withSEP23 {
		err = newOp.FromXDRWithSEP23(xdrOp)
	} else {
		err = newOp.FromXDR(xdrOp)
	}
	return newOp, err
}

func accountFromXDR(account *xdr.MuxedAccount, withSEP23 bool) string {
	if account != nil {
		if withSEP23 {
			return account.SEP23Address()
		} else {
			aid := account.ToAccountId()
			return aid.Address()
		}
	}
	return ""
}
