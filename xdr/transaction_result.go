package xdr

import "github.com/stellar/go/support/errors"

// Successful returns true if the transaction succeeded
func (r TransactionResult) Successful() bool {
	return r.Result.Code == TransactionResultCodeTxSuccess ||
		r.Result.Code == TransactionResultCodeTxFeeBumpInnerSuccess
}

// OperationResults returns the operation results for the transaction
func (r TransactionResult) OperationResults() ([]OperationResult, bool) {
	innerResults, ok := r.Result.GetInnerResultPair()
	if ok {
		return innerResults.Result.Result.GetResults()
	}
	return r.Result.GetResults()
}

// Successful returns true if the transaction succeeded
func (r TransactionResultPair) Successful() bool {
	return r.Result.Successful()
}

// OperationResults returns the operation results for the transaction
func (r TransactionResultPair) OperationResults() ([]OperationResult, bool) {
	return r.Result.OperationResults()
}

// InnerHash returns the hash of the inner transaction.
// This function can only be called on fee bump transactions.
func (r TransactionResultPair) InnerHash() Hash {
	return r.Result.Result.MustInnerResultPair().TransactionHash
}

// ExtractBalanceId will parse the operation result at `opIndex` within the
// given `txResult`.
//
// If the specified operation index does not point to a successful
// `CreateClaimableBalance` operation result, this function panics.
func (r TransactionResult) ExtractBalanceID(opIndex int) (string, error) {
	opResults, ok := r.OperationResults()
	if !ok {
		return "", errors.New("Failed to retrieve transaction's operation results")
	}

	if opIndex < 0 || opIndex >= len(opResults) {
		return "", errors.New("Invalid operation index")
	}

	result := opResults[opIndex]
	balanceId, err := MarshalHex(result.MustTr().MustCreateClaimableBalanceResult().BalanceId)
	if err != nil {
		return "", errors.Wrap(err, "Failed to determine balance ID")
	}

	return balanceId, nil
}
