package bridge

import (
	"github.com/stellar/go/amount"
	shared "github.com/stellar/go/services/internal/bridge-compliance-shared"
	"github.com/stellar/go/services/internal/bridge-compliance-shared/http/helpers"
	"github.com/stellar/go/services/internal/bridge-compliance-shared/protocols"
	"github.com/stellar/go/txnbuild"
)

// ChangeTrustOperationBody represents change_trust operation
type ChangeTrustOperationBody struct {
	Source *string
	Asset  protocols.Asset
	// nil means max limit
	Limit *string
}

// Build returns a txnbuild.Operation
func (op ChangeTrustOperationBody) Build() txnbuild.Operation {
	txnOp := txnbuild.ChangeTrust{
		Line: txnbuild.CreditAsset{Code: op.Asset.Code, Issuer: op.Asset.Issuer},
	}

	if op.Limit != nil {
		txnOp.Limit = *op.Limit
	}

	if op.Source != nil {
		txnOp.SourceAccount = &txnbuild.SimpleAccount{AccountID: *op.Source}
	}

	return &txnOp
}

// Validate validates if operation body is valid.
func (op ChangeTrustOperationBody) Validate() error {
	err := op.Asset.Validate()
	if err != nil {
		return helpers.NewInvalidParameterError("asset", err.Error())
	}

	if op.Limit != nil {
		_, err := amount.Parse(*op.Limit)
		if err != nil {
			return helpers.NewInvalidParameterError("limit", "Limit is not a valid amount.")
		}
	}

	if op.Source != nil && !shared.IsValidAccountID(*op.Source) {
		return helpers.NewInvalidParameterError("source", "Source must be a public key (starting with `G`).")
	}

	return nil
}
