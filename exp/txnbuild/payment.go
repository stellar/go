package txnbuild

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// Payment represents the Stellar payment operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type Payment struct {
	Destination   string
	Amount        string
	Asset         *Asset
	destAccountID xdr.AccountId
}

// BuildXDR for Payment returns a fully configured XDR Operation.
func (p *Payment) BuildXDR() (xdr.Operation, error) {
	var err error
	var xdrOp xdr.PaymentOp

	err = p.destAccountID.SetAddress(p.Destination)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to set destination address")
	}
	xdrOp.Destination = p.destAccountID

	xdrOp.Amount, err = amount.Parse(p.Amount)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to parse amount")
	}

	if p.Asset == nil {
		return xdr.Operation{}, errors.New("You must specify an asset for payment")
	}
	xdrAsset, err := p.Asset.ToXDR()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to set asset type")
	}
	xdrOp.Asset = xdrAsset

	opType := xdr.OperationTypePayment
	body, err := xdr.NewOperationBody(opType, xdrOp)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to build XDR OperationBody")
	}

	return xdr.Operation{Body: body}, nil
}
