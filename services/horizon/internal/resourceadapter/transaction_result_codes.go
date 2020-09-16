package resourceadapter

import (
	"context"
	"fmt"

	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/txsub"
)

// Populate fills out the details
func PopulateTransactionResultCodes(ctx context.Context,
	transactionHash string,
	dest *protocol.TransactionResultCodes,
	fail *txsub.FailedTransactionError,
) (err error) {

	result, err := fail.Result()
	if err != nil {
		return
	}
	fmt.Print(result)

	dest.TransactionCode, err = fail.TransactionResultCode(transactionHash)
	if err != nil {
		return
	}

	dest.OperationCodes, err = fail.OperationResultCodes()
	if err != nil {
		return
	}

	return
}
