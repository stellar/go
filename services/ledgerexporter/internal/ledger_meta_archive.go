package ledgerexporter

import (
	"github.com/stellar/go/support/compressxdr"
	"github.com/stellar/go/support/datastore"
	"github.com/stellar/go/xdr"
)

// LedgerMetaArchive represents a file with metadata and binary data.
type LedgerMetaArchive struct {
	ObjectKey string
	Data      xdr.LedgerCloseMetaBatch
	metaData  datastore.MetaData
}

// NewLedgerMetaArchiveFromXDR creates a new LedgerMetaArchive instance.
func NewLedgerMetaArchiveFromXDR(config *Config, key string, data xdr.LedgerCloseMetaBatch) (*LedgerMetaArchive, error) {
	startLedger, err := data.GetLedger(uint32(data.StartSequence))
	if err != nil {
		return &LedgerMetaArchive{}, err

	}
	endLedger, err := data.GetLedger(uint32(data.EndSequence))
	if err != nil {
		return &LedgerMetaArchive{}, err
	}

	return &LedgerMetaArchive{
		ObjectKey: key,
		Data:      data,
		metaData: datastore.MetaData{
			StartLedger:          startLedger.LedgerSequence(),
			EndLedger:            endLedger.LedgerSequence(),
			StartLedgerCloseTime: startLedger.LedgerCloseTime(),
			EndLedgerCloseTime:   endLedger.LedgerCloseTime(),
			Network:              config.Network,
			CompressionType:      compressxdr.DefaultCompressor.Name(),
			ProtocolVersion:      endLedger.ProtocolVersion(),
			CoreVersion:          config.CoreVersion,
			Version:              version,
		},
	}, nil
}
