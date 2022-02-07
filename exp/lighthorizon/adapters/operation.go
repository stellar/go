package adapters

import (
	"fmt"
	"time"

	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/xdr"
)

func PopulateOperation(op *common.Operation) (operations.Operation, error) {
	hash, err := op.TransactionHash()
	if err != nil {
		return nil, err
	}

	baseOp := operations.Base{
		TransactionSuccessful: op.TransactionResult.Successful(),
		SourceAccount:         op.SourceAccount(),
		LedgerCloseTime:       time.Unix(int64(op.LedgerHeader.ScpValue.CloseTime), 0).UTC(),
		TransactionHash:       hash,
		TypeI:                 int32(op.Get().Body.Type),
	}
	switch op.Get().Body.Type {
	case xdr.OperationTypeCreateAccount:
		return populateCreateAccountOperation(op, baseOp)
	case xdr.OperationTypePayment:
		return populatePaymentOperation(op, baseOp)
	case xdr.OperationTypePathPaymentStrictReceive:
		return populatePathPaymentStrictReceiveOperation(op, baseOp)
	case xdr.OperationTypePathPaymentStrictSend:
		return operations.PathPaymentStrictSend{
			Payment: operations.Payment{
				Base: baseOp,
			},
		}, nil
	case xdr.OperationTypeManageBuyOffer:
		return populateManageBuyOfferOperation(op, baseOp)
	case xdr.OperationTypeManageSellOffer:
		return populateManageSellOfferOperation(op, baseOp)
	case xdr.OperationTypeCreatePassiveSellOffer:
		return operations.CreatePassiveSellOffer{
			Offer: operations.Offer{
				Base: baseOp,
			},
		}, nil
	case xdr.OperationTypeSetOptions:
		return operations.SetOptions{
			Base: baseOp,
		}, nil
	case xdr.OperationTypeChangeTrust:
		return operations.ChangeTrust{
			Base: baseOp,
		}, nil
	case xdr.OperationTypeAllowTrust:
		return operations.AllowTrust{
			Base: baseOp,
		}, nil
	case xdr.OperationTypeAccountMerge:
		return operations.AccountMerge{
			Base: baseOp,
		}, nil
	case xdr.OperationTypeInflation:
		return operations.Inflation{
			Base: baseOp,
		}, nil
	case xdr.OperationTypeManageData:
		return operations.ManageData{
			Base: baseOp,
		}, nil
	case xdr.OperationTypeBumpSequence:
		return operations.BumpSequence{
			Base: baseOp,
		}, nil
	case xdr.OperationTypeCreateClaimableBalance:
		return operations.CreateClaimableBalance{
			Base: baseOp,
		}, nil
	case xdr.OperationTypeClaimClaimableBalance:
		return operations.ClaimClaimableBalance{
			Base: baseOp,
		}, nil
	case xdr.OperationTypeBeginSponsoringFutureReserves:
		return operations.BeginSponsoringFutureReserves{
			Base: baseOp,
		}, nil
	case xdr.OperationTypeEndSponsoringFutureReserves:
		return operations.EndSponsoringFutureReserves{
			Base: baseOp,
		}, nil
	case xdr.OperationTypeRevokeSponsorship:
		return operations.RevokeSponsorship{
			Base: baseOp,
		}, nil
	case xdr.OperationTypeClawback:
		return operations.Clawback{
			Base: baseOp,
		}, nil
	case xdr.OperationTypeClawbackClaimableBalance:
		return operations.ClawbackClaimableBalance{
			Base: baseOp,
		}, nil
	case xdr.OperationTypeSetTrustLineFlags:
		return operations.SetTrustLineFlags{
			Base: baseOp,
		}, nil
	case xdr.OperationTypeLiquidityPoolDeposit:
		return operations.LiquidityPoolDeposit{
			Base: baseOp,
		}, nil
	case xdr.OperationTypeLiquidityPoolWithdraw:
		return operations.LiquidityPoolWithdraw{
			Base: baseOp,
		}, nil
	default:
		return nil, fmt.Errorf("Unknown operation type: %s", op.Get().Body.Type)
	}
}
