package configsetting

import (
	"fmt"
	"strconv"
	"time"

	"github.com/stellar/go/ingest"
	utils "github.com/stellar/go/ingest/processors/processor_utils"
	"github.com/stellar/go/xdr"
)

// ConfigSettingOutput is a representation of soroban config settings that aligns with the Bigquery table config_settings
type ConfigSettingOutput struct {
	ConfigSettingId                      int32               `json:"config_setting_id"`
	ContractMaxSizeBytes                 uint32              `json:"contract_max_size_bytes"`
	LedgerMaxInstructions                int64               `json:"ledger_max_instructions"`
	TxMaxInstructions                    int64               `json:"tx_max_instructions"`
	FeeRatePerInstructionsIncrement      int64               `json:"fee_rate_per_instructions_increment"`
	TxMemoryLimit                        uint32              `json:"tx_memory_limit"`
	LedgerMaxDiskReadEntries             uint32              `json:"ledger_max_disk_read_entries"`
	LedgerMaxDiskReadBytes               uint32              `json:"ledger_max_disk_read_bytes"`
	LedgerMaxWriteLedgerEntries          uint32              `json:"ledger_max_write_ledger_entries"`
	LedgerMaxWriteBytes                  uint32              `json:"ledger_max_write_bytes"`
	TxMaxDiskReadEntries                 uint32              `json:"tx_max_disk_read_entries"`
	TxMaxDiskReadBytes                   uint32              `json:"tx_max_disk_read_bytes"`
	TxMaxWriteLedgerEntries              uint32              `json:"tx_max_write_ledger_entries"`
	TxMaxWriteBytes                      uint32              `json:"tx_max_write_bytes"`
	FeeDiskReadLedgerEntry               int64               `json:"fee_disk_read_ledger_entry"`
	FeeWriteLedgerEntry                  int64               `json:"fee_write_ledger_entry"`
	FeeDiskRead1Kb                       int64               `json:"fee_disk_read_1kb"`
	SorobanStateTargetSizeBytes          int64               `json:"soroban_state_target_size_bytes"`
	RentFee1KbSorobanStateSizeLow        int64               `json:"rent_fee_1kb_soroban_state_size_low"`
	RentFee1KbSorobanStateSizeHigh       int64               `json:"rent_fee_1kb_soroban_state_size_high"`
	SorobanStateRentFeeGrowthFactor      uint32              `json:"soroban_state_rent_fee_growth_factor"`
	FeeHistorical1Kb                     int64               `json:"fee_historical_1kb"`
	TxMaxContractEventsSizeBytes         uint32              `json:"tx_max_contract_events_size_bytes"`
	FeeContractEvents1Kb                 int64               `json:"fee_contract_events_1kb"`
	LedgerMaxTxsSizeBytes                uint32              `json:"ledger_max_txs_size_bytes"`
	TxMaxSizeBytes                       uint32              `json:"tx_max_size_bytes"`
	FeeTxSize1Kb                         int64               `json:"fee_tx_size_1kb"`
	ContractCostParamsCpuInsns           []map[string]string `json:"contract_cost_params_cpu_insns"`
	ContractCostParamsMemBytes           []map[string]string `json:"contract_cost_params_mem_bytes"`
	ContractDataKeySizeBytes             uint32              `json:"contract_data_key_size_bytes"`
	ContractDataEntrySizeBytes           uint32              `json:"contract_data_entry_size_bytes"`
	MaxEntryTtl                          uint32              `json:"max_entry_ttl"`
	MinTemporaryTtl                      uint32              `json:"min_temporary_ttl"`
	MinPersistentTtl                     uint32              `json:"min_persistent_ttl"`
	AutoBumpLedgers                      uint32              `json:"auto_bump_ledgers"`
	PersistentRentRateDenominator        int64               `json:"persistent_rent_rate_denominator"`
	TempRentRateDenominator              int64               `json:"temp_rent_rate_denominator"`
	MaxEntriesToArchive                  uint32              `json:"max_entries_to_archive"`
	LiveSorobanStateSizeWindowSampleSize uint32              `json:"live_soroban_state_size_window_sample_size"`
	EvictionScanSize                     uint64              `json:"eviction_scan_size"`
	StartingEvictionScanLevel            uint32              `json:"starting_eviction_scan_level"`
	LedgerMaxTxCount                     uint32              `json:"ledger_max_tx_count"`
	LiveSorobanStateSizeWindow           []uint64            `json:"live_soroban_state_size_window"`
	LastModifiedLedger                   uint32              `json:"last_modified_ledger"`
	LedgerEntryChange                    uint32              `json:"ledger_entry_change"`
	Deleted                              bool                `json:"deleted"`
	ClosedAt                             time.Time           `json:"closed_at"`
	LedgerSequence                       uint32              `json:"ledger_sequence"`
}

// TransformConfigSetting converts an config setting ledger change entry into a form suitable for BigQuery
func TransformConfigSetting(ledgerChange ingest.Change, header xdr.LedgerHeaderHistoryEntry) (ConfigSettingOutput, error) {
	ledgerEntry, changeType, outputDeleted, err := utils.ExtractEntryFromChange(ledgerChange)
	if err != nil {
		return ConfigSettingOutput{}, err
	}

	configSetting, ok := ledgerEntry.Data.GetConfigSetting()
	if !ok {
		return ConfigSettingOutput{}, fmt.Errorf("could not extract config setting from ledger entry; actual type is %s", ledgerEntry.Data.Type)
	}

	configSettingId := configSetting.ConfigSettingId

	contractMaxSizeBytes, _ := configSetting.GetContractMaxSizeBytes()

	contractCompute, _ := configSetting.GetContractCompute()
	ledgerMaxInstructions := contractCompute.LedgerMaxInstructions
	txMaxInstructions := contractCompute.TxMaxInstructions
	feeRatePerInstructionsIncrement := contractCompute.FeeRatePerInstructionsIncrement
	txMemoryLimit := contractCompute.TxMemoryLimit

	contractLedgerCost, _ := configSetting.GetContractLedgerCost()
	ledgerMaxDiskReadEntries := contractLedgerCost.LedgerMaxDiskReadEntries
	ledgerMaxDiskReadBytes := contractLedgerCost.LedgerMaxDiskReadBytes
	ledgerMaxWriteLedgerEntries := contractLedgerCost.LedgerMaxWriteLedgerEntries
	ledgerMaxWriteBytes := contractLedgerCost.LedgerMaxWriteBytes
	txMaxDiskReadEntries := contractLedgerCost.TxMaxDiskReadEntries
	txMaxDiskReadBytes := contractLedgerCost.TxMaxDiskReadBytes
	txMaxWriteLedgerEntries := contractLedgerCost.TxMaxWriteLedgerEntries
	txMaxWriteBytes := contractLedgerCost.TxMaxWriteBytes
	feeDiskReadLedgerEntry := contractLedgerCost.FeeDiskReadLedgerEntry
	feeWriteLedgerEntry := contractLedgerCost.FeeWriteLedgerEntry
	feeDiskRead1Kb := contractLedgerCost.FeeDiskRead1Kb
	sorobanStateTargetSizeBytes := contractLedgerCost.SorobanStateTargetSizeBytes
	rentFee1KbSorobanStateSizeLow := contractLedgerCost.RentFee1KbSorobanStateSizeLow
	rentFee1KbSorobanStateSizeHigh := contractLedgerCost.RentFee1KbSorobanStateSizeHigh
	sorobanStateRentFeeGrowthFactor := contractLedgerCost.SorobanStateRentFeeGrowthFactor

	contractHistoricalData, _ := configSetting.GetContractHistoricalData()
	feeHistorical1Kb := contractHistoricalData.FeeHistorical1Kb

	contractMetaData, _ := configSetting.GetContractEvents()
	txMaxContractEventsSizeBytes := contractMetaData.TxMaxContractEventsSizeBytes
	feeContractEvents1Kb := contractMetaData.FeeContractEvents1Kb

	contractBandwidth, _ := configSetting.GetContractBandwidth()
	ledgerMaxTxsSizeBytes := contractBandwidth.LedgerMaxTxsSizeBytes
	txMaxSizeBytes := contractBandwidth.TxMaxSizeBytes
	feeTxSize1Kb := contractBandwidth.FeeTxSize1Kb

	paramsCpuInsns, _ := configSetting.GetContractCostParamsCpuInsns()
	contractCostParamsCpuInsns := serializeParams(paramsCpuInsns)

	paramsMemBytes, _ := configSetting.GetContractCostParamsMemBytes()
	contractCostParamsMemBytes := serializeParams(paramsMemBytes)

	contractDataKeySizeBytes, _ := configSetting.GetContractDataKeySizeBytes()

	contractDataEntrySizeBytes, _ := configSetting.GetContractDataEntrySizeBytes()

	stateArchivalSettings, _ := configSetting.GetStateArchivalSettings()
	maxEntryTtl := stateArchivalSettings.MaxEntryTtl
	minTemporaryTtl := stateArchivalSettings.MinTemporaryTtl
	minPersistentTtl := stateArchivalSettings.MinPersistentTtl
	persistentRentRateDenominator := stateArchivalSettings.PersistentRentRateDenominator
	tempRentRateDenominator := stateArchivalSettings.TempRentRateDenominator
	maxEntriesToArchive := stateArchivalSettings.MaxEntriesToArchive
	liveSorobanStateSizeWindowSampleSize := stateArchivalSettings.LiveSorobanStateSizeWindowSampleSize
	evictionScanSize := stateArchivalSettings.EvictionScanSize
	startingEvictionScanLevel := stateArchivalSettings.StartingEvictionScanLevel

	contractExecutionLanes, _ := configSetting.GetContractExecutionLanes()
	ledgerMaxTxCount := contractExecutionLanes.LedgerMaxTxCount

	sizeWindowsXDR, _ := configSetting.GetLiveSorobanStateSizeWindow()
	sizeWindows := make([]uint64, 0, len(sizeWindowsXDR))
	for _, sizeWindow := range sizeWindowsXDR {
		sizeWindows = append(sizeWindows, uint64(sizeWindow))
	}

	closedAt, err := utils.TimePointToUTCTimeStamp(header.Header.ScpValue.CloseTime)
	if err != nil {
		return ConfigSettingOutput{}, err
	}

	ledgerSequence := header.Header.LedgerSeq

	transformedConfigSetting := ConfigSettingOutput{
		ConfigSettingId:                      int32(configSettingId),
		ContractMaxSizeBytes:                 uint32(contractMaxSizeBytes),
		LedgerMaxInstructions:                int64(ledgerMaxInstructions),
		TxMaxInstructions:                    int64(txMaxInstructions),
		FeeRatePerInstructionsIncrement:      int64(feeRatePerInstructionsIncrement),
		TxMemoryLimit:                        uint32(txMemoryLimit),
		LedgerMaxDiskReadEntries:             uint32(ledgerMaxDiskReadEntries),
		LedgerMaxDiskReadBytes:               uint32(ledgerMaxDiskReadBytes),
		LedgerMaxWriteLedgerEntries:          uint32(ledgerMaxWriteLedgerEntries),
		LedgerMaxWriteBytes:                  uint32(ledgerMaxWriteBytes),
		TxMaxDiskReadEntries:                 uint32(txMaxDiskReadEntries),
		TxMaxDiskReadBytes:                   uint32(txMaxDiskReadBytes),
		TxMaxWriteLedgerEntries:              uint32(txMaxWriteLedgerEntries),
		TxMaxWriteBytes:                      uint32(txMaxWriteBytes),
		FeeDiskReadLedgerEntry:               int64(feeDiskReadLedgerEntry),
		FeeWriteLedgerEntry:                  int64(feeWriteLedgerEntry),
		FeeDiskRead1Kb:                       int64(feeDiskRead1Kb),
		SorobanStateTargetSizeBytes:          int64(sorobanStateTargetSizeBytes),
		RentFee1KbSorobanStateSizeLow:        int64(rentFee1KbSorobanStateSizeLow),
		RentFee1KbSorobanStateSizeHigh:       int64(rentFee1KbSorobanStateSizeHigh),
		SorobanStateRentFeeGrowthFactor:      uint32(sorobanStateRentFeeGrowthFactor),
		FeeHistorical1Kb:                     int64(feeHistorical1Kb),
		TxMaxContractEventsSizeBytes:         uint32(txMaxContractEventsSizeBytes),
		FeeContractEvents1Kb:                 int64(feeContractEvents1Kb),
		LedgerMaxTxsSizeBytes:                uint32(ledgerMaxTxsSizeBytes),
		TxMaxSizeBytes:                       uint32(txMaxSizeBytes),
		FeeTxSize1Kb:                         int64(feeTxSize1Kb),
		ContractCostParamsCpuInsns:           contractCostParamsCpuInsns,
		ContractCostParamsMemBytes:           contractCostParamsMemBytes,
		ContractDataKeySizeBytes:             uint32(contractDataKeySizeBytes),
		ContractDataEntrySizeBytes:           uint32(contractDataEntrySizeBytes),
		MaxEntryTtl:                          uint32(maxEntryTtl),
		MinTemporaryTtl:                      uint32(minTemporaryTtl),
		MinPersistentTtl:                     uint32(minPersistentTtl),
		PersistentRentRateDenominator:        int64(persistentRentRateDenominator),
		TempRentRateDenominator:              int64(tempRentRateDenominator),
		MaxEntriesToArchive:                  uint32(maxEntriesToArchive),
		LiveSorobanStateSizeWindowSampleSize: uint32(liveSorobanStateSizeWindowSampleSize),
		EvictionScanSize:                     uint64(evictionScanSize),
		StartingEvictionScanLevel:            uint32(startingEvictionScanLevel),
		LedgerMaxTxCount:                     uint32(ledgerMaxTxCount),
		LiveSorobanStateSizeWindow:           sizeWindows,
		LastModifiedLedger:                   uint32(ledgerEntry.LastModifiedLedgerSeq),
		LedgerEntryChange:                    uint32(changeType),
		Deleted:                              outputDeleted,
		ClosedAt:                             closedAt,
		LedgerSequence:                       uint32(ledgerSequence),
	}
	return transformedConfigSetting, nil
}

func serializeParams(costParams xdr.ContractCostParams) []map[string]string {
	params := make([]map[string]string, 0, len(costParams))
	for _, contractCostParam := range costParams {
		serializedParam := map[string]string{}
		serializedParam["ExtV"] = strconv.Itoa(int(contractCostParam.Ext.V))
		serializedParam["ConstTerm"] = strconv.Itoa(int(contractCostParam.ConstTerm))
		serializedParam["LinearTerm"] = strconv.Itoa(int(contractCostParam.LinearTerm))
		params = append(params, serializedParam)
	}

	return params
}
