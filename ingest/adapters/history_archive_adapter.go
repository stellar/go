package ingestadapters

import (
	"fmt"

	"github.com/stellar/go/ingest/io"
	"github.com/stellar/go/support/historyarchive"
)

// HistoryArchiveAdapter is an adapter for the historyarchive package to read from history archives
type HistoryArchiveAdapter struct {
	archive historyarchive.Archive
}

// MakeHistoryArchiveAdapter is a factory method to make a HistoryArchiveAdapter
func MakeHistoryArchiveAdapter(archive historyarchive.Archive) *HistoryArchiveAdapter {
	return &HistoryArchiveAdapter{archive: archive}
}

// GetLatestLedgerSequence returns the latest ledger sequence or an error
func (haa *HistoryArchiveAdapter) GetLatestLedgerSequence() (uint32, error) {
	has, e := haa.archive.GetRootHAS()
	if e != nil {
		return 0, fmt.Errorf("could not get root HAS: %s", e)
	}

	return has.CurrentLedger, nil
}

// GetState returns a reader with the state of the ledger at the provided sequence number
func (haa *HistoryArchiveAdapter) GetState(sequence uint32) (io.StateReader, error) {
	return nil, nil
}

// GetLedger gets a ledger transaction result at the provided sequence number
func (haa *HistoryArchiveAdapter) GetLedger(sequence uint32) (io.ArchiveLedgerReader, error) {
	return nil, nil
}
