// Note: This is placed in the xdrill directory/package just for this example
// Processors may be placed in a different location/package; To be discussed
package xdrill

import (
	"github.com/stellar/go/xdr"
)

func TransformLedgerXDR(lcm xdr.LedgerCloseMeta) (LedgerClosedOutput, error) {
	outputLedgerHeader, err := xdr.MarshalBase64(lcm.LedgerHeaderHistoryEntry().Header)
	if err != nil {
		return LedgerClosedOutput{}, err
	}

	var outputSorobanFeeWrite1Kb int64
	sorobanFeeWrite1Kb, ok := lcm.SorobanFeeWrite1Kb()
	if ok {
		outputSorobanFeeWrite1Kb = sorobanFeeWrite1Kb
	}

	var outputTotalByteSizeOfBucketList uint64
	totalByteSizeOfBucketList, ok := lcm.TotalByteSizeOfBucketList()
	if ok {
		outputTotalByteSizeOfBucketList = totalByteSizeOfBucketList
	}

	var outputNodeID string
	nodeID, ok := lcm.NodeID()
	if ok {
		outputNodeID = nodeID
	}

	var outputSigature string
	signature, ok := lcm.Signature()
	if ok {
		outputSigature = signature
	}

	ledgerOutput := LedgerClosedOutput{
		Sequence:                   lcm.LedgerSequence(),
		LedgerHash:                 lcm.LedgerHash().String(),
		PreviousLedgerHash:         lcm.PreviousLedgerHash().String(),
		LedgerHeader:               outputLedgerHeader,
		TransactionCount:           int32(lcm.CountTransactions()),
		OperationCount:             int32(lcm.CountOperations()),
		SuccessfulTransactionCount: int32(lcm.CountSuccessfulTransactions()),
		FailedTransactionCount:     int32(lcm.CountFailedTransactions()),
		TxSetOperationCount:        string(lcm.CountSuccessfulOperations()),
		ClosedAt:                   lcm.LedgerClosedAt(),
		TotalCoins:                 lcm.TotalCoins(),
		FeePool:                    lcm.FeePool(),
		BaseFee:                    lcm.BaseFee(),
		BaseReserve:                lcm.BaseReserve(),
		MaxTxSetSize:               lcm.MaxTxSetSize(),
		ProtocolVersion:            lcm.ProtocolVersion(),
		LedgerID:                   lcm.LedgerID(),
		SorobanFeeWrite1Kb:         outputSorobanFeeWrite1Kb,
		NodeID:                     outputNodeID,
		Signature:                  outputSigature,
		TotalByteSizeOfBucketList:  outputTotalByteSizeOfBucketList,
	}

	return ledgerOutput, nil
}
