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
	Asset         string // TODO: Not used yet
	destAccountID xdr.AccountId
	xdrAsset      xdr.Asset
	xdrOp         xdr.PaymentOp
}

// Init for Payment initialises the required XDR fields for this operation.
func (p *Payment) Init() error {
	err := p.destAccountID.SetAddress(p.Destination)
	if err != nil {
		return errors.Wrap(err, "Failed to set destination address")
	}
	p.xdrOp.Destination = p.destAccountID

	p.xdrOp.Amount, err = amount.Parse(p.Amount)
	if err != nil {
		return errors.Wrap(err, "Failed to parse amount")
	}

	// TODO: Generalise to non-native currencies
	p.xdrAsset, err = xdr.NewAsset(xdr.AssetTypeAssetTypeNative, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to set asset type")
	}

	return err
}

// NewXDROperationBody for Payment initialises the corresponding XDR body.
func (p *Payment) NewXDROperationBody() (xdr.OperationBody, error) {
	opType := xdr.OperationTypePayment
	return xdr.NewOperationBody(opType, p.xdrOp)
}
