// Note: This is placed in the xdrill directory/package just for this example
// Processors may be placed in a different location/package; To be discussed
package xdrill

import (
	"fmt"
	"time"

	"github.com/stellar/go/exp/xdrill/ledger"
	"github.com/stellar/go/xdr"
)

type LedgerClosedOutput struct {
	Sequence                   uint32    `json:"sequence"` // sequence number of the ledger
	LedgerHash                 string    `json:"ledger_hash"`
	PreviousLedgerHash         string    `json:"previous_ledger_hash"`
	LedgerHeader               string    `json:"ledger_header"` // base 64 encoding of the ledger header
	TransactionCount           int32     `json:"transaction_count"`
	OperationCount             int32     `json:"operation_count"` // counts only operations that were a part of successful transactions
	SuccessfulTransactionCount int32     `json:"successful_transaction_count"`
	FailedTransactionCount     int32     `json:"failed_transaction_count"`
	TxSetOperationCount        string    `json:"tx_set_operation_count"` // counts all operations, even those that are part of failed transactions
	ClosedAt                   time.Time `json:"closed_at"`              // UTC timestamp
	TotalCoins                 int64     `json:"total_coins"`
	FeePool                    int64     `json:"fee_pool"`
	BaseFee                    uint32    `json:"base_fee"`
	BaseReserve                uint32    `json:"base_reserve"`
	MaxTxSetSize               uint32    `json:"max_tx_set_size"`
	ProtocolVersion            uint32    `json:"protocol_version"`
	LedgerID                   int64     `json:"id"`
	SorobanFeeWrite1Kb         int64     `json:"soroban_fee_write_1kb"`
	NodeID                     string    `json:"node_id"`
	Signature                  string    `json:"signature"`
	TotalByteSizeOfBucketList  uint64    `json:"total_byte_size_of_bucket_list"`
}

func TransformLedger(lcm xdr.LedgerCloseMeta) (LedgerClosedOutput, error) {
	ledger := ledger.Ledger{
		LedgerCloseMeta: lcm,
	}

	outputLedgerHeader, err := xdr.MarshalBase64(ledger.LedgerHeaderHistoryEntry().Header)
	if err != nil {
		return LedgerClosedOutput{}, err
	}

	outputSuccessfulTransactionCount, outputFailedTransactionCount, ok := ledger.TransactionCounts()
	if !ok {
		return LedgerClosedOutput{}, fmt.Errorf("could not get transaction counts")
	}

	outputOperationCount, outputTxSetOperationCount, ok := ledger.OperationCounts()
	if !ok {
		return LedgerClosedOutput{}, fmt.Errorf("could not get operation counts")
	}

	var outputSorobanFeeWrite1Kb int64
	sorobanFeeWrite1Kb, ok := ledger.SorobanFeeWrite1Kb()
	if ok {
		outputSorobanFeeWrite1Kb = sorobanFeeWrite1Kb
	}

	var outputTotalByteSizeOfBucketList uint64
	totalByteSizeOfBucketList, ok := ledger.TotalByteSizeOfBucketList()
	if ok {
		outputTotalByteSizeOfBucketList = totalByteSizeOfBucketList
	}

	var outputNodeID string
	nodeID, ok := ledger.NodeID()
	if ok {
		outputNodeID = nodeID
	}

	var outputSigature string
	signature, ok := ledger.Signature()
	if ok {
		outputSigature = signature
	}

	ledgerOutput := LedgerClosedOutput{
		Sequence:                   ledger.LedgerSequence(),
		LedgerHash:                 ledger.Hash(),
		PreviousLedgerHash:         ledger.Hash(),
		LedgerHeader:               outputLedgerHeader,
		TransactionCount:           outputSuccessfulTransactionCount,
		OperationCount:             outputOperationCount,
		SuccessfulTransactionCount: outputSuccessfulTransactionCount,
		FailedTransactionCount:     outputFailedTransactionCount,
		TxSetOperationCount:        string(outputTxSetOperationCount),
		ClosedAt:                   ledger.ClosedAt(),
		TotalCoins:                 ledger.TotalCoins(),
		FeePool:                    ledger.FeePool(),
		BaseFee:                    ledger.BaseFee(),
		BaseReserve:                ledger.BaseReserve(),
		MaxTxSetSize:               ledger.MaxTxSetSize(),
		ProtocolVersion:            ledger.LedgerVersion(),
		LedgerID:                   ledger.ID(),
		SorobanFeeWrite1Kb:         outputSorobanFeeWrite1Kb,
		NodeID:                     outputNodeID,
		Signature:                  outputSigature,
		TotalByteSizeOfBucketList:  outputTotalByteSizeOfBucketList,
	}

	return ledgerOutput, nil
}
