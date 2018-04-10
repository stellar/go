package bridge

import (
	b "github.com/stellar/go/build"
	"github.com/stellar/go/services/internal/bridge-compliance-shared/protocols"
)

// ChangeTrustOperationBody represents change_trust operation
type ChangeTrustOperationBody struct {
	Source *string
	Asset  protocols.Asset
	// nil means max limit
	Limit *string
}

// ToTransactionMutator returns go-stellar-base TransactionMutator
func (op ChangeTrustOperationBody) ToTransactionMutator() b.TransactionMutator {
	mutators := []interface{}{
		op.Asset.ToBaseAsset(),
	}

	if op.Limit == nil {
		// Set MaxLimit
		mutators = append(mutators, b.MaxLimit)
	} else {
		mutators = append(mutators, b.Limit(*op.Limit))
	}

	if op.Source != nil {
		mutators = append(mutators, b.SourceAccount{*op.Source})
	}

	return b.ChangeTrust(mutators...)
}

// Validate validates if operation body is valid.
func (op ChangeTrustOperationBody) Validate() error {
	panic("TODO")
	// if !op.Asset.Validate() {
	// 	return helpers.NewInvalidParameterError("asset", op.Asset.String(), "Asset is invalid.")
	// }

	// if op.Limit != nil {
	// 	if !helpers.IsValidAmount(*op.Limit) {
	// 		return helpers.NewInvalidParameterError("limit", *op.Limit, "Limit is not a valid amount.")
	// 	}
	// }

	// if op.Source != nil && !helpers.IsValidAccountID(*op.Source) {
	// 	return helpers.NewInvalidParameterError("source", *op.Source, "Source must be a public key (starting with `G`).")
	// }

	// return nil
}
