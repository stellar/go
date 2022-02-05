package adapters

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func populatePathPaymentStrictReceiveOperation(
	op *xdr.Operation,
	baseOp operations.Base,
) (operations.PathPayment, error) {
	payment := op.Body.PathPaymentStrictReceiveOp
	baseOp.Type = "path_payment_strict_receive"

	var (
		sendAssetType string
		sendCode      string
		sendIssuer    string
	)
	err := payment.SendAsset.Extract(&sendAssetType, &sendCode, &sendIssuer)
	if err != nil {
		return operations.PathPayment{}, errors.Wrap(err, "xdr.Asset.Extract error")
	}

	var (
		destAssetType string
		destCode      string
		destIssuer    string
	)
	err = payment.DestAsset.Extract(&destAssetType, &destCode, &destIssuer)
	if err != nil {
		return operations.PathPayment{}, errors.Wrap(err, "xdr.Asset.Extract error")
	}

	return operations.PathPayment{
		Payment: operations.Payment{
			Base: baseOp,
			To:   payment.Destination.Address(),
			Asset: base.Asset{
				Type:   assetType,
				Code:   code,
				Issuer: issuer,
			},
			Amount: amount.StringFromInt64(int64(payment.Amount)),
		},
	}, nil
}
