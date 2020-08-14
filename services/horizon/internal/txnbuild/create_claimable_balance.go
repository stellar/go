package txnbuild

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// CreateClaimableBalance represents the Stellar create claimable balance operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type CreateClaimableBalance struct {
	Amount        string
	Asset         Asset
	Destinations  []string
	SourceAccount Account
}

// BuildXDR for CreateClaimableBalance returns a fully configured XDR Operation.
func (cb *CreateClaimableBalance) BuildXDR() (xdr.Operation, error) {
	xdrAsset, err := cb.Asset.ToXDR()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to set XDR 'Asset' field")
	}
	xdrAmount, err := amount.Parse(cb.Amount)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to parse 'Amount'")
	}

	claimants := []xdr.Claimant{}

	for _, d := range cb.Destinations {
		c := xdr.Claimant{
			Type: xdr.ClaimantTypeClaimantTypeV0,
			V0: &xdr.ClaimantV0{
				Predicate: xdr.ClaimPredicate{
					Type: xdr.ClaimPredicateTypeClaimPredicateUnconditional,
				},
			},
		}
		err = c.V0.Destination.SetAddress(d)
		if err != nil {
			return xdr.Operation{}, errors.Wrapf(err, "failed to set destination address: %s", d)
		}
		claimants = append(claimants, c)
	}

	xdrOp := xdr.CreateClaimableBalanceOp{
		Asset:     xdrAsset,
		Amount:    xdrAmount,
		Claimants: claimants,
	}

	opType := xdr.OperationTypeCreateClaimableBalance
	body, err := xdr.NewOperationBody(opType, xdrOp)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to build XDR OperationBody")
	}
	op := xdr.Operation{Body: body}
	SetOpSourceAccount(&op, cb.SourceAccount)
	return op, nil
}

// FromXDR for CreateClaimableBalance initializes the txnbuild struct from the corresponding xdr Operation.
func (cb *CreateClaimableBalance) FromXDR(xdrOp xdr.Operation) error {
	result, ok := xdrOp.Body.GetCreateClaimableBalanceOp()
	if !ok {
		return errors.New("error parsing create_claimable_balance operation from xdr")
	}

	cb.SourceAccount = accountFromXDR(xdrOp.SourceAccount)
	for _, c := range result.Claimants {
		claimant := c.MustV0()
		cb.Destinations = append(cb.Destinations, claimant.Destination.Address())

	}

	asset, err := assetFromXDR(result.Asset)
	if err != nil {
		return errors.Wrap(err, "error parsing asset in create_claimable_balance operation")
	}
	cb.Asset = asset
	cb.Amount = amount.String(result.Amount)

	return nil
}

// Validate for CreateClaimableBalance validates the required struct fields. It returns an error if any of the fields are
// invalid. Otherwise, it returns nil.
func (cb *CreateClaimableBalance) Validate() error {
	for _, d := range cb.Destinations {
		err := validateStellarPublicKey(d)
		if err != nil {
			return NewValidationError("Destinations", err.Error())
		}
	}

	err := validateAmount(cb.Amount)
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
func (cb *CreateClaimableBalance) GetSourceAccount() Account {
	return cb.SourceAccount
}
