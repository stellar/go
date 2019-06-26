package processors

import (
	"github.com/stellar/go/xdr"
)

// CSVPrinter prints ledger entries to a file or stdout (when Filename is empty).
// Can be used both for processing state and ledgers.
type CSVPrinter struct {
	noStateProcessor

	Filename string
}

// StatusLogger prints number of processed entries every N
// entries.
type StatusLogger struct {
	noStateProcessor

	N int
}

// EntryTypeFilter is a pipeline.StateProcessor that filters out all
// entries that are not of type `Type`.
type EntryTypeFilter struct {
	concurrentProcessor
	noStateProcessor

	Type xdr.LedgerEntryType
}

type concurrentProcessor struct{}

func (n *concurrentProcessor) IsConcurrent() bool {
	return true
}

type noStateProcessor struct{}

func (n *noStateProcessor) Reset() {
	// No internal state
}
