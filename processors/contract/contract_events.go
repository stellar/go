package contract

import (
	"fmt"
	"time"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/processors/utils"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
)

// ContractEventOutput is a representation of soroban contract events and diagnostic events
type ContractEventOutput struct {
	TransactionHash          string                         `json:"transaction_hash"`
	TransactionID            int64                          `json:"transaction_id"`
	Successful               bool                           `json:"successful"`
	LedgerSequence           uint32                         `json:"ledger_sequence"`
	ClosedAt                 time.Time                      `json:"closed_at"`
	InSuccessfulContractCall bool                           `json:"in_successful_contract_call"`
	ContractId               string                         `json:"contract_id"`
	Type                     int32                          `json:"type"`
	TypeString               string                         `json:"type_string"`
	Topics                   map[string][]map[string]string `json:"topics"`
	TopicsDecoded            map[string][]map[string]string `json:"topics_decoded"`
	Data                     map[string]string              `json:"data"`
	DataDecoded              map[string]string              `json:"data_decoded"`
	ContractEventXDR         string                         `json:"contract_event_xdr"`
}

// TransformContractEvent converts a transaction's contract events and diagnostic events into a form suitable for BigQuery.
// It is known that contract events are a subset of the diagnostic events XDR definition. We are opting to call all of these events
// contract events for better clarity to data analytics users.
func TransformContractEvent(transaction ingest.LedgerTransaction, lhe xdr.LedgerHeaderHistoryEntry) ([]ContractEventOutput, error) {
	ledgerHeader := lhe.Header
	outputTransactionHash := utils.HashToHexString(transaction.Result.TransactionHash)
	outputLedgerSequence := uint32(ledgerHeader.LedgerSeq)

	transactionIndex := uint32(transaction.Index)

	outputTransactionID := toid.New(int32(outputLedgerSequence), int32(transactionIndex), 0).ToInt64()

	outputCloseTime, err := utils.TimePointToUTCTimeStamp(ledgerHeader.ScpValue.CloseTime)
	if err != nil {
		return []ContractEventOutput{}, fmt.Errorf("for ledger %d; transaction %d (transaction id=%d): %v", outputLedgerSequence, transactionIndex, outputTransactionID, err)
	}

	// GetDiagnosticEvents will return all contract events and diagnostic events emitted
	contractEvents, err := transaction.GetDiagnosticEvents()
	if err != nil {
		return []ContractEventOutput{}, err
	}

	var transformedContractEvents []ContractEventOutput

	for _, contractEvent := range contractEvents {
		var outputContractId string
		outputTopicsJson := make(map[string][]map[string]string, 1)
		outputTopicsDecodedJson := make(map[string][]map[string]string, 1)

		outputInSuccessfulContractCall := contractEvent.InSuccessfulContractCall
		event := contractEvent.Event
		outputType := event.Type
		outputTypeString := event.Type.String()

		eventTopics := event.Body.GetTopics()
		outputTopics, outputTopicsDecoded := xdr.SerializeScValArray(eventTopics)
		outputTopicsJson["topics"] = outputTopics
		outputTopicsDecodedJson["topics_decoded"] = outputTopicsDecoded

		eventData := event.Body.GetData()
		outputData, outputDataDecoded := eventData.Serialize()

		// Convert the xdr ContractId to string
		if event.ContractId != nil {
			outputContractId, _ = event.ContractId.ToContractAddress()
		}

		outputContractEventXDR, err := xdr.MarshalBase64(contractEvent)
		if err != nil {
			return []ContractEventOutput{}, err
		}

		outputTransactionID := toid.New(int32(outputLedgerSequence), int32(transactionIndex), 0).ToInt64()
		outputSuccessful := transaction.Result.Successful()

		transformedDiagnosticEvent := ContractEventOutput{
			TransactionHash:          outputTransactionHash,
			TransactionID:            outputTransactionID,
			Successful:               outputSuccessful,
			LedgerSequence:           outputLedgerSequence,
			ClosedAt:                 outputCloseTime,
			InSuccessfulContractCall: outputInSuccessfulContractCall,
			ContractId:               outputContractId,
			Type:                     int32(outputType),
			TypeString:               outputTypeString,
			Topics:                   outputTopicsJson,
			TopicsDecoded:            outputTopicsDecodedJson,
			Data:                     outputData,
			DataDecoded:              outputDataDecoded,
			ContractEventXDR:         outputContractEventXDR,
		}

		transformedContractEvents = append(transformedContractEvents, transformedDiagnosticEvent)
	}

	return transformedContractEvents, nil
}

// getEventTopics and getEventData functions have been moved to xdr/event.go
// as GetTopics() and GetData() methods on ContractEventBody

// SerializeScVal and SerializeScValArray have been moved to xdr/scval_helpers.go
// as methods on ScVal for better reusability across processors
