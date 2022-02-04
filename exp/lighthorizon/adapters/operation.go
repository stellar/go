package adapters

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/network"
	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func PopulateOperation(
	op *xdr.Operation,
	transactionEnvelope *xdr.TransactionEnvelope,
	transactionResult *xdr.TransactionResult,
	ledgerHeader *xdr.LedgerHeader,
) (operations.Operation, error) {
	sourceAccount := transactionEnvelope.SourceAccount().ToAccountId().Address()
	if op.SourceAccount != nil {
		sourceAccount = op.SourceAccount.Address()
	}

	hash, err := network.HashTransactionInEnvelope(*transactionEnvelope, network.PublicNetworkPassphrase)
	if err != nil {
		return nil, err
	}

	baseOp := operations.Base{
		TransactionSuccessful: transactionResult.Successful(),
		SourceAccount:         sourceAccount,
		LedgerCloseTime:       time.Unix(int64(ledgerHeader.ScpValue.CloseTime), 0).UTC(),
		TransactionHash:       hex.EncodeToString(hash[:]),
	}
	switch op.Body.Type {
	case xdr.OperationTypeCreateAccount:
		createAccount := op.Body.CreateAccountOp
		baseOp.Type = "create_account"
		return operations.CreateAccount{
			Base:            baseOp,
			StartingBalance: amount.String(createAccount.StartingBalance),
			Funder:          sourceAccount,
			Account:         createAccount.Destination.Address(),
		}, nil
	case xdr.OperationTypePayment:
		payment := op.Body.PaymentOp
		var (
			assetType string
			code      string
			issuer    string
		)
		err := payment.Asset.Extract(&assetType, &code, &issuer)
		if err != nil {
			return nil, errors.Wrap(err, "xdr.Asset.Extract error")
		}

		return operations.Payment{
			Base: baseOp,
			To:   payment.Destination.Address(),
			Asset: base.Asset{
				Type:   assetType,
				Code:   code,
				Issuer: issuer,
			},
			Amount: amount.StringFromInt64(int64(payment.Amount)),
		}, nil
	case xdr.OperationTypePathPaymentStrictReceive:
		return operations.PathPaymentStrictSend{
			Payment: operations.Payment{
				Base: baseOp,
			},
		}, nil
	case xdr.OperationTypePathPaymentStrictSend:
		return operations.PathPaymentStrictSend{
			Payment: operations.Payment{
				Base: baseOp,
			},
		}, nil
	case xdr.OperationTypeManageBuyOffer:
		return operations.ManageBuyOffer{
			Offer: operations.Offer{
				Base: baseOp,
			},
		}, nil
	case xdr.OperationTypeManageSellOffer:
		return operations.ManageSellOffer{
			Offer: operations.Offer{
				Base: baseOp,
			},
		}, nil
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
		return nil, fmt.Errorf("Unknown operation type: %s", op.Body.Type)
	}
}
