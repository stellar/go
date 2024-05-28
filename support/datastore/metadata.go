package datastore

import "strconv"

type MetaData struct {
	StartLedger          uint32
	EndLedger            uint32
	StartLedgerCloseTime int64
	EndLedgerCloseTime   int64
	ProtocolVersion      string
	CoreVersion          string
	Network              string
	CompressionType      string
	Version              string
}

func (m MetaData) ToMap() map[string]string {
	return map[string]string{
		"x-lexie-start-ledger":            strconv.FormatUint(uint64(m.StartLedger), 10),
		"x-lexie-end-ledger":              strconv.FormatUint(uint64(m.EndLedger), 10),
		"x-lexie-start-ledger-close-time": strconv.FormatInt(m.StartLedgerCloseTime, 10),
		"x-lexie-end-ledger-close-time":   strconv.FormatInt(m.EndLedgerCloseTime, 10),
		"x-lexie-protocol-version":        m.ProtocolVersion,
		"x-lexie-core-version":            m.CoreVersion,
		"x-lexie-network":                 m.Network,
		"x-lexie-compression-type":        m.CompressionType,
		"x-lexie-version":                 m.Version,
	}
}
func NewMetaDataFromMap(data map[string]string) (MetaData, error) {
	var metaData MetaData

	if val, ok := data["x-lexie-start-ledger"]; ok {
		startLedger, err := strconv.ParseUint(val, 10, 32)
		if err != nil {
			return metaData, err
		}
		metaData.StartLedger = uint32(startLedger)
	}

	if val, ok := data["x-lexie-end-ledger"]; ok {
		endLedger, err := strconv.ParseUint(val, 10, 32)
		if err != nil {
			return metaData, err
		}
		metaData.EndLedger = uint32(endLedger)
	}

	if val, ok := data["x-lexie-start-ledger-close-time"]; ok {
		startLedgerCloseTime, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return metaData, err
		}
		metaData.StartLedgerCloseTime = startLedgerCloseTime
	}

	if val, ok := data["x-lexie-end-ledger-close-time"]; ok {
		endLedgerCloseTime, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return metaData, err
		}
		metaData.EndLedgerCloseTime = endLedgerCloseTime
	}

	metaData.ProtocolVersion = data["x-lexie-protocol-version"]
	metaData.CoreVersion = data["x-lexie-core-version"]
	metaData.Network = data["x-lexie-network"]
	metaData.CompressionType = data["x-lexie-compression-type"]
	metaData.Version = data["x-lexie-version"]

	return metaData, nil
}
