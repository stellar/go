package txnbuild

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// PathPayment represents the Stellar path payment operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type PathPayment struct {
	SendAsset   *Asset
	SendMax     string
	Destination string
	DestAsset   *Asset
	DestAmount  string
	Path        []Asset
	xdrOp       xdr.PathPaymentOp
}

// BuildXDR for Payment returns a fully configured XDR Operation.
func (pp *PathPayment) BuildXDR() (xdr.Operation, error) {
	var err error

	// Set XDR send asset
	if pp.SendAsset == nil {
		return xdr.Operation{}, errors.New("You must specify an asset to send for payment")
	}
	xdrSendAsset, err := pp.SendAsset.ToXDR()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to set asset type")
	}
	pp.xdrOp.SendAsset = xdrSendAsset

	// Set XDR send max
	pp.xdrOp.SendMax, err = amount.Parse(pp.SendMax)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to parse maximum amount to send")
	}

	// Set XDR destination
	err = pp.xdrOp.Destination.SetAddress(pp.Destination)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to set destination address")
	}

	// Set XDR destination asset
	if pp.DestAsset == nil {
		return xdr.Operation{}, errors.New("You must specify an asset for destination account to receive")
	}
	xdrDestAsset, err := pp.DestAsset.ToXDR()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to set asset type")
	}
	pp.xdrOp.DestAsset = xdrDestAsset

	// Set XDR destination amount
	pp.xdrOp.DestAmount, err = amount.Parse(pp.DestAmount)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to parse amount of asset destination account receives")
	}

	// Set XDR path
	var xdrPath []xdr.Asset
	var xdrPathAsset xdr.Asset
	for _, asset := range pp.Path {
		xdrPathAsset, err = asset.ToXDR()
		if err != nil {
			return xdr.Operation{}, errors.Wrap(err, "Failed to set asset type")
		}
		xdrPath = append(xdrPath, xdrPathAsset)
	}
	pp.xdrOp.Path = xdrPath

	opType := xdr.OperationTypePathPayment
	body, err := xdr.NewOperationBody(opType, pp.xdrOp)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to build XDR OperationBody")
	}

	return xdr.Operation{Body: body}, nil
}
