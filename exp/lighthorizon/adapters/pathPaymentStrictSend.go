package adapters

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/support/errors"
)

func populatePathPaymentStrictSendOperation(op *common.Operation, baseOp operations.Base) (operations.PathPaymentStrictSend, error) {
	payment := op.Get().Body.MustPathPaymentStrictSendOp()

	var (
		sendAssetType string
		sendCode      string
		sendIssuer    string
	)
	err := payment.SendAsset.Extract(&sendAssetType, &sendCode, &sendIssuer)
	if err != nil {
		return operations.PathPaymentStrictSend{}, errors.Wrap(err, "xdr.Asset.Extract error")
	}

	var (
		destAssetType string
		destCode      string
		destIssuer    string
	)
	err = payment.DestAsset.Extract(&destAssetType, &destCode, &destIssuer)
	if err != nil {
		return operations.PathPaymentStrictSend{}, errors.Wrap(err, "xdr.Asset.Extract error")
	}

	destAmount := amount.String(0)
	if op.TransactionResult.Successful() {
		result := op.OperationResult().MustPathPaymentStrictSendResult()
		destAmount = amount.String(result.DestAmount())
	}

	var path = make([]base.Asset, len(payment.Path))
	for i := range payment.Path {
		var (
			assetType string
			code      string
			issuer    string
		)
		err = payment.Path[i].Extract(&assetType, &code, &issuer)
		if err != nil {
			return operations.PathPaymentStrictSend{}, errors.Wrap(err, "xdr.Asset.Extract error")
		}

		path[i] = base.Asset{
			Type:   assetType,
			Code:   code,
			Issuer: issuer,
		}
	}

	return operations.PathPaymentStrictSend{
		Payment: operations.Payment{
			Base: baseOp,
			From: op.SourceAccount().Address(),
			To:   payment.Destination.Address(),
			Asset: base.Asset{
				Type:   destAssetType,
				Code:   destCode,
				Issuer: destIssuer,
			},
			Amount: destAmount,
		},
		Path:              path,
		SourceAmount:      amount.String(payment.SendAmount),
		DestinationMin:    amount.String(payment.DestMin),
		SourceAssetType:   sendAssetType,
		SourceAssetCode:   sendCode,
		SourceAssetIssuer: sendIssuer,
	}, nil
}
