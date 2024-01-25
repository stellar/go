package exporter

import (
	"fmt"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LedgerCloseMetaObject represents a file with metadata and binary data.
type LedgerCloseMetaObject struct {
	// file name
	objectKey string
	// Actual binary data
	data xdr.LedgerCloseMetaBatch
}

func NewLedgerCloseMetaObject(key string, startSeq uint32, endSeq uint32) *LedgerCloseMetaObject {
	return &LedgerCloseMetaObject{
		objectKey: key,
		data: xdr.LedgerCloseMetaBatch{
			StartSequence: xdr.Uint32(startSeq),
			EndSequence:   xdr.Uint32(endSeq),
		},
	}
}

func (f *LedgerCloseMetaObject) GetLastLedgerCloseMetaSequence() (uint32, error) {
	if len(f.data.LedgerCloseMetas) == 0 {
		return 0, errors.New("LedgerCloseMetas is empty")
	}

	return f.data.LedgerCloseMetas[len(f.data.LedgerCloseMetas)-1].LedgerSequence(), nil
}

// AddLedgerCloseMeta adds a ledger
func (f *LedgerCloseMetaObject) AddLedgerCloseMeta(ledgerCloseMeta xdr.LedgerCloseMeta) error {
	lastSequence, err := f.GetLastLedgerCloseMetaSequence()
	if err == nil {
		if ledgerCloseMeta.LedgerSequence() != lastSequence+1 {
			return fmt.Errorf("ledgers must be added sequentially. Sequence number: %d, "+
				"expected sequence number: %d", ledgerCloseMeta.LedgerSequence(), lastSequence+1)
		}
	}

	f.data.LedgerCloseMetas = append(f.data.LedgerCloseMetas, ledgerCloseMeta)
	return nil
}

// LedgerCount returns the number of ledgers added so far
func (f *LedgerCloseMetaObject) LedgerCount() uint32 {
	return uint32(len(f.data.LedgerCloseMetas))
}

func (f *LedgerCloseMetaObject) GetData() ([]byte, error) {
	return f.data.MarshalBinary()
}

func (f *LedgerCloseMetaObject) GetCompressedData() ([]byte, error) {
	blob, err := f.GetData()
	if err != nil {
		return nil, err
	}
	return Compress(blob)
}
