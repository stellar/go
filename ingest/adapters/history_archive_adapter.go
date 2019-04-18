package ingestadapters

import (
	"github.com/stellar/go/support/historyarchive"
)

// HistoryArchiveAdapter is an adapter for the historyarchive package to read from history archives
type HistoryArchiveAdapter struct {
	archive historyarchive.Archive
}

// GetLatestLedgerSequence returns the latest ledger sequence or an error
func (haa *HistoryArchiveAdapter) GetLatestLedgerSequence() (uint32, error) {
	return nil, nil
}

// GetState returns a reader with the state of the ledger at the provided sequence number
func (haa *HistoryArchiveAdapter) GetState(sequence uint32) (io.StateReader, error) {
	return nil, nil
}

// GetLedger gets a ledger transaction result at the provided sequence number
func (haa *HistoryArchiveAdapter) GetLedger(sequence uint32) (io.ArchiveLedgerReader, error) {
	return nil, nil
}
