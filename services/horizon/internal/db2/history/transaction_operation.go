package history

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// TransactionOperation represents the data for a single operation within a transaction
type TransactionOperation struct {
	Index          uint32
	Transaction    io.LedgerTransaction
	Operation      xdr.Operation
	LedgerSequence uint32
}

// ID returns the ID for the operation.
func (op *TransactionOperation) ID() int64 {
	return toid.New(
		int32(op.LedgerSequence),
		int32(op.Transaction.Index),
		int32(op.Index),
	).ToInt64()
}

// TransactionID returns the id for the transaction related with this operation.
func (op *TransactionOperation) TransactionID() int64 {
	return toid.New(int32(op.LedgerSequence), int32(op.Transaction.Index), 0).ToInt64()
}

// SourceAccount returns the operation's source account.
func (op *TransactionOperation) SourceAccount() *xdr.AccountId {
	sourceAccount := op.Operation.SourceAccount
	if sourceAccount != nil {
		return sourceAccount
	}

	return &op.Transaction.Envelope.Tx.SourceAccount
}

// OperationType returns the operation type.
func (op *TransactionOperation) OperationType() xdr.OperationType {
	return op.Operation.Body.Type
}

// Details returns the operation details as a map which can be stored as JSON.
func (op *TransactionOperation) Details() map[string]interface{} {
	details := map[string]interface{}{}
	source := op.SourceAccount()
	pop := op.Operation.Body.MustPaymentOp()
	details["from"] = source.Address()
	details["to"] = pop.Destination.Address()
	details["amount"] = amount.String(pop.Amount)
	assetDetails(details, pop.Asset, "")

	return details
}

// assetDetails sets the details for `a` on `result` using keys with `prefix`
func assetDetails(result map[string]interface{}, a xdr.Asset, prefix string) error {
	var (
		assetType string
		code      string
		issuer    string
	)
	err := a.Extract(&assetType, &code, &issuer)
	if err != nil {
		err = errors.Wrap(err, "xdr.Asset.Extract error")
		return err
	}
	result[prefix+"asset_type"] = assetType

	if a.Type == xdr.AssetTypeAssetTypeNative {
		return nil
	}

	result[prefix+"asset_code"] = code
	result[prefix+"asset_issuer"] = issuer
	return nil
}
