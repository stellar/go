package adapters

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/support/errors"
)

func populatePaymentOperation(op *common.Operation, baseOp operations.Base) (operations.Payment, error) {
	payment := op.Get().Body.MustPaymentOp()

	var (
		assetType string
		code      string
		issuer    string
	)
	err := payment.Asset.Extract(&assetType, &code, &issuer)
	if err != nil {
		return operations.Payment{}, errors.Wrap(err, "xdr.Asset.Extract error")
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
}
