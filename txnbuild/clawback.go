package txnbuild

import (
	"github.com/pkg/errors"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/xdr"
)

// Clawback represents the Stellar clawback operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type Clawback struct {
	From          string
	Amount        string
	Asset         Asset
	SourceAccount Account
}

// BuildXDR for Clawback returns a fully configured XDR Operation.
func (cb *Clawback) BuildXDR() (xdr.Operation, error) {
	var fromMuxedAccount xdr.MuxedAccount

	err := fromMuxedAccount.SetAddress(cb.From)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to set from address")
	}

	if cb.Asset == nil {
		return xdr.Operation{}, errors.New("you must specify an asset for the clawback")
	}
	// Validate this is an issued asset
	if cb.Asset.IsNative() {
		return xdr.Operation{}, errors.New("clawbacks don't support the native (XLM) asset")
	}

	xdrAmount, err := amount.Parse(cb.Amount)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to parse amount")
	}

	// Clawback uses an asset code, map to it
	xdrAsset := xdr.Asset{}

	assetCode, err := xdrAsset.ToAssetCode(cb.Asset.GetCode())
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "can't convert asset to asset code")
	}

	opType := xdr.OperationTypeClawback
	xdrOp := xdr.ClawbackOp{
		From:   fromMuxedAccount,
		Amount: xdrAmount,
		Asset:  assetCode,
	}
	body, err := xdr.NewOperationBody(opType, xdrOp)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to build XDR Operation")
	}
	op := xdr.Operation{Body: body}
	SetOpSourceAccount(&op, cb.SourceAccount)
	return op, nil
}

// FromXDR for Clawback initialises the txnbuild struct from the corresponding xdr Operation.
func (cb *Clawback) FromXDR(xdrOp xdr.Operation) error {
	result, ok := xdrOp.Body.GetClawbackOp()
	if !ok {
		return errors.New("error parsing clawback operation from xdr")
	}

	cb.SourceAccount = accountFromXDR(xdrOp.SourceAccount)
	fromAID := result.From.ToAccountId()
	cb.From = fromAID.Address()
	cb.Amount = amount.String(result.Amount)
	asset, err := assetCodeToCreditAsset(result.Asset)
	if err != nil {
		return errors.Wrap(err, "error parsing clawback operation from xdr")
	}
	cb.Asset = asset

	return nil
}

// Validate for Clawback validates the required struct fields. It returns an error if any
// of the fields are invalid. Otherwise, it returns nil.
func (cb *Clawback) Validate() error {
	_, err := xdr.AddressToAccountId(cb.From)
	if err != nil {
		return NewValidationError("From", err.Error())
	}

	err = validateAmount(cb.Amount)
	if err != nil {
		return NewValidationError("Amount", err.Error())
	}

	err = validateStellarAsset(cb.Asset)
	if err != nil {
		return NewValidationError("Asset", err.Error())
	}

	return nil
}

// GetSourceAccount returns the source account of the operation, or nil if not
// set.
func (cb *Clawback) GetSourceAccount() Account {
	return cb.SourceAccount
}
