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
	Asset         Asset
	SourceAccount Account
}

// BuildXDR for Payment returns a fully configured XDR Operation.
func (p *Payment) BuildXDR() (xdr.Operation, error) {
	var destAccountID xdr.AccountId

	err := destAccountID.SetAddress(p.Destination)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to set destination address")
	}

	xdrAmount, err := amount.Parse(p.Amount)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to parse amount")
	}

	if p.Asset == nil {
		return xdr.Operation{}, errors.New("you must specify an asset for payment")
	}
	xdrAsset, err := p.Asset.ToXDR()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to set asset type")
	}

	opType := xdr.OperationTypePayment
	xdrOp := xdr.PaymentOp{
		Destination: destAccountID,
		Amount:      xdrAmount,
		Asset:       xdrAsset,
	}
	body, err := xdr.NewOperationBody(opType, xdrOp)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to build XDR Operation")
	}
	op := xdr.Operation{Body: body}
	SetOpSourceAccount(&op, p.SourceAccount)
	return op, nil
}

// FromXDR for Payment initialises the txnbuild struct from the corresponding xdr Operation.
func (p *Payment) FromXDR(xdrOp xdr.Operation) error {
	result, ok := xdrOp.Body.GetPaymentOp()
	if !ok {
		return errors.New("error parsing payment operation from xdr")
	}

	if xdrOp.SourceAccount != nil {
		p.SourceAccount = &SimpleAccount{AccountID: xdrOp.SourceAccount.Address()}
	}

	p.Destination = result.Destination.Address()
	p.Amount = amount.String(result.Amount)

	asset, err := assetFromXDR(result.Asset)
	if err != nil {
		return errors.Wrap(err, "error parsing payment operation from xdr")
	}
	p.Asset = asset

	return nil
}
